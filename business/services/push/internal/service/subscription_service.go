package service

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
)

// SubscriptionService 订阅服务
// 管理主题与连接的映射关系，负责订阅/取消订阅操作
// 以及根据主题查找需要推送的连接
//
// 主题命名规范：
//   - agent:{agentId}            座席个人主题（自动订阅）
//   - tenant:{tenantId}          租户广播主题（自动订阅）
//   - queue:{queueId}            队列监控主题
//   - monitor:{type}             监控数据主题
//   - broadcast                  全局广播主题（自动订阅）
type SubscriptionService struct {
	// 连接管理器
	connManager *ConnectionManager

	// 主题 -> 连接ID 集合
	// 用于快速查找订阅了某个主题的所有连接
	topicSubscribers map[string]map[string]bool

	// 读写锁
	mu sync.RWMutex
}

// NewSubscriptionService 创建订阅服务
func NewSubscriptionService(connManager *ConnectionManager) *SubscriptionService {
	return &SubscriptionService{
		connManager:      connManager,
		topicSubscribers: make(map[string]map[string]bool),
	}
}

// ===== 订阅/取消订阅 =====

// Subscribe 连接订阅主题
func (s *SubscriptionService) Subscribe(conn *model.Connection, topics []string) {
	if conn == nil || len(topics) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connID := conn.ConnID()
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}

		// 添加到连接的订阅列表
		conn.Subscribe(topic)

		// 添加到主题的订阅者列表
		if _, ok := s.topicSubscribers[topic]; !ok {
			s.topicSubscribers[topic] = make(map[string]bool)
		}
		s.topicSubscribers[topic][connID] = true
	}

	logger.Debug("subscribe topics",
		zap.String("conn_id", connID),
		zap.Strings("topics", topics),
		zap.Int("total_subs", conn.SubscriptionCount()),
	)
}

// Unsubscribe 连接取消订阅主题
func (s *SubscriptionService) Unsubscribe(conn *model.Connection, topics []string) {
	if conn == nil || len(topics) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connID := conn.ConnID()
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}

		// 从连接的订阅列表移除
		conn.Unsubscribe(topic)

		// 从主题的订阅者列表移除
		if subscribers, ok := s.topicSubscribers[topic]; ok {
			delete(subscribers, connID)
			// 如果主题没有订阅者了，清理
			if len(subscribers) == 0 {
				delete(s.topicSubscribers, topic)
			}
		}
	}

	logger.Debug("unsubscribe topics",
		zap.String("conn_id", connID),
		zap.Strings("topics", topics),
		zap.Int("total_subs", conn.SubscriptionCount()),
	)
}

// UnsubscribeAll 取消连接的所有订阅（连接断开时调用）
func (s *SubscriptionService) UnsubscribeAll(conn *model.Connection) {
	if conn == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connID := conn.ConnID()
	topics := conn.GetSubscriptions()

	for _, topic := range topics {
		if subscribers, ok := s.topicSubscribers[topic]; ok {
			delete(subscribers, connID)
			if len(subscribers) == 0 {
				delete(s.topicSubscribers, topic)
			}
		}
	}

	logger.Debug("unsubscribe all topics",
		zap.String("conn_id", connID),
		zap.Int("count", len(topics)),
	)
}

// ===== 订阅查询 =====

// GetTopicSubscribers 获取订阅了指定主题的所有连接
func (s *SubscriptionService) GetTopicSubscribers(topic string) []*model.Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subscriberIDs, ok := s.topicSubscribers[topic]
	if !ok || len(subscriberIDs) == 0 {
		return nil
	}

	result := make([]*model.Connection, 0, len(subscriberIDs))
	for connID := range subscriberIDs {
		if conn, exists := s.connManager.GetConnection(connID); exists {
			if conn.IsConnected() {
				result = append(result, conn)
			}
		}
	}
	return result
}

// GetTopicSubscriberCount 获取主题订阅数
func (s *SubscriptionService) GetTopicSubscriberCount(topic string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if subscribers, ok := s.topicSubscribers[topic]; ok {
		return len(subscribers)
	}
	return 0
}

// GetAllTopics 获取所有主题
func (s *SubscriptionService) GetAllTopics() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int, len(s.topicSubscribers))
	for topic, subscribers := range s.topicSubscribers {
		result[topic] = len(subscribers)
	}
	return result
}

// ===== 自动订阅 =====

// AutoSubscribeForAgent 为座席连接自动订阅默认主题
// 包括：个人主题、租户主题、全局广播主题
func (s *SubscriptionService) AutoSubscribeForAgent(conn *model.Connection) {
	if conn == nil || conn.UserType() != model.UserTypeAgent {
		return
	}

	topics := []string{
		s.AgentTopic(conn.UserID()),
		s.TenantTopic(conn.TenantID()),
		s.BroadcastTopic(),
	}

	s.Subscribe(conn, topics)
}

// AutoSubscribeForAdmin 为管理员连接自动订阅默认主题
// 包括：租户广播主题（如果有租户限制）、全局广播主题
func (s *SubscriptionService) AutoSubscribeForAdmin(conn *model.Connection) {
	if conn == nil || conn.UserType() != model.UserTypeAdmin {
		return
	}

	var topics []string
	// 管理员默认订阅租户主题（如果有租户限制）
	if conn.TenantID() > 0 {
		topics = append(topics, s.TenantTopic(conn.TenantID()))
	}
	// 全局广播主题（所有管理员都订阅）
	topics = append(topics, s.BroadcastTopic())

	s.Subscribe(conn, topics)
}

// AutoSubscribeForMonitor 为监控大屏连接自动订阅默认主题
// 包括：租户监控主题、全局广播主题
func (s *SubscriptionService) AutoSubscribeForMonitor(conn *model.Connection) {
	if conn == nil || conn.UserType() != model.UserTypeMonitor {
		return
	}

	var topics []string
	// 监控大屏默认订阅租户监控主题
	if conn.TenantID() > 0 {
		topics = append(topics, s.TenantMonitorTopic(conn.TenantID()))
	}
	// 全局广播主题
	topics = append(topics, s.BroadcastTopic())

	s.Subscribe(conn, topics)
}

// ===== 主题名称生成 =====

// AgentTopic 生成座席主题名
func (s *SubscriptionService) AgentTopic(agentID int64) string {
	return fmt.Sprintf("%s%d", model.TopicAgentPrefix, agentID)
}

// TenantTopic 生成租户主题名
func (s *SubscriptionService) TenantTopic(tenantID int64) string {
	return fmt.Sprintf("%s%d", model.TopicTenantPrefix, tenantID)
}

// QueueTopic 生成队列主题名
func (s *SubscriptionService) QueueTopic(queueID int64) string {
	return fmt.Sprintf("%s%d", model.TopicQueuePrefix, queueID)
}

// MonitorTopic 生成监控主题名
func (s *SubscriptionService) MonitorTopic(monitorType string) string {
	return fmt.Sprintf("%s%s", model.TopicMonitorPrefix, monitorType)
}

// TenantMonitorTopic 生成租户监控主题名
func (s *SubscriptionService) TenantMonitorTopic(tenantID int64) string {
	return fmt.Sprintf("%stenant:%d", model.TopicMonitorPrefix, tenantID)
}

// BroadcastTopic 广播主题名
func (s *SubscriptionService) BroadcastTopic() string {
	return model.TopicBroadcast
}

// ===== 主题匹配 =====

// IsValidTopic 检查主题名是否合法
func (s *SubscriptionService) IsValidTopic(topic string) bool {
	if topic == "" {
		return false
	}
	if topic == model.TopicBroadcast {
		return true
	}
	if strings.HasPrefix(topic, model.TopicAgentPrefix) ||
		strings.HasPrefix(topic, model.TopicTenantPrefix) ||
		strings.HasPrefix(topic, model.TopicQueuePrefix) ||
		strings.HasPrefix(topic, model.TopicMonitorPrefix) {
		// 前缀后必须有内容
		parts := strings.SplitN(topic, ":", 2)
		return len(parts) == 2 && parts[1] != ""
	}
	return false
}
