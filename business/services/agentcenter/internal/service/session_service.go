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

// Redis Key 前缀
const (
	PrefixAgentSession = "vocara:agent:session:" // HASH  {tenantId}:{agentId}
	PrefixAgentOnline  = "vocara:agent:online:"  // SET   {tenantId}
	PrefixAgentIdle    = "vocara:agent:idle:"    // ZSET  {skillGroupId}   score=lastReadyTs
	PrefixAgentSkills  = "vocara:agent:skills:"  // SET   {agentId}
	PrefixAgentLock    = "vocara:agent:lock:"    // STRING {agentId} 分布式锁
)

// SessionService 座席会话管理（基于 Redis）
type SessionService struct {
	redis      redis.UniversalClient
	sessionTTL time.Duration
}

// NewSessionService 创建会话服务
func NewSessionService(rdb redis.UniversalClient, sessionTTL int) *SessionService {
	if sessionTTL <= 0 {
		sessionTTL = 120 // 默认 120 秒
	}
	return &SessionService{
		redis:      rdb,
		sessionTTL: time.Duration(sessionTTL) * time.Second,
	}
}

// sessionKey 座席会话 key
func (s *SessionService) sessionKey(tenantID, agentID string) string {
	return PrefixAgentSession + tenantID + ":" + agentID
}

// SessionKey 导出的会话 key 计算方法，仅供测试代码断言真实键格式使用，业务代码请走内部 sessionKey。
func (s *SessionService) SessionKey(tenantID, agentID string) string {
	return s.sessionKey(tenantID, agentID)
}

// onlineKey 在线座席集合 key
func (s *SessionService) onlineKey(tenantID string) string {
	return PrefixAgentOnline + tenantID
}

// idleKey 空闲座席 ZSET key
func (s *SessionService) idleKey(skillGroupID string) string {
	return PrefixAgentIdle + skillGroupID
}

// skillsKey 座席技能组 SET key
func (s *SessionService) skillsKey(agentID string) string {
	return PrefixAgentSkills + agentID
}

// lockKey 分布式锁 key
func (s *SessionService) lockKey(agentID string) string {
	return PrefixAgentLock + agentID
}

// GetSession 获取座席会话
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

// SetSession 设置座席会话（带过期时间）
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

	// 设置过期时间
	return s.redis.Expire(ctx, key, s.sessionTTL).Err()
}

// UpdateSessionField 更新会话单个字段
func (s *SessionService) UpdateSessionField(ctx context.Context, tenantID, agentID, field string, value interface{}) error {
	key := s.sessionKey(tenantID, agentID)
	err := s.redis.HSet(ctx, key, field, value).Err()
	if err != nil {
		return fmt.Errorf("update session field failed: %w", err)
	}
	// 续期
	return s.redis.Expire(ctx, key, s.sessionTTL).Err()
}

// UpdateStatus 更新座席状态
func (s *SessionService) UpdateStatus(ctx context.Context, tenantID, agentID string, status model.AgentStatus) error {
	return s.UpdateSessionField(ctx, tenantID, agentID, "status", int(status))
}

// DeleteSession 删除会话（座席离线）
func (s *SessionService) DeleteSession(ctx context.Context, tenantID, agentID string) error {
	key := s.sessionKey(tenantID, agentID)
	return s.redis.Del(ctx, key).Err()
}

// RefreshTTL 续期会话
func (s *SessionService) RefreshTTL(ctx context.Context, tenantID, agentID string) error {
	key := s.sessionKey(tenantID, agentID)

	// 同时更新 lastActiveTime
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

// SessionExists 会话是否存在
func (s *SessionService) SessionExists(ctx context.Context, tenantID, agentID string) (bool, error) {
	key := s.sessionKey(tenantID, agentID)
	n, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check session exists failed: %w", err)
	}
	return n > 0, nil
}

// ---------- 在线座席集合 ----------

// AddOnline 添加到在线集合
func (s *SessionService) AddOnline(ctx context.Context, tenantID, agentID string) error {
	key := s.onlineKey(tenantID)
	return s.redis.SAdd(ctx, key, agentID).Err()
}

// RemoveOnline 从在线集合移除
func (s *SessionService) RemoveOnline(ctx context.Context, tenantID, agentID string) error {
	key := s.onlineKey(tenantID)
	return s.redis.SRem(ctx, key, agentID).Err()
}

// IsOnline 是否在线
func (s *SessionService) IsOnline(ctx context.Context, tenantID, agentID string) (bool, error) {
	key := s.onlineKey(tenantID)
	return s.redis.SIsMember(ctx, key, agentID).Result()
}

// GetOnlineCount 在线座席数量
func (s *SessionService) GetOnlineCount(ctx context.Context, tenantID string) (int64, error) {
	key := s.onlineKey(tenantID)
	return s.redis.SCard(ctx, key).Result()
}

// GetOnlineList 获取在线座席列表
func (s *SessionService) GetOnlineList(ctx context.Context, tenantID string) ([]string, error) {
	key := s.onlineKey(tenantID)
	return s.redis.SMembers(ctx, key).Result()
}

// ---------- 空闲座席 ZSET ----------

// AddIdle 添加到空闲集合（每个技能组一个 ZSET）
func (s *SessionService) AddIdle(ctx context.Context, skillGroupID, agentID string, score int64) error {
	key := s.idleKey(skillGroupID)
	return s.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(score),
		Member: agentID,
	}).Err()
}

// AddIdleBatch 批量添加到空闲集合
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

// RemoveIdle 从空闲集合移除
func (s *SessionService) RemoveIdle(ctx context.Context, skillGroupID, agentID string) error {
	key := s.idleKey(skillGroupID)
	return s.redis.ZRem(ctx, key, agentID).Err()
}

// RemoveIdleBatch 批量从空闲集合移除
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

// PopIdle 弹出一个空闲座席（最长空闲优先，score 小的先出）
func (s *SessionService) PopIdle(ctx context.Context, skillGroupID string) (string, int64, error) {
	key := s.idleKey(skillGroupID)
	// ZPOPMIN: 弹出 score 最小的（最早就绪的）
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

// GetIdleCount 获取空闲座席数量
func (s *SessionService) GetIdleCount(ctx context.Context, skillGroupID string) (int64, error) {
	key := s.idleKey(skillGroupID)
	return s.redis.ZCard(ctx, key).Result()
}

// globalIdleUnionPrefix 跨技能组并集计算使用的临时键前缀
const globalIdleUnionPrefix = PrefixAgentIdle + "union:"

// GetGlobalIdleCount 返回跨技能组去重后的就绪座席总数。
// 同一名座席可属于多个技能组、出现在多个空闲 ZSET 中，
// 故以 ZUNIONSTORE 对全部空闲集合求并集后再计数，避免重复统计。
func (s *SessionService) GetGlobalIdleCount(ctx context.Context) (int64, error) {
	var keys []string
	iter := s.redis.ScanType(ctx, 0, PrefixAgentIdle+"*", 100, "zset").Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// 跳过并集计算遗留的临时键
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

// GetIdleList 获取空闲座席列表
func (s *SessionService) GetIdleList(ctx context.Context, skillGroupID string, start, stop int64) ([]string, error) {
	key := s.idleKey(skillGroupID)
	return s.redis.ZRange(ctx, key, start, stop).Result()
}

// ---------- 座席技能组映射 ----------

// SetAgentSkills 设置座席技能组
func (s *SessionService) SetAgentSkills(ctx context.Context, agentID string, skillIDs []string) error {
	key := s.skillsKey(agentID)
	// 先清空再设置
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

// GetAgentSkills 获取座席技能组
func (s *SessionService) GetAgentSkills(ctx context.Context, agentID string) ([]string, error) {
	key := s.skillsKey(agentID)
	return s.redis.SMembers(ctx, key).Result()
}

// DeleteAgentSkills 删除座席技能组缓存
func (s *SessionService) DeleteAgentSkills(ctx context.Context, agentID string) error {
	key := s.skillsKey(agentID)
	return s.redis.Del(ctx, key).Err()
}

// ---------- 分布式锁 ----------

// TryLock 尝试获取座席分布式锁
func (s *SessionService) TryLock(ctx context.Context, agentID string, ttl time.Duration) (bool, string, error) {
	key := s.lockKey(agentID)
	value := fmt.Sprintf("lock-%d-%s", time.Now().UnixNano(), agentID)

	ok, err := s.redis.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, "", fmt.Errorf("try lock failed: %w", err)
	}
	return ok, value, nil
}

// Unlock 释放分布式锁
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

// ---------- 过期扫描 ----------

// ScanExpiredSessions 扫描过期会话，返回过期的座席列表
// 扫描在线集合中不存在 session 的座席
func (s *SessionService) ScanExpiredSessions(ctx context.Context, tenantID string) ([]string, error) {
	onlineKey := s.onlineKey(tenantID)

	// 获取所有在线座席
	agents, err := s.redis.SMembers(ctx, onlineKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get online agents failed: %w", err)
	}

	if len(agents) == 0 {
		return nil, nil
	}

	// 批量检查 session 是否存在（使用 pipeline）
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

// DebugSession 打印调试信息
func (s *SessionService) DebugSession(ctx context.Context, tenantID, agentID string) {
	session, _ := s.GetSession(ctx, tenantID, agentID)
	data, _ := json.Marshal(session)
	logger.Debug("session debug",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("session", string(data)),
	)
}
