package service

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
)

// PushService 推送服务
// 提供各种推送方法，是对外推送的核心入口
// 所有推送操作最终都通过此服务触发
//
// 推送流程：
// 1. 调用方调用 PushService 的推送方法
// 2. PushService 构造消息，调用 ConnectionManager 查询目标连接
// 3. 对于不在本节点的座席，通过 RouterService 转发到目标节点
// 4. 目标连接通过 sendCh 异步发送消息
type PushService struct {
	connManager    *ConnectionManager
	subService     *SubscriptionService
	routerService  *RouterService
}

// NewPushService 创建推送服务
func NewPushService(
	connManager *ConnectionManager,
	subService *SubscriptionService,
	routerService *RouterService,
) *PushService {
	return &PushService{
		connManager:   connManager,
		subService:    subService,
		routerService: routerService,
	}
}

// ===== 推送给指定座席 =====

// PushToAgent 推送给指定座席
// 如果座席在本节点，直接推送；如果在其他节点，通过路由服务转发
func (s *PushService) PushToAgent(tenantID, agentID int64, event string, data interface{}) (int, error) {
	// 构造消息
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// 先尝试本节点推送
	count := s.pushToAgentLocal(tenantID, agentID, msgBytes)

	// 如果路由服务可用，也尝试跨节点推送
	if s.routerService != nil && s.routerService.IsEnabled() {
		remoteCount, err := s.routerService.PushToAgent(tenantID, agentID, msg)
		if err != nil {
			logger.Warn("push to agent via router failed",
				zap.Int64("agent_id", agentID),
				zap.Int64("tenant_id", tenantID),
				zap.String("event", event),
				zap.Error(err),
			)
		}
		count += remoteCount
	}

	logger.Debug("push to agent",
		zap.Int64("agent_id", agentID),
		zap.Int64("tenant_id", tenantID),
		zap.String("event", event),
		zap.Int("count", count),
	)

	return count, nil
}

// pushToAgentLocal 本地节点推送给座席
func (s *PushService) pushToAgentLocal(tenantID, agentID int64, msgBytes []byte) int {
	conns := s.connManager.GetAgentConnections(tenantID, agentID)
	if len(conns) == 0 {
		return 0
	}

	count := 0
	for _, conn := range conns {
		if !conn.IsConnected() {
			continue
		}
		if conn.Send(msgBytes) {
			count++
		} else {
			s.connManager.IncrementFailedMessages()
			logger.Warn("send to agent connection failed, buffer full",
				zap.String("conn_id", conn.ConnID()),
				zap.Int64("agent_id", agentID),
			)
		}
	}
	return count
}

// ===== 推送给整个租户 =====

// PushToTenant 推送给指定租户的所有连接
// 包括该租户下的所有座席、管理员、监控大屏
func (s *PushService) PushToTenant(tenantID int64, event string, data interface{}, userTypes []string) (int, error) {
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// 本地推送
	count := s.pushToTenantLocal(tenantID, msgBytes, userTypes)

	// 跨节点推送（通过主题方式）
	if s.routerService != nil && s.routerService.IsEnabled() {
		topic := s.subService.TenantTopic(tenantID)
		remoteCount, err := s.routerService.PushToTopic(topic, msg)
		if err != nil {
			logger.Warn("push to tenant via router failed",
				zap.Int64("tenant_id", tenantID),
				zap.String("event", event),
				zap.Error(err),
			)
		}
		count += remoteCount
	}

	logger.Debug("push to tenant",
		zap.Int64("tenant_id", tenantID),
		zap.String("event", event),
		zap.Int("count", count),
	)

	return count, nil
}

// pushToTenantLocal 本地推送给租户
func (s *PushService) pushToTenantLocal(tenantID int64, msgBytes []byte, userTypes []string) int {
	count := 0
	typeFilter := make(map[string]bool)
	for _, t := range userTypes {
		typeFilter[t] = true
	}
	hasFilter := len(typeFilter) > 0

	// 推送给座席
	if !hasFilter || typeFilter[model.UserTypeAgent] {
		agentConns := s.connManager.GetTenantAgentConns(tenantID)
		for _, conn := range agentConns {
			if !conn.IsConnected() {
				continue
			}
			if conn.Send(msgBytes) {
				count++
			} else {
				s.connManager.IncrementFailedMessages()
			}
		}
	}

	// 推送给管理员（需要匹配租户）
	if !hasFilter || typeFilter[model.UserTypeAdmin] {
		adminConns := s.connManager.GetAllAdminConnections()
		for _, conn := range adminConns {
			if !conn.IsConnected() {
				continue
			}
			if conn.TenantID() == tenantID || conn.TenantID() == 0 {
				if conn.Send(msgBytes) {
					count++
				} else {
					s.connManager.IncrementFailedMessages()
				}
			}
		}
	}

	// 推送给监控大屏（需要匹配租户）
	if !hasFilter || typeFilter[model.UserTypeMonitor] {
		monitorConns := s.connManager.GetAllMonitorConnections()
		for _, conn := range monitorConns {
			if !conn.IsConnected() {
				continue
			}
			if conn.TenantID() == tenantID || conn.TenantID() == 0 {
				if conn.Send(msgBytes) {
					count++
				} else {
					s.connManager.IncrementFailedMessages()
				}
			}
		}
	}

	return count
}

// ===== 推送到主题 =====

// PushToTopic 推送到指定主题
// 所有订阅了该主题的连接都会收到消息
func (s *PushService) PushToTopic(topic string, event string, data interface{}) (int, error) {
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// 本地推送
	count := s.pushToTopicLocal(topic, msgBytes)

	// 跨节点推送
	if s.routerService != nil && s.routerService.IsEnabled() {
		remoteCount, err := s.routerService.PushToTopic(topic, msg)
		if err != nil {
			logger.Warn("push to topic via router failed",
				zap.String("topic", topic),
				zap.String("event", event),
				zap.Error(err),
			)
		}
		count += remoteCount
	}

	logger.Debug("push to topic",
		zap.String("topic", topic),
		zap.String("event", event),
		zap.Int("count", count),
	)

	return count, nil
}

// pushToTopicLocal 本地推送到主题
func (s *PushService) pushToTopicLocal(topic string, msgBytes []byte) int {
	conns := s.subService.GetTopicSubscribers(topic)
	if len(conns) == 0 {
		return 0
	}

	count := 0
	for _, conn := range conns {
		if !conn.IsConnected() {
			continue
		}
		if conn.Send(msgBytes) {
			count++
		} else {
			s.connManager.IncrementFailedMessages()
		}
	}
	return count
}

// ===== 广播 =====

// PushToAll 广播给所有连接
func (s *PushService) PushToAll(event string, data interface{}) (int, error) {
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// 本地广播
	count := s.pushToAllLocal(msgBytes)

	// 跨节点广播
	if s.routerService != nil && s.routerService.IsEnabled() {
		remoteCount, err := s.routerService.PushToAll(msg)
		if err != nil {
			logger.Warn("push to all via router failed",
				zap.String("event", event),
				zap.Error(err),
			)
		}
		count += remoteCount
	}

	logger.Debug("push to all",
		zap.String("event", event),
		zap.Int("count", count),
	)

	return count, nil
}

// pushToAllLocal 本地广播
func (s *PushService) pushToAllLocal(msgBytes []byte) int {
	// 通过广播主题推送
	topic := s.subService.BroadcastTopic()
	count := s.pushToTopicLocal(topic, msgBytes)

	// 另外推送给所有连接（确保即使没订阅广播主题也能收到）
	// 这里可以选择是否需要，暂时通过主题方式即可
	return count
}

// ===== 推送系统通知 =====

// PushSystemNotice 推送系统通知
func (s *PushService) PushSystemNotice(tenantID int64, notice *model.SystemNoticeData) (int, error) {
	return s.PushToTenant(tenantID, model.EventSystemNotice, notice, nil)
}

// PushError 推送错误通知给指定连接
func (s *PushService) PushError(conn *model.Connection, code int, message, detail string) {
	if conn == nil || !conn.IsConnected() {
		return
	}

	errData := &model.ErrorData{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
	msg := model.NewMessage(model.EventError, errData)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Error("marshal error message failed", zap.Error(err))
		return
	}

	conn.Send(msgBytes)
}

// ===== 心跳响应 =====

// SendHeartbeat 发送心跳响应给指定连接
func (s *PushService) SendHeartbeat(conn *model.Connection) {
	if conn == nil || !conn.IsConnected() {
		return
	}

	stats := s.connManager.GetStats()
	hbData := &model.HeartbeatData{
		ServerTime: model.NewMessage(model.EventHeartbeat, nil).Timestamp,
		Load:       stats.TotalConns,
	}
	msg := model.NewMessage(model.EventHeartbeat, hbData)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Error("marshal heartbeat message failed", zap.Error(err))
		return
	}

	conn.Send(msgBytes)
}
