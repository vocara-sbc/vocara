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

// eventSource 事件来源服务名
const eventSource = "agentcenter"

// AgentEventData 座席事件数据，与 push 服务的座席事件字段保持一致。
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

// parseID 安全地将字符串 ID 解析为 int64，失败返回 0 并记录日志。
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

// parseSkillGroupIDs 将字符串技能组 ID 数组转换为 int64 数组，转换失败的元素跳过。
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

// AgentClaims 座席 JWT 负载
type AgentClaims struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	TenantID  string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// AgentService 座席服务
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

// NewAgentService 创建座席服务
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

// checkReadyLimit 在进入就绪态前校验授权容量，统计的是当前已就绪座席数（本座席尚未入集合）。
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

// publishEvent 异步发布事件，不阻塞主流程
func (svc *AgentService) publishEvent(eventType string, tenantIDStr string, data interface{}) {
	if svc.eventPublisher == nil {
		return
	}

	// 解析 tenant_id 为 uint64（事件总线使用 uint64）
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

// Login 座席登录
func (svc *AgentService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	// 1. 查询座席
	agent, err := svc.agentRepo.GetByAgentID(ctx, req.TenantID, req.AgentID)
	if err != nil {
		logger.Error("get agent failed", zap.Error(err), zap.String("agent_id", req.AgentID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if agent == nil {
		return nil, errors.ErrAgentNotFound
	}

	// 2. 校验密码
	if !utils.CheckPassword(req.Password, agent.Password) {
		return nil, errors.ErrAgentPasswordWrong
	}

	// 3. 检查座席状态
	if agent.Status != 1 {
		return nil, errors.ErrAgentStatusInvalid
	}

	// 4. 更新最后登录时间
	if err := svc.agentRepo.UpdateLastLogin(ctx, req.TenantID, req.AgentID); err != nil {
		logger.Warn("update last login failed", zap.Error(err))
	}

	// 5. 生成 Token
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

// generateToken 生成 JWT Token
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

// ParseToken 解析 JWT Token
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

// GetJWTSecret 获取 JWT 密钥（给中间件用）
func (svc *AgentService) GetJWTSecret() string {
	return svc.jwtSecret
}

// Logout 座席登出
func (svc *AgentService) Logout(ctx context.Context, tenantID, agentID string) error {
	// 先签出再删除会话
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		logger.Warn("get session failed on logout", zap.Error(err))
	}

	if session != nil && session.GetStatus() != model.StatusOffline {
		// 签出
		if err := svc.doCheckout(ctx, session, "logout"); err != nil {
			logger.Warn("checkout on logout failed", zap.Error(err))
		}
	}

	// 删除 session
	if err := svc.sessionSvc.DeleteSession(ctx, tenantID, agentID); err != nil {
		logger.Warn("delete session failed", zap.Error(err))
	}

	logger.Info("agent logout",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
	)
	return nil
}

// Checkin 座席签入
func (svc *AgentService) Checkin(ctx context.Context, tenantID, agentID string, req *model.CheckinRequest) error {
	// 1. 查询座席
	agent, err := svc.agentRepo.GetByAgentID(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if agent == nil {
		return errors.ErrAgentNotFound
	}

	// 2. 检查分机
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

	// 3. 检查是否已签入
	existingSession, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if existingSession != nil && existingSession.GetStatus().IsOnline() {
		return errors.ErrAgentAlreadyCheckedIn
	}

	// 4. 获取座席技能组
	skillIDs := req.Skills
	if len(skillIDs) == 0 {
		// 从数据库获取默认技能组
		dbSkills, err := svc.agentRepo.GetAgentSkills(ctx, tenantID, agentID)
		if err != nil {
			logger.Warn("get agent skills from db failed", zap.Error(err))
		} else {
			skillIDs = dbSkills
		}
	}

	// 5. 创建会话，状态 CHECKIN
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

	// 6. 加入在线集合
	if err := svc.sessionSvc.AddOnline(ctx, tenantID, agentID); err != nil {
		logger.Warn("add online failed", zap.Error(err))
	}

	// 7. 设置技能组缓存
	if len(skillIDs) > 0 {
		if err := svc.sessionSvc.SetAgentSkills(ctx, agentID, skillIDs); err != nil {
			logger.Warn("set agent skills failed", zap.Error(err))
		}
	}

	// 8. 绑定分机
	if err := svc.dnRepo.BindAgent(ctx, tenantID, req.DnNumber, agentID); err != nil {
		logger.Warn("bind dn agent failed", zap.Error(err))
	}

	// 9. 状态转换 CHECKIN → CHECKED
	if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusChecked); err != nil {
		logger.Error("transition to CHECKED failed", zap.Error(err))
		// 回滚
		_ = svc.sessionSvc.DeleteSession(ctx, tenantID, agentID)
		_ = svc.sessionSvc.RemoveOnline(ctx, tenantID, agentID)
		return err
	}

	// 10. 创建服务记录
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

	// 11. 创建数据库在线座席记录
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

	// 发布签入事件（异步）
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

// Checkout 座席签出
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

	// 1. 从所有空闲队列移除
	if len(session.SkillGroups) > 0 {
		if err := svc.sessionSvc.RemoveIdleBatch(ctx, agentID, session.SkillGroups); err != nil {
			logger.Warn("remove idle batch failed", zap.Error(err))
		}
	}

	// 2. 状态转换到 OFFLINE
	if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusOffline); err != nil {
		// 强制设为 OFFLINE
		if updErr := svc.sessionSvc.UpdateStatus(ctx, tenantID, agentID, model.StatusOffline); updErr != nil {
			logger.Warn("force update status to offline failed", zap.Error(updErr))
		}
	}

	// 3. 从在线集合移除
	if err := svc.sessionSvc.RemoveOnline(ctx, tenantID, agentID); err != nil {
		logger.Warn("remove online failed", zap.Error(err))
	}

	// 4. 删除技能组缓存
	if err := svc.sessionSvc.DeleteAgentSkills(ctx, agentID); err != nil {
		logger.Warn("delete agent skills failed", zap.Error(err))
	}

	// 5. 解绑分机
	if session.DnNumber != "" {
		if err := svc.dnRepo.UnbindAgent(ctx, tenantID, session.DnNumber); err != nil {
			logger.Warn("unbind dn failed", zap.Error(err))
		}
	}

	// 6. 删除 session
	if err := svc.sessionSvc.DeleteSession(ctx, tenantID, agentID); err != nil {
		logger.Warn("delete session failed", zap.Error(err))
	}

	// 7. 更新服务记录
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

	// 8. 删除数据库在线座席记录
	if err := svc.onlineAgentRepo.Delete(ctx, tenantID, agentID); err != nil {
		logger.Warn("delete online agent record failed", zap.Error(err))
	}

	logger.Info("agent checkout",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("reason", reason),
	)

	// 发布签出事件（异步）
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

// TransitionStatus 状态转换（带校验）
func (svc *AgentService) TransitionStatus(ctx context.Context, tenantID, agentID string, target model.AgentStatus) error {
	// 1. 获取当前状态
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}

	currentStatus := session.GetStatus()

	// 2. 校验状态转换
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

	// 3. 执行状态转换
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

// doTransition 执行状态转换（不做校验，内部调用）
func (svc *AgentService) doTransition(ctx context.Context, session *model.AgentSession, target model.AgentStatus) error {
	agentID := session.AgentID
	tenantID := session.TenantID
	from := session.GetStatus()
	now := utils.NowUnix()

	// 进入 READY：加入空闲队列
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

	// 离开 READY：从空闲队列移除
	if from == model.StatusReady && target != model.StatusReady {
		if len(session.SkillGroups) > 0 {
			if err := svc.sessionSvc.RemoveIdleBatch(ctx, agentID, session.SkillGroups); err != nil {
				logger.Warn("remove idle batch failed", zap.Error(err))
			}
		}
	}

	// 更新状态
	session.Status = int(target)
	session.LastActiveTime = now

	if err := svc.sessionSvc.SetSession(ctx, session); err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, "更新会话失败")
	}

	// 异步更新数据库在线记录状态（不影响主流程）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn("update online agent status panic", zap.Any("recover", r))
			}
		}()
		bgCtx := context.Background()
		_ = svc.onlineAgentRepo.UpdateStatus(bgCtx, tenantID, agentID, int(target))
	}()

	// 发布状态变更事件（异步，不阻塞主流程）
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

// Heartbeat 心跳保活
func (svc *AgentService) Heartbeat(ctx context.Context, tenantID, agentID string) (*model.HeartbeatResponse, error) {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return nil, errors.ErrSessionExpired
	}

	// 续期
	if err := svc.sessionSvc.RefreshTTL(ctx, tenantID, agentID); err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, "续期失败")
	}

	// 更新数据库心跳时间（异步）
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

// InternalEstablishSession 内部接口：建立或恢复座席会话（push WS 连接即签入，幂等）。
// 会话存在且未过期：续期并返回当前会话（重连携带状态恢复语义，不得重置状态）；
// 不存在：校验座席与分机后新建（复刻 Checkin 流程，不报重复签入错误）。
func (svc *AgentService) InternalEstablishSession(ctx context.Context, tenantID, agentID, dnNumber string) (*model.AgentSession, error) {
	// 1. 幂等恢复
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

	// 2. 新建：校验座席
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

	// 3. 校验分机
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

	// 4. 技能组
	skillIDs, err := svc.agentRepo.GetAgentSkills(ctx, tenantID, agentID)
	if err != nil {
		logger.Warn("get agent skills from db failed", zap.Error(err))
	}

	// 5. 创建会话（CHECKIN）
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

	// 6. 状态转换 CHECKIN → CHECKED（失败回滚）
	if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusChecked); err != nil {
		logger.Error("transition to CHECKED failed", zap.Error(err))
		_ = svc.sessionSvc.DeleteSession(ctx, tenantID, agentID)
		_ = svc.sessionSvc.RemoveOnline(ctx, tenantID, agentID)
		return nil, err
	}

	// 7. 创建服务记录（对齐 Checkin 第 10 步，保证 WS 签入与 REST 签入统计语义一致）
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

	// 8. 数据库在线座席记录
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

	// 9. 发布签入事件（权威状态广播，与 Checkin 一致）
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

	// 返回最新会话（状态已转换）
	return svc.sessionSvc.GetSession(ctx, tenantID, agentID)
}

// InternalRefreshSession 内部接口：会话续期（push 保活协程调用）。
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
	// 异步更新数据库心跳时间（与 Heartbeat 一致）
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

// InternalDeleteSession 内部接口：签出清会话（幂等）。
func (svc *AgentService) InternalDeleteSession(ctx context.Context, tenantID, agentID string) error {
	return svc.Logout(ctx, tenantID, agentID) // Logout 本身幂等：无会话时仅删键
}

// Busy 示忙
func (svc *AgentService) Busy(ctx context.Context, tenantID, agentID string, reason string) error {
	return svc.TransitionStatus(ctx, tenantID, agentID, model.StatusBusy)
}

// Ready 示闲
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

// EndAfterWork 结束事后处理
func (svc *AgentService) EndAfterWork(ctx context.Context, tenantID, agentID string) error {
	return svc.TransitionStatus(ctx, tenantID, agentID, model.StatusReady)
}

// GetAgentInfo 获取座席信息
func (svc *AgentService) GetAgentInfo(ctx context.Context, tenantID, agentID string) (*model.AgentInfoResponse, error) {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}

	if session == nil {
		// 从数据库获取基础信息
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

// GetAgentStatus 获取座席状态
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

// ---------- 内部管理接口（供 ACD 调用） ----------

// TryAcquireAgent 尝试从指定技能组分配一个空闲座席（加锁防超卖）
func (svc *AgentService) TryAcquireAgent(ctx context.Context, tenantID, skillGroupID string) (string, error) {
	// 最多重试 3 次
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		// 1. 从空闲队列弹出一个座席
		agentID, _, err := svc.sessionSvc.PopIdle(ctx, skillGroupID)
		if err != nil {
			return "", errors.WrapWithCode(err, errors.CodeRedisError, "弹出空闲座席失败")
		}
		if agentID == "" {
			return "", errors.ErrAgentNoIdle
		}

		// 2. 获取座席会话，检查当前状态
		session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
		if err != nil || session == nil {
			// 会话已失效，继续下一个
			logger.Warn("idle agent session not found, skip",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
			continue
		}

		// 3. 加分布式锁
		ok, lockValue, err := svc.sessionSvc.TryLock(ctx, agentID, 5*time.Second)
		if err != nil {
			// 加锁异常，把座席放回空闲队列
			_ = svc.sessionSvc.AddIdle(ctx, skillGroupID, agentID, session.LastReadyTime)
			return "", errors.WrapWithCode(err, errors.CodeRedisError, "加锁失败")
		}
		if !ok {
			// 锁被占用，说明正在被其他分配流程处理，跳过
			logger.Info("agent locked, skip", zap.String("agent_id", agentID))
			continue
		}

		// 4. 二次校验状态
		currentSession, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
		if err != nil || currentSession == nil || !currentSession.GetStatus().IsIdle() {
			// 状态已变化，释放锁，继续下一个
			_, _ = svc.sessionSvc.Unlock(ctx, agentID, lockValue)
			logger.Info("agent not idle, skip", zap.String("agent_id", agentID))
			continue
		}

		// 5. 状态转换到 WAIT_SERVER
		if err := svc.TransitionStatus(ctx, tenantID, agentID, model.StatusWaitServer); err != nil {
			_, _ = svc.sessionSvc.Unlock(ctx, agentID, lockValue)
			logger.Warn("transition to wait server failed", zap.Error(err))
			continue
		}

		// 6. 释放锁
		_, _ = svc.sessionSvc.Unlock(ctx, agentID, lockValue)

		logger.Info("acquire idle agent success",
			zap.String("skill_group_id", skillGroupID),
			zap.String("agent_id", agentID),
		)

		return agentID, nil
	}

	return "", errors.ErrAgentNoIdle
}

// Ring 通知座席振铃
func (svc *AgentService) Ring(ctx context.Context, tenantID, agentID, callID string) error {
	// 获取 session
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

// ResumeReady 分配失败，重置为空闲
func (svc *AgentService) ResumeReady(ctx context.Context, tenantID, agentID string) error {
	session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
	}
	if session == nil {
		return errors.ErrAgentOffline
	}

	// 清空当前通话
	session.CurrentCallID = ""

	return svc.doTransition(ctx, session, model.StatusReady)
}

// InternalCallAnswer 通话接听通知：座席摘机后 RINGING → CALLING。
// callID 与座席当前通话不一致时忽略，避免串通话的事件误改状态。
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

// InternalCallEnd 通话结束通知：座席释放回就绪并重新进入空闲池。
// 通话在等待服务、振铃、通话任一环节结束都统一复位为 READY；
// 非通话相关状态（已就绪、示忙、离线）按幂等处理，不改变当前状态。
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
	// callID 不匹配说明该座席已在服务新通话，不得释放
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

// UpdateStatus 内部更新座席状态（带强制选项，供 ACD 调用）；reason 为示忙原因，目标状态为 BUSY 时透传。
func (svc *AgentService) UpdateStatus(ctx context.Context, tenantID, agentID string, target model.AgentStatus, force bool, reason string) error {
	if force {
		// 强制更新，不校验状态转换
		session, err := svc.sessionSvc.GetSession(ctx, tenantID, agentID)
		if err != nil {
			return errors.WrapWithCode(err, errors.CodeRedisError, errors.MsgRedisError)
		}
		if session == nil {
			return errors.ErrAgentOffline
		}
		return svc.doTransition(ctx, session, target)
	}

	// 示忙走 Busy 透传原因
	if target == model.StatusBusy {
		return svc.Busy(ctx, tenantID, agentID, reason)
	}

	// 普通状态转换
	return svc.TransitionStatus(ctx, tenantID, agentID, target)
}

// GetSessionService 暴露 session service（给其他内部模块用）
func (svc *AgentService) GetSessionService() *SessionService {
	return svc.sessionSvc
}
