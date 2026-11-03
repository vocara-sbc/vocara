package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"vocara/pkg/errors"
	"vocara/pkg/eventbus"
	"vocara/pkg/license"
	"vocara/pkg/logger"
	"vocara/pkg/utils"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// eventSource is the service name used as the event source.
const eventSource = "agentcenter"

// AgentEventData contains agent event fields matching those used by the push service.
type AgentEventData struct {
	AgentID     int64   `json:"agent_id"`
	TenantID    int64   `json:"tenant_id"`
	AgentName   string  `json:"agent_name"`
	Status      string  `json:"status"`
	StatusText  string  `json:"status_text"`
	PrevStatus  string  `json:"prev_status"`
	Extension   string  `json:"extension"`
	SkillGroups []int64 `json:"skill_groups"`
}

// parseID parses a string ID as int64, returning 0 and logging a warning on failure.
func parseID(idStr, fieldName string) int64 {
	if idStr == "" {
		return 0
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Warn("parse id failed",
			zap.String("field", fieldName),
			zap.String("value", idStr),
			zap.Error(err),
		)
		return 0
	}
	return id
}

// parseSkillGroupIDs converts string skill group IDs to int64 values and skips elements that fail to parse.
func parseSkillGroupIDs(ids []string) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}
	result := make([]int64, 0, len(ids))
	for _, idStr := range ids {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			result = append(result, id)
		} else {
			logger.Warn("parse skill group id failed",
				zap.String("value", idStr),
				zap.Error(err),
			)
		}
	}
	return result
}

// AgentClaims contains the agent JWT payload.
type AgentClaims struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	TenantID  string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// AgentService provides agent operations.
type AgentService struct {
	agentRepo       *repository.AgentRepo
	dnRepo          *repository.DnRepo
	skillRepo       *repository.SkillRepo
	onlineAgentRepo *repository.OnlineAgentRepo
	sessionSvc      *SessionService
	jwtSecret       string
	jwtIssuer       string
	jwtExpireHour   int
	eventPublisher  eventbus.Publisher
	licenseManager  *license.Manager
}

// NewAgentService creates an agent service.
func NewAgentService(
	agentRepo *repository.AgentRepo,
	dnRepo *repository.DnRepo,
	skillRepo *repository.SkillRepo,
	onlineAgentRepo *repository.OnlineAgentRepo,
	sessionSvc *SessionService,
	jwtSecret, jwtIssuer string,
	jwtExpireHour int,
	eventPublisher eventbus.Publisher,
	licenseManager *license.Manager,
) *AgentService {
	return &AgentService{
		agentRepo:       agentRepo,
		dnRepo:          dnRepo,
		skillRepo:       skillRepo,
		onlineAgentRepo: onlineAgentRepo,
		sessionSvc:      sessionSvc,
		jwtSecret:       jwtSecret,
		jwtIssuer:       jwtIssuer,
		jwtExpireHour:   jwtExpireHour,
		eventPublisher:  eventPublisher,
		licenseManager:  licenseManager,
	}
}

// checkReadyLimit validates license capacity before an agent becomes ready; the count excludes this agent because it is not yet in the ready set.
func (svc *AgentService) checkReadyLimit(ctx context.Context) error {
	if svc.licenseManager == nil {
		return nil
	}
	count, err := svc.sessionSvc.GetGlobalIdleCount(ctx)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, "统计就绪座席数失败")
	}
	if !svc.licenseManager.LimitAllowed(license.LimitAgents, int(count)) {
		return errors.NewBizError(errors.CodeLicenseLimitExceeded, errors.MsgLicenseLimitExceeded).
			WithDetail("就绪座席数已达授权上限")
	}
	return nil
}

// publishEvent publishes an event asynchronously without blocking the main flow.
func (svc *AgentService) publishEvent(eventType string, tenantIDStr string, data interface{}) {
	if svc.eventPublisher == nil {
		return
	}

	// Parse tenant_id as uint64 for the event bus.
	var tenantID uint64
	if tenantIDStr != "" {
		if id, err := strconv.ParseUint(tenantIDStr, 10, 64); err == nil {
			tenantID = id
		}
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn("publish event panic",
					zap.Any("recover", r),
					zap.String("event_type", eventType),
				)
			}
		}()

		event := eventbus.NewEvent(eventType, tenantID, eventSource, data)
		if err := svc.eventPublisher.Publish(context.Background(), event); err != nil {
			logger.Warn("publish event failed",
				zap.String("event_type", eventType),
				zap.Error(err),
			)
		}
	}()
}

// Login authenticates an agent.
func (svc *AgentService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	// 1. Look up the agent.
	agent, err := svc.agentRepo.GetByAgentID(ctx, req.TenantID, req.AgentID)
	if err != nil {
		logger.Error("get agent failed", zap.Error(err), zap.String("agent_id", req.AgentID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if agent == nil {
		return nil, errors.ErrAgentNotFound
	}

	// 2. Validate the password.
	if !utils.CheckPassword(req.Password, agent.Password) {
		return nil, errors.ErrAgentPasswordWrong
	}

	// 3. Check the agent status.
	if agent.Status != 1 {
		return nil, errors.ErrAgentStatusInvalid
	}

	// 4. Update the last login time.
	if err := svc.agentRepo.UpdateLastLogin(ctx, req.TenantID, req.AgentID); err != nil {
		logger.Warn("update last login failed", zap.Error(err))
	}

	// 5. Generate a token.
	token, err := svc.generateToken(agent.AgentNumber, agent.AgentName, agent.TenantID)
	if err != nil {
		logger.Error("generate token failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeSystemError, "生成Token失败")
	}

	logger.Info("agent login success",
		zap.String("tenant_id", req.TenantID),
		zap.String("agent_id", req.AgentID),
	)

	return &model.LoginResponse{
		Token:     token,
		AgentID:   agent.AgentNumber,
		AgentName: agent.AgentName,
		ExpiresIn: svc.jwtExpireHour * 3600,
	}, nil
}

// generateToken generates a JWT.
func (svc *AgentService) generateToken(agentID, agentName, tenantID string) (string, error) {
	claims := AgentClaims{
		AgentID:   agentID,
		AgentName: agentName,
		TenantID:  tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    svc.jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(svc.jwtExpireHour) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(svc.jwtSecret))
}

// ParseToken parses a JWT.
func (svc *AgentService) ParseToken(tokenString string) (*AgentClaims, error) {
	claims := &AgentClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(svc.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// GetJWTSecret returns the JWT secret for middleware.
func (svc *AgentService) GetJWTSecret() string {
	return svc.jwtSecret
}

// Logout logs out an agent.
func (svc *AgentService) Logout(ctx context.Context, tenantID, agentID string) error {
	// Check out before deleting the session.
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		logger.Warn("get session failed on logout", zap.Error(err))
	}

	if session != nil && session.GetStatus() != model.StatusOffline {
		// Check out the agent.
		if err := svc.doCheckout(ctx, session, "logout"); err != nil {
			logger.Warn("checkout on logout failed", zap.Error(err))
		}
	}

	// Delete the session.
	if err := svc.sessionSvc.DeleteSession(ctx, tenantID, agentID); err != nil {
		logger.Warn("delete session failed", zap.Error(err))
	}

	logger.Info("agent logout",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
	)
	return nil
}

// Checkin checks in an agent.
func (svc *AgentService) Checkin(ctx context.Context, tenantID, agentID string, req *model.CheckinRequest) error {
	// 1. Look up the agent.
	agent, err := svc.agentRepo.GetByAgentID(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if agent == nil {
		return errors.ErrAgentNotFound
	}

	// 2. Check the extension.
	dn, err := svc.dnRepo.GetByDnNumber(ctx, tenantID, req.DnNumber)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if dn == nil {
		return errors.ErrDNNotFound
	}
	if dn.Status != 1 {
		return errors.ErrDNNotAvailable
	}

	// 3. Check whether the agent is already checked in.
	existingSession, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if existingSession != nil && existingSession.GetStatus().IsOnline() {
		return errors.ErrAgentAlreadyCheckedIn
	}

	// 4. Get the agent's skill groups.
	skillIDs := req.Skills
	if len(skillIDs) == 0 {
		// Load the default skill groups from the database.
		dbSkills, err := svc.agentRepo.GetAgentSkills(ctx, tenantID, agentID)
		if err != nil {
			logger.Warn("get agent skills from db failed", zap.Error(err))
		} else {
			skillIDs = dbSkills
		}
	}

	// 5. Create a session in the CHECKIN state.
	now := utils.NowUnix()
	session := &model.AgentSession{
		AgentID:     agentID,
		AgentName:   agent.AgentName,
		TenantID:    tenantID,
		Status:      int(model.StatusCheckin),
		DnNumber:    req.DnNumber,
		SkillGroups: skillIDs,
		CheckinTime: now,
		IP:          req.IP,
	}

	if err := svc.sessionSvc.SetSession(ctx, session); err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, "创建会话失败")
	}

	// 6. Add the agent to the online set.
	if err := svc.sessionSvc.AddOnline(ctx, tenantID, agentID); err != nil {
		logger.Warn("add online failed", zap.Error(err))
	}

	// 7. Cache the skill groups.
	if len(skillIDs) > 0 {
		if err := svc.sessionSvc.SetAgentSkills(ctx, agentID, skillIDs); err != nil {
			logger.Warn("set agent skills failed", zap.Error(err))
		}
	}

	// 8. Bind the extension.
	if err := svc.dnRepo.BindAgent(ctx, tenantID, req.DnNumber, agentID); err != nil {
		logger.Warn("bind dn agent failed", zap.Error(err))
	}

	// 9. Transition from CHECKIN to CHECKED.
	if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusChecked); err != nil {
		logger.Error("transition to CHECKED failed", zap.Error(err))
		// Roll back the session.
		_ = svc.sessionSvc.DeleteSession(ctx, tenantID, agentID)
		_ = svc.sessionSvc.RemoveOnline(ctx, tenantID, agentID)
		return err
	}

	// 10. Create a service record.
	record := &model.AgentServiceRecord{
		RecordID:    utils.UUID(),
		AgentID:     agentID,
		TenantID:    tenantID,
		ServiceType: 1,
		CheckinTime: time.Now(),
	}
	if err := svc.agentRepo.CreateServiceRecord(ctx, record); err != nil {
		logger.Warn("create service record failed", zap.Error(err))
	}

	// 11. Create the online agent record in the database.
	onlineAgent := &model.OnlineAgent{
		AgentID:       agentID,
		AgentName:     agent.AgentName,
		TenantID:      tenantID,
		DnNumber:      req.DnNumber,
		Status:        int(model.StatusChecked),
		StatusName:    model.StatusChecked.String(),
		IP:            req.IP,
		CheckinTime:   time.Now(),
		LastHeartbeat: time.Now(),
	}
	if err := svc.onlineAgentRepo.Upsert(ctx, onlineAgent); err != nil {
		logger.Warn("upsert online agent failed", zap.Error(err))
	}

	logger.Info("agent checkin success",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("dn_number", req.DnNumber),
		zap.Int("skills_count", len(skillIDs)),
	)

	// Publish the check-in event asynchronously.
	svc.publishEvent(eventbus.EventAgentCheckIn, tenantID, AgentEventData{
		AgentID:     parseID(agentID, "agent_id"),
		TenantID:    parseID(tenantID, "tenant_id"),
		AgentName:   agent.AgentName,
		Status:      model.StatusChecked.String(),
		StatusText:  model.StatusChecked.String(),
		PrevStatus:  model.StatusOffline.String(),
		Extension:   req.DnNumber,
		SkillGroups: parseSkillGroupIDs(skillIDs),
	})

	return nil
}

// Checkout checks out an agent.
func (svc *AgentService) Checkout(ctx context.Context, tenantID, agentID string, req *model.CheckoutRequest) error {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}

	return svc.doCheckout(ctx, session, req.Reason)
}

func (svc *AgentService) doCheckout(ctx context.Context, session *model.AgentSession, reason string) error {
	agentID := session.AgentID
	tenantID := session.TenantID
	prevStatus := session.GetStatus()
	prevSkillGroups := session.SkillGroups
	agentName := session.AgentName
	extension := session.DnNumber

	// 1. Remove the agent from all idle queues.
	if len(session.SkillGroups) > 0 {
		if err := svc.sessionSvc.RemoveIdleBatch(ctx, agentID, session.SkillGroups); err != nil {
			logger.Warn("remove idle batch failed", zap.Error(err))
		}
	}

	// 2. Transition to OFFLINE.
	if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusOffline); err != nil {
		// Force the state to OFFLINE.
		if updErr := svc.sessionSvc.UpdateStatus(ctx, tenantID, agentID, model.StatusOffline); updErr != nil {
			logger.Warn("force update status to offline failed", zap.Error(updErr))
		}
	}

	// 3. Remove the agent from the online set.
	if err := svc.sessionSvc.RemoveOnline(ctx, tenantID, agentID); err != nil {
		logger.Warn("remove online failed", zap.Error(err))
	}

	// 4. Delete the skill group cache.
	if err := svc.sessionSvc.DeleteAgentSkills(ctx, agentID); err != nil {
		logger.Warn("delete agent skills failed", zap.Error(err))
	}

	// 5. Unbind the extension.
	if session.DnNumber != "" {
		if err := svc.dnRepo.UnbindAgent(ctx, tenantID, session.DnNumber); err != nil {
			logger.Warn("unbind dn failed", zap.Error(err))
		}
	}

	// 6. Delete the session.
	if err := svc.sessionSvc.DeleteSession(ctx, tenantID, agentID); err != nil {
		logger.Warn("delete session failed", zap.Error(err))
	}

	// 7. Update the service record.
	record, err := svc.agentRepo.GetLatestServiceRecord(ctx, tenantID, agentID)
	if err != nil {
		logger.Warn("get latest service record failed", zap.Error(err))
	} else if record != nil {
		checkoutAt := time.Now()
		record.CheckoutTime = &checkoutAt
		if err := svc.agentRepo.UpdateServiceRecord(ctx, record); err != nil {
			logger.Warn("update service record failed", zap.Error(err))
		}
	}

	// 8. Delete the online agent record from the database.
	if err := svc.onlineAgentRepo.Delete(ctx, tenantID, agentID); err != nil {
		logger.Warn("delete online agent record failed", zap.Error(err))
	}

	logger.Info("agent checkout",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("reason", reason),
	)

	// Publish the checkout event asynchronously.
	svc.publishEvent(eventbus.EventAgentCheckOut, tenantID, AgentEventData{
		AgentID:     parseID(agentID, "agent_id"),
		TenantID:    parseID(tenantID, "tenant_id"),
		AgentName:   agentName,
		Status:      model.StatusOffline.String(),
		StatusText:  model.StatusOffline.String(),
		PrevStatus:  prevStatus.String(),
		Extension:   extension,
		SkillGroups: parseSkillGroupIDs(prevSkillGroups),
	})

	return nil
}

// TransitionStatus validates and performs an agent state transition.
func (svc *AgentService) TransitionStatus(ctx context.Context, tenantID, agentID string, target model.AgentStatus) error {
	// 1. Get the current state.
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}

	currentStatus := session.GetStatus()

	// 2. Validate the state transition.
	if !currentStatus.CanTransition(target) {
		logger.Warn("invalid status transition",
			zap.String("agent_id", agentID),
			zap.String("from", currentStatus.String()),
			zap.String("to", target.String()),
		)
		return errors.WrapWithCode(
			fmt.Errorf("cannot transition from %s to %s", currentStatus.String(), target.String()),
			errors.CodeAgentStateTransition,
			errors.MsgAgentStateTransition,
		)
	}

	// 3. Perform the state transition.
	if err := svc.doTransition(ctx, session, target); err != nil {
		return err
	}

	logger.Info("agent status transition",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("from", currentStatus.String()),
		zap.String("to", target.String()),
	)

	return nil
}

// doTransition performs a state transition without validation for internal callers.
func (svc *AgentService) doTransition(ctx context.Context, session *model.AgentSession, target model.AgentStatus) error {
	agentID := session.AgentID
	tenantID := session.TenantID
	from := session.GetStatus()
	now := utils.NowUnix()

	// Add the agent to idle queues when entering READY.
	if target == model.StatusReady {
		if err := svc.checkReadyLimit(ctx); err != nil {
			return err
		}
		session.LastReadyTime = now
		if len(session.SkillGroups) > 0 {
			if err := svc.sessionSvc.AddIdleBatch(ctx, agentID, session.SkillGroups, now); err != nil {
				return errors.WrapWithCode(err, errors.CodeRedisError, "加入空闲队列失败")
			}
		}
	}

	// Remove the agent from idle queues when leaving READY.
	if from == model.StatusReady && target != model.StatusReady {
		if len(session.SkillGroups) > 0 {
			if err := svc.sessionSvc.RemoveIdleBatch(ctx, agentID, session.SkillGroups); err != nil {
				logger.Warn("remove idle batch failed", zap.Error(err))
			}
		}
	}

	// Update the state.
	session.Status = int(target)
	session.LastActiveTime = now

	if err := svc.sessionSvc.SetSession(ctx, session); err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, "更新会话失败")
	}

	// Update the online record state in the database asynchronously without affecting the main flow.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn("update online agent status panic", zap.Any("recover", r))
			}
		}()
		bgCtx := context.Background()
		_ = svc.onlineAgentRepo.UpdateStatus(bgCtx, tenantID, agentID, int(target))
	}()

	// Publish the status change event asynchronously without blocking the main flow.
	svc.publishEvent(eventbus.EventAgentStatusChange, tenantID, AgentEventData{
		AgentID:     parseID(agentID, "agent_id"),
		TenantID:    parseID(tenantID, "tenant_id"),
		AgentName:   session.AgentName,
		Status:      target.String(),
		StatusText:  target.String(),
		PrevStatus:  from.String(),
		Extension:   session.DnNumber,
		SkillGroups: parseSkillGroupIDs(session.SkillGroups),
	})

	return nil
}

// Heartbeat keeps an agent session alive.
func (svc *AgentService) Heartbeat(ctx context.Context, tenantID, agentID string) (*model.HeartbeatResponse, error) {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return nil, errors.ErrSessionExpired
	}

	// Renew the session TTL.
	if err := svc.sessionSvc.RefreshTTL(ctx, tenantID, agentID); err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, "续期失败")
	}

	// Update the database heartbeat time asynchronously.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn("update heartbeat panic", zap.Any("recover", r))
			}
		}()
		bgCtx := context.Background()
		_ = svc.onlineAgentRepo.UpdateHeartbeat(bgCtx, tenantID, agentID)
	}()

	return &model.HeartbeatResponse{
		AgentID: agentID,
		Status:  session.Status,
	}, nil
}

// InternalEstablishSession establishes or restores an agent session for a push WebSocket connection, with idempotent check-in semantics.
// For an existing unexpired session, it renews the TTL and returns the session without resetting its state so reconnects preserve state.
// For a missing session, it validates the agent and extension and follows the Checkin flow without returning a duplicate check-in error.
func (svc *AgentService) InternalEstablishSession(ctx context.Context, tenantID, agentID, dnNumber string) (*model.AgentSession, error) {
	// 1. Restore the session idempotently.
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session != nil {
		if err := svc.sessionSvc.RefreshTTL(ctx, tenantID, agentID); err != nil {
			logger.Warn("refresh session ttl on establish failed", zap.String("agent_id", agentID), zap.Error(err))
		}
		return session, nil
	}

	// 2. Validate the agent before creating a session.
	agent, err := svc.agentRepo.GetByAgentID(ctx, tenantID, agentID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if agent == nil {
		return nil, errors.ErrAgentNotFound
	}
	if agent.Status != 1 {
		return nil, errors.ErrAgentStatusInvalid
	}

	// 3. Validate the extension.
	dn, err := svc.dnRepo.GetByDnNumber(ctx, tenantID, dnNumber)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if dn == nil {
		return nil, errors.ErrDNNotFound
	}
	if dn.Status != 1 {
		return nil, errors.ErrDNNotAvailable
	}

	// 4. Load the skill groups.
	skillIDs, err := svc.agentRepo.GetAgentSkills(ctx, tenantID, agentID)
	if err != nil {
		logger.Warn("get agent skills from db failed", zap.Error(err))
	}

	// 5. Create a session in the CHECKIN state.
	now := utils.NowUnix()
	session = &model.AgentSession{
		AgentID:     agentID,
		AgentName:   agent.AgentName,
		TenantID:    tenantID,
		Status:      int(model.StatusCheckin),
		DnNumber:    dnNumber,
		SkillGroups: skillIDs,
		CheckinTime: now,
	}
	if err := svc.sessionSvc.SetSession(ctx, session); err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, "创建会话失败")
	}
	if err := svc.sessionSvc.AddOnline(ctx, tenantID, agentID); err != nil {
		logger.Warn("add online failed", zap.Error(err))
	}
	if len(skillIDs) > 0 {
		if err := svc.sessionSvc.SetAgentSkills(ctx, agentID, skillIDs); err != nil {
			logger.Warn("set agent skills failed", zap.Error(err))
		}
	}
	if err := svc.dnRepo.BindAgent(ctx, tenantID, dnNumber, agentID); err != nil {
		logger.Warn("bind dn agent failed", zap.Error(err))
	}

	// 6. Transition from CHECKIN to CHECKED and roll back on failure.
	if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusChecked); err != nil {
		logger.Error("transition to CHECKED failed", zap.Error(err))
		_ = svc.sessionSvc.DeleteSession(ctx, tenantID, agentID)
		_ = svc.sessionSvc.RemoveOnline(ctx, tenantID, agentID)
		return nil, err
	}

	// 7. Create a service record as in Checkin step 10 so WS and REST check-in use consistent statistics.
	record := &model.AgentServiceRecord{
		RecordID:    utils.UUID(),
		AgentID:     agentID,
		TenantID:    tenantID,
		ServiceType: 1,
		CheckinTime: time.Now(),
	}
	if err := svc.agentRepo.CreateServiceRecord(ctx, record); err != nil {
		logger.Warn("create service record failed", zap.Error(err))
	}

	// 8. Create the online agent record in the database.
	onlineAgent := &model.OnlineAgent{
		AgentID:       agentID,
		AgentName:     agent.AgentName,
		TenantID:      tenantID,
		DnNumber:      dnNumber,
		Status:        int(model.StatusChecked),
		StatusName:    model.StatusChecked.String(),
		CheckinTime:   time.Now(),
		LastHeartbeat: time.Now(),
	}
	if err := svc.onlineAgentRepo.Upsert(ctx, onlineAgent); err != nil {
		logger.Warn("upsert online agent failed", zap.Error(err))
	}

	logger.Info("agent session established via ws",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("dn_number", dnNumber),
	)

	// 9. Publish the authoritative check-in state event, consistent with Checkin.
	svc.publishEvent(eventbus.EventAgentCheckIn, tenantID, AgentEventData{
		AgentID:     parseID(agentID, "agent_id"),
		TenantID:    parseID(tenantID, "tenant_id"),
		AgentName:   agent.AgentName,
		Status:      model.StatusChecked.String(),
		StatusText:  model.StatusChecked.String(),
		PrevStatus:  model.StatusOffline.String(),
		Extension:   dnNumber,
		SkillGroups: parseSkillGroupIDs(skillIDs),
	})

	// Return the latest session after the state transition.
	return svc.sessionSvc.GetSession(ctx, tenantID, agentID)
}

// InternalRefreshSession renews a session for the push keepalive goroutine.
func (svc *AgentService) InternalRefreshSession(ctx context.Context, tenantID, agentID string) error {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrSessionExpired
	}
	if err := svc.sessionSvc.RefreshTTL(ctx, tenantID, agentID); err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, "续期失败")
	}
	// Update the database heartbeat time asynchronously, consistent with Heartbeat.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn("update heartbeat panic", zap.Any("recover", r))
			}
		}()
		_ = svc.onlineAgentRepo.UpdateHeartbeat(context.Background(), tenantID, agentID)
	}()
	return nil
}

// InternalDeleteSession checks out an agent and clears the session idempotently.
func (svc *AgentService) InternalDeleteSession(ctx context.Context, tenantID, agentID string) error {
	return svc.Logout(ctx, tenantID, agentID) // Logout is idempotent and only deletes keys when no session exists.
}

// Busy marks an agent as busy.
func (svc *AgentService) Busy(ctx context.Context, tenantID, agentID string, reason string) error {
	return svc.TransitionStatus(ctx, tenantID, agentID, model.StatusBusy)
}

// Ready marks an agent as ready.
func (svc *AgentService) Ready(ctx context.Context, tenantID, agentID string) error {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}

	return svc.TransitionStatus(ctx, tenantID, agentID, model.StatusReady)
}

// EndAfterWork ends after-call work.
func (svc *AgentService) EndAfterWork(ctx context.Context, tenantID, agentID string) error {
	return svc.TransitionStatus(ctx, tenantID, agentID, model.StatusReady)
}

// GetAgentInfo returns agent information.
func (svc *AgentService) GetAgentInfo(ctx context.Context, tenantID, agentID string) (*model.AgentInfoResponse, error) {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}

	if session == nil {
		// Load basic information from the database.
		agent, err := svc.agentRepo.GetByAgentID(ctx, tenantID, agentID)
		if err != nil {
			return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
		}
		if agent == nil {
			return nil, errors.ErrAgentNotFound
		}
		return &model.AgentInfoResponse{
			AgentID:    agent.AgentNumber,
			AgentName:  agent.AgentName,
			TenantID:   agent.TenantID,
			Status:     int(model.StatusOffline),
			StatusName: model.StatusOffline.String(),
		}, nil
	}

	status := session.GetStatus()
	return &model.AgentInfoResponse{
		AgentID:       session.AgentID,
		AgentName:     session.AgentName,
		TenantID:      session.TenantID,
		Status:        session.Status,
		StatusName:    status.String(),
		DnNumber:      session.DnNumber,
		SkillGroups:   session.SkillGroups,
		CurrentCallID: session.CurrentCallID,
		LastReadyTime: session.LastReadyTime,
		CheckinTime:   session.CheckinTime,
	}, nil
}

// GetAgentStatus returns the agent status.
func (svc *AgentService) GetAgentStatus(ctx context.Context, tenantID, agentID string) (*model.AgentStatusResponse, error) {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}

	status := model.StatusOffline
	if session != nil {
		status = session.GetStatus()
	}

	return &model.AgentStatusResponse{
		AgentID:    agentID,
		Status:     int(status),
		StatusName: status.String(),
	}, nil
}

// ---------- Internal management API for ACD ----------

// TryAcquireAgent attempts to allocate an idle agent from a skill group, using a lock to prevent oversell.
func (svc *AgentService) TryAcquireAgent(ctx context.Context, tenantID, skillGroupID string) (string, error) {
	// Retry up to three times.
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		// 1. Pop an agent from the idle queue.
		agentID, _, err := svc.sessionSvc.PopIdle(ctx, skillGroupID)
		if err != nil {
			return "", errors.WrapWithCode(err, errors.CodeRedisError, "弹出空闲座席失败")
		}
		if agentID == "" {
			return "", errors.ErrAgentNoIdle
		}

		// 2. Get the agent session and check its current state.
		session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
		if err != nil || session == nil {
			// The session is invalid, so continue with the next agent.
			logger.Warn("idle agent session not found, skip",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
			continue
		}

		// 3. Acquire the distributed lock.
		ok, lockValue, err := svc.sessionSvc.TryLock(ctx, agentID, 5*time.Second)
		if err != nil {
			// Return the agent to the idle queue if locking fails.
			_ = svc.sessionSvc.AddIdle(ctx, skillGroupID, agentID, session.LastReadyTime)
			return "", errors.WrapWithCode(err, errors.CodeRedisError, "加锁失败")
		}
		if !ok {
			// Another allocation flow holds the lock, so skip this agent.
			logger.Info("agent locked, skip", zap.String("agent_id", agentID))
			continue
		}

		// 4. Perform second-chance validation of the state.
		currentSession, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
		if err != nil || currentSession == nil || !currentSession.GetStatus().IsIdle() {
			// The state has changed, so release the lock and continue with the next agent.
			_, _ = svc.sessionSvc.Unlock(ctx, agentID, lockValue)
			logger.Info("agent not idle, skip", zap.String("agent_id", agentID))
			continue
		}

		// 5. Transition to WAIT_SERVER.
		if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusWaitServer); err != nil {
			_, _ = svc.sessionSvc.Unlock(ctx, agentID, lockValue)
			logger.Warn("transition to wait server failed", zap.Error(err))
			continue
		}

		// 6. Release the lock.
		_, _ = svc.sessionSvc.Unlock(ctx, agentID, lockValue)

		logger.Info("acquire idle agent success",
			zap.String("skill_group_id", skillGroupID),
			zap.String("agent_id", agentID),
		)

		return agentID, nil
	}

	return "", errors.ErrAgentNoIdle
}

// Ring notifies an agent of an incoming call.
func (svc *AgentService) Ring(ctx context.Context, tenantID, agentID, callID string) error {
	// Get the session.
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}

	session.CurrentCallID = callID

	// WAIT_SERVER → RINGING
	if session.GetStatus() == model.StatusWaitServer {
		if err := svc.doTransition(ctx, session, model.StatusRinging); err != nil {
			return err
		}
	}

	return nil
}

// ResumeReady resets an agent to ready after an allocation failure.
func (svc *AgentService) ResumeReady(ctx context.Context, tenantID, agentID string) error {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}

	// Clear the current call.
	session.CurrentCallID = ""

	return svc.doTransition(ctx, session, model.StatusReady)
}

// InternalCallAnswer transitions an agent from RINGING to CALLING after the agent answers.
// It ignores a callID that differs from the agent's current call so unrelated call events cannot change the state.
func (svc *AgentService) InternalCallAnswer(ctx context.Context, tenantID, agentID, callID string) error {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}
	if session.CurrentCallID != callID {
		logger.Warn("call answer event call mismatch, ignore",
			zap.String("agent_id", agentID),
			zap.String("event_call_id", callID),
			zap.String("current_call_id", session.CurrentCallID),
		)
		return nil
	}
	if session.GetStatus() != model.StatusRinging {
		return nil
	}
	if err := svc.doTransition(ctx, session, model.StatusCalling); err != nil {
		return err
	}
	logger.Info("agent call answered",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("call_id", callID),
	)
	return nil
}

// InternalCallEnd releases an agent to READY and returns the agent to the idle pool when a call ends.
// It resets the agent to READY when the call ends while waiting for service, ringing, or in progress.
// Non-call states such as READY, BUSY, and OFFLINE are handled idempotently without changing the current state.
func (svc *AgentService) InternalCallEnd(ctx context.Context, tenantID, agentID, callID string) error {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}
	if !session.GetStatus().IsInCall() {
		return nil
	}
	// A mismatched callID means the agent is serving another call and must not be released.
	if callID != "" && session.CurrentCallID != "" && session.CurrentCallID != callID {
		logger.Warn("call end event call mismatch, ignore",
			zap.String("agent_id", agentID),
			zap.String("event_call_id", callID),
			zap.String("current_call_id", session.CurrentCallID),
		)
		return nil
	}
	session.CurrentCallID = ""
	if err := svc.doTransition(ctx, session, model.StatusReady); err != nil {
		return err
	}
	logger.Info("agent call ended and released to ready",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("call_id", callID),
	)
	return nil
}

// UpdateStatus updates an agent state for ACD, with an option to force the transition; reason is passed through when the target is BUSY.
func (svc *AgentService) UpdateStatus(ctx context.Context, tenantID, agentID string, target model.AgentStatus, force bool, reason string) error {
	if force {
		// Force the update without validating the state transition.
		session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
		if err != nil {
			return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
		}
		if session == nil {
			return errors.ErrAgentOffline
		}
		return svc.doTransition(ctx, session, target)
	}

	// Pass the reason through Busy when marking the agent busy.
	if target == model.StatusBusy {
		return svc.Busy(ctx, tenantID, agentID, reason)
	}

	// Perform a normal state transition.
	return svc.TransitionStatus(ctx, tenantID, agentID, target)
}

// GetSessionService exposes the session service to other internal modules.
func (svc *AgentService) GetSessionService() *SessionService {
	return svc.sessionSvc
}
