package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vocara/pkg/logger"
	"vocara/services/agentcenter/internal/model"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redis key prefixes.
const (
	PrefixAgentSession = "vocara:agent:session:" // HASH  {tenantId}:{agentId}
	PrefixAgentOnline  = "vocara:agent:online:"  // SET   {tenantId}
	PrefixAgentIdle    = "vocara:agent:idle:"    // ZSET  {skillGroupId}   score=lastReadyTs
	PrefixAgentSkills  = "vocara:agent:skills:"  // SET   {agentId}
	PrefixAgentLock    = "vocara:agent:lock:"    // STRING {agentId} distributed lock
)

// SessionService manages agent sessions in Redis.
type SessionService struct {
	redis      redis.UniversalClient
	sessionTTL time.Duration
}

// NewSessionService creates a session service.
func NewSessionService(rdb redis.UniversalClient, sessionTTL int) *SessionService {
	if sessionTTL <= 0 {
		sessionTTL = 120 // default to 120 seconds
	}
	return &SessionService{
		redis:      rdb,
		sessionTTL: time.Duration(sessionTTL) * time.Second,
	}
}

// sessionKey returns the agent session key.
func (s *SessionService) sessionKey(tenantID, agentID string) string {
	return PrefixAgentSession + tenantID + ":" + agentID
}

// SessionKey exposes the session key calculation for tests that assert the actual key format; business code should use sessionKey.
func (s *SessionService) SessionKey(tenantID, agentID string) string {
	return s.sessionKey(tenantID, agentID)
}

// onlineKey returns the online agent set key.
func (s *SessionService) onlineKey(tenantID string) string {
	return PrefixAgentOnline + tenantID
}

// idleKey returns the idle agent ZSET key.
func (s *SessionService) idleKey(skillGroupID string) string {
	return PrefixAgentIdle + skillGroupID
}

// skillsKey returns the agent skill group SET key.
func (s *SessionService) skillsKey(agentID string) string {
	return PrefixAgentSkills + agentID
}

// lockKey returns the distributed lock key.
func (s *SessionService) lockKey(agentID string) string {
	return PrefixAgentLock + agentID
}

// GetSession returns an agent session.
func (s *SessionService) GetSession(ctx context.Context, tenantID, agentID string) (*model.AgentSession, error) {
	key := s.sessionKey(tenantID, agentID)
	data, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get session failed: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	session := &model.AgentSession{}
	if v, ok := data["agentId"]; ok {
		session.AgentID = v
	}
	if v, ok := data["agentName"]; ok {
		session.AgentName = v
	}
	if v, ok := data["tenantId"]; ok {
		session.TenantID = v
	}
	if v, ok := data["status"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			session.Status = n
		}
	}
	if v, ok := data["dnNumber"]; ok {
		session.DnNumber = v
	}
	if v, ok := data["skillGroups"]; ok && v != "" {
		session.SkillGroups = strings.Split(v, ",")
	}
	if v, ok := data["currentCallId"]; ok {
		session.CurrentCallID = v
	}
	if v, ok := data["lastReadyTime"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			session.LastReadyTime = n
		}
	}
	if v, ok := data["lastActiveTime"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			session.LastActiveTime = n
		}
	}
	if v, ok := data["checkinTime"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			session.CheckinTime = n
		}
	}
	if v, ok := data["ip"]; ok {
		session.IP = v
	}

	return session, nil
}

// SetSession stores an agent session with an expiration time.
func (s *SessionService) SetSession(ctx context.Context, session *model.AgentSession) error {
	key := s.sessionKey(session.TenantID, session.AgentID)

	now := time.Now().Unix()
	session.LastActiveTime = now

	skillGroupsStr := strings.Join(session.SkillGroups, ",")

	err := s.redis.HSet(ctx, key, map[string]interface{}{
		"agentId":        session.AgentID,
		"agentName":      session.AgentName,
		"tenantId":       session.TenantID,
		"status":         session.Status,
		"dnNumber":       session.DnNumber,
		"skillGroups":    skillGroupsStr,
		"currentCallId":  session.CurrentCallID,
		"lastReadyTime":  session.LastReadyTime,
		"lastActiveTime": session.LastActiveTime,
		"checkinTime":    session.CheckinTime,
		"ip":             session.IP,
	}).Err()
	if err != nil {
		return fmt.Errorf("set session failed: %w", err)
	}

	// Set the expiration time.
	return s.redis.Expire(ctx, key, s.sessionTTL).Err()
}

// UpdateSessionField updates a single session field.
func (s *SessionService) UpdateSessionField(ctx context.Context, tenantID, agentID, field string, value interface{}) error {
	key := s.sessionKey(tenantID, agentID)
	err := s.redis.HSet(ctx, key, field, value).Err()
	if err != nil {
		return fmt.Errorf("update session field failed: %w", err)
	}
	// Renew the TTL.
	return s.redis.Expire(ctx, key, s.sessionTTL).Err()
}

// UpdateStatus updates the agent status.
func (s *SessionService) UpdateStatus(ctx context.Context, tenantID, agentID string, status model.AgentStatus) error {
	return s.UpdateSessionField(ctx, tenantID, agentID, "status", int(status))
}

// DeleteSession deletes a session when an agent goes offline.
func (s *SessionService) DeleteSession(ctx context.Context, tenantID, agentID string) error {
	key := s.sessionKey(tenantID, agentID)
	return s.redis.Del(ctx, key).Err()
}

// RefreshTTL renews a session.
func (s *SessionService) RefreshTTL(ctx context.Context, tenantID, agentID string) error {
	key := s.sessionKey(tenantID, agentID)

	// Also update lastActiveTime.
	now := time.Now().Unix()
	pipe := s.redis.TxPipeline()
	pipe.HSet(ctx, key, "lastActiveTime", now)
	pipe.Expire(ctx, key, s.sessionTTL)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("refresh ttl failed: %w", err)
	}
	return nil
}

// SessionExists reports whether a session exists.
func (s *SessionService) SessionExists(ctx context.Context, tenantID, agentID string) (bool, error) {
	key := s.sessionKey(tenantID, agentID)
	n, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check session exists failed: %w", err)
	}
	return n > 0, nil
}

// ---------- Online agent set ----------

// AddOnline adds an agent to the online set.
func (s *SessionService) AddOnline(ctx context.Context, tenantID, agentID string) error {
	key := s.onlineKey(tenantID)
	return s.redis.SAdd(ctx, key, agentID).Err()
}

// RemoveOnline removes an agent from the online set.
func (s *SessionService) RemoveOnline(ctx context.Context, tenantID, agentID string) error {
	key := s.onlineKey(tenantID)
	return s.redis.SRem(ctx, key, agentID).Err()
}

// IsOnline reports whether an agent is online.
func (s *SessionService) IsOnline(ctx context.Context, tenantID, agentID string) (bool, error) {
	key := s.onlineKey(tenantID)
	return s.redis.SIsMember(ctx, key, agentID).Result()
}

// GetOnlineCount returns the number of online agents.
func (s *SessionService) GetOnlineCount(ctx context.Context, tenantID string) (int64, error) {
	key := s.onlineKey(tenantID)
	return s.redis.SCard(ctx, key).Result()
}

// GetOnlineList returns the online agents.
func (s *SessionService) GetOnlineList(ctx context.Context, tenantID string) ([]string, error) {
	key := s.onlineKey(tenantID)
	return s.redis.SMembers(ctx, key).Result()
}

// ---------- Idle agent ZSET ----------

// AddIdle adds an agent to the idle set, with one ZSET per skill group.
func (s *SessionService) AddIdle(ctx context.Context, skillGroupID, agentID string, score int64) error {
	key := s.idleKey(skillGroupID)
	return s.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(score),
		Member: agentID,
	}).Err()
}

// AddIdleBatch adds an agent to multiple idle sets.
func (s *SessionService) AddIdleBatch(ctx context.Context, agentID string, skillGroupIDs []string, score int64) error {
	if len(skillGroupIDs) == 0 {
		return nil
	}
	pipe := s.redis.TxPipeline()
	for _, sgID := range skillGroupIDs {
		key := s.idleKey(sgID)
		pipe.ZAdd(ctx, key, redis.Z{
			Score:  float64(score),
			Member: agentID,
		})
	}
	_, err := pipe.Exec(ctx)
	return err
}

// RemoveIdle removes an agent from the idle set.
func (s *SessionService) RemoveIdle(ctx context.Context, skillGroupID, agentID string) error {
	key := s.idleKey(skillGroupID)
	return s.redis.ZRem(ctx, key, agentID).Err()
}

// RemoveIdleBatch removes an agent from multiple idle sets.
func (s *SessionService) RemoveIdleBatch(ctx context.Context, agentID string, skillGroupIDs []string) error {
	if len(skillGroupIDs) == 0 {
		return nil
	}
	pipe := s.redis.TxPipeline()
	for _, sgID := range skillGroupIDs {
		key := s.idleKey(sgID)
		pipe.ZRem(ctx, key, agentID)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// PopIdle pops an idle agent, preferring the longest-idle agent with the lowest score.
func (s *SessionService) PopIdle(ctx context.Context, skillGroupID string) (string, int64, error) {
	key := s.idleKey(skillGroupID)
	// ZPOPMIN pops the lowest score, representing the agent that became ready earliest.
	result, err := s.redis.ZPopMin(ctx, key, 1).Result()
	if err != nil {
		if err == redis.Nil {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("pop idle agent failed: %w", err)
	}
	if len(result) == 0 {
		return "", 0, nil
	}
	agentID, ok := result[0].Member.(string)
	if !ok {
		return "", 0, fmt.Errorf("invalid idle member type")
	}
	return agentID, int64(result[0].Score), nil
}

// GetIdleCount returns the number of idle agents.
func (s *SessionService) GetIdleCount(ctx context.Context, skillGroupID string) (int64, error) {
	key := s.idleKey(skillGroupID)
	return s.redis.ZCard(ctx, key).Result()
}

// globalIdleUnionPrefix is the temporary key prefix used to compute the union across skill groups.
const globalIdleUnionPrefix = PrefixAgentIdle + "union:"

// GetGlobalIdleCount returns the total number of ready agents deduplicated across skill groups.
// An agent can belong to multiple skill groups and appear in multiple idle ZSETs,
// so ZUNIONSTORE combines all idle sets before counting to avoid duplicates.
func (s *SessionService) GetGlobalIdleCount(ctx context.Context) (int64, error) {
	var keys []string
	iter := s.redis.ScanType(ctx, 0, PrefixAgentIdle+"*", 100, "zset").Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// Skip temporary keys left by union calculations.
		if !strings.HasPrefix(key, globalIdleUnionPrefix) {
			keys = append(keys, key)
		}
	}
	if err := iter.Err(); err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		return 0, nil
	}

	unionKey := fmt.Sprintf("%s%d", globalIdleUnionPrefix, time.Now().UnixNano())
	defer func() { _ = s.redis.Del(context.Background(), unionKey) }()

	if err := s.redis.ZUnionStore(ctx, unionKey, &redis.ZStore{
		Keys:      keys,
		Aggregate: "SUM",
	}).Err(); err != nil {
		return 0, err
	}
	return s.redis.ZCard(ctx, unionKey).Result()
}

// GetIdleList returns the idle agents in a skill group.
func (s *SessionService) GetIdleList(ctx context.Context, skillGroupID string, start, stop int64) ([]string, error) {
	key := s.idleKey(skillGroupID)
	return s.redis.ZRange(ctx, key, start, stop).Result()
}

// ---------- Agent skill group mappings ----------

// SetAgentSkills sets an agent's skill groups.
func (s *SessionService) SetAgentSkills(ctx context.Context, agentID string, skillIDs []string) error {
	key := s.skillsKey(agentID)
	// Clear the existing set before writing the new values.
	pipe := s.redis.TxPipeline()
	pipe.Del(ctx, key)
	if len(skillIDs) > 0 {
		members := make([]interface{}, len(skillIDs))
		for i, id := range skillIDs {
			members[i] = id
		}
		pipe.SAdd(ctx, key, members...)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// GetAgentSkills returns an agent's skill groups.
func (s *SessionService) GetAgentSkills(ctx context.Context, agentID string) ([]string, error) {
	key := s.skillsKey(agentID)
	return s.redis.SMembers(ctx, key).Result()
}

// DeleteAgentSkills deletes an agent's skill group cache.
func (s *SessionService) DeleteAgentSkills(ctx context.Context, agentID string) error {
	key := s.skillsKey(agentID)
	return s.redis.Del(ctx, key).Err()
}

// ---------- Distributed lock ----------

// TryLock attempts to acquire an agent's distributed lock.
func (s *SessionService) TryLock(ctx context.Context, agentID string, ttl time.Duration) (bool, string, error) {
	key := s.lockKey(agentID)
	value := fmt.Sprintf("lock-%d-%s", time.Now().UnixNano(), agentID)

	ok, err := s.redis.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, "", fmt.Errorf("try lock failed: %w", err)
	}
	return ok, value, nil
}

// Unlock releases a distributed lock.
func (s *SessionService) Unlock(ctx context.Context, agentID, lockValue string) (bool, error) {
	key := s.lockKey(agentID)
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	result, err := s.redis.Eval(ctx, script, []string{key}, lockValue).Result()
	if err != nil {
		return false, fmt.Errorf("unlock failed: %w", err)
	}
	return result.(int64) == 1, nil
}

// ---------- Expiration scan ----------

// ScanExpiredSessions returns agents with expired sessions.
// It scans the online set for agents whose sessions no longer exist.
func (s *SessionService) ScanExpiredSessions(ctx context.Context, tenantID string) ([]string, error) {
	onlineKey := s.onlineKey(tenantID)

	// Get all online agents.
	agents, err := s.redis.SMembers(ctx, onlineKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get online agents failed: %w", err)
	}

	if len(agents) == 0 {
		return nil, nil
	}

	// Check session existence in a pipeline.
	var expired []string
	pipe := s.redis.Pipeline()
	intCmds := make([]*redis.IntCmd, len(agents))
	for i, agentID := range agents {
		key := s.sessionKey(tenantID, agentID)
		intCmds[i] = pipe.Exists(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline exists failed: %w", err)
	}

	for i, cmd := range intCmds {
		if cmd.Val() == 0 {
			expired = append(expired, agents[i])
		}
	}

	return expired, nil
}

// DebugSession logs session debugging information.
func (s *SessionService) DebugSession(ctx context.Context, tenantID, agentID string) {
	session, _ := s.GetSession(ctx, tenantID, agentID)
	data, _ := json.Marshal(session)
	logger.Debug("session debug",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("session", string(data)),
	)
}
