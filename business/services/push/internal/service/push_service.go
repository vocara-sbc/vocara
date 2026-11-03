package service

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
)

// PushService provides push operations.
// It is the core entry point for outbound delivery.
// All push operations are triggered through this service.
//
// Push flow:
// 1. The caller invokes a PushService push method.
// 2. PushService constructs the message and queries ConnectionManager for target connections.
// 3. RouterService forwards messages for agents on other nodes to their target nodes.
// 4. Target connections send messages asynchronously through sendCh.
type PushService struct {
	connManager    *ConnectionManager
	subService     *SubscriptionService
	routerService  *RouterService
}

// NewPushService creates a push service.
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

// ===== Push to a specific agent =====

// PushToAgent pushes to a specific agent.
// It pushes directly to agents on the local node and uses the router service for agents on other nodes.
func (s *PushService) PushToAgent(tenantID, agentID int64, event string, data interface{}) (int, error) {
	// Construct the message.
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// Try local-node delivery first.
	count := s.pushToAgentLocal(tenantID, agentID, msgBytes)

	// Also try cross-node delivery when the router service is available.
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

// pushToAgentLocal pushes to an agent on the local node.
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

// ===== Push to an entire tenant =====

// PushToTenant pushes to all connections for the specified tenant,
// including its agents, administrators, and monitoring dashboards.
func (s *PushService) PushToTenant(tenantID int64, event string, data interface{}, userTypes []string) (int, error) {
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// Local delivery.
	count := s.pushToTenantLocal(tenantID, msgBytes, userTypes)

	// Cross-node delivery through a topic.
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

// pushToTenantLocal pushes to a tenant on the local node.
func (s *PushService) pushToTenantLocal(tenantID int64, msgBytes []byte, userTypes []string) int {
	count := 0
	typeFilter := make(map[string]bool)
	for _, t := range userTypes {
		typeFilter[t] = true
	}
	hasFilter := len(typeFilter) > 0

	// Push to agents.
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

	// Push to administrators with a matching tenant.
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

	// Push to monitoring dashboards with a matching tenant.
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

// ===== Push to a topic =====

// PushToTopic pushes to a specific topic.
// All connections subscribed to the topic receive the message.
func (s *PushService) PushToTopic(topic string, event string, data interface{}) (int, error) {
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// Local delivery.
	count := s.pushToTopicLocal(topic, msgBytes)

	// Cross-node delivery.
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

// pushToTopicLocal pushes to a topic on the local node.
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

// ===== Broadcast =====

// PushToAll broadcasts to all connections.
func (s *PushService) PushToAll(event string, data interface{}) (int, error) {
	msg := model.NewMessage(event, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshal message failed: %w", err)
	}

	// Local broadcast.
	count := s.pushToAllLocal(msgBytes)

	// Cross-node broadcast.
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

// pushToAllLocal broadcasts on the local node.
func (s *PushService) pushToAllLocal(msgBytes []byte) int {
	// Push through the broadcast topic.
	topic := s.subService.BroadcastTopic()
	count := s.pushToTopicLocal(topic, msgBytes)

	// Delivery to every connection could ensure receipt without a broadcast subscription.
	// Topic-based delivery is sufficient here.
	return count
}

// ===== Push system notices =====

// PushSystemNotice pushes a system notice.
func (s *PushService) PushSystemNotice(tenantID int64, notice *model.SystemNoticeData) (int, error) {
	return s.PushToTenant(tenantID, model.EventSystemNotice, notice, nil)
}

// PushError pushes an error notice to a specific connection.
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

// ===== Heartbeat responses =====

// SendHeartbeat sends a heartbeat response to a specific connection.
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
