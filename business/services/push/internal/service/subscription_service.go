package service

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
)

// SubscriptionService manages topic-to-connection mappings.
// It handles subscriptions and unsubscriptions.
// It finds the connections that should receive messages for a topic.
//
// Topic naming conventions:
//   - agent:{agentId}            Agent personal topic (automatic subscription)
//   - tenant:{tenantId}          Tenant broadcast topic (automatic subscription)
//   - queue:{queueId}            Queue monitoring topic
//   - monitor:{type}             Monitoring data topic
//   - broadcast                  Global broadcast topic (automatic subscription)
type SubscriptionService struct {
	// Connection manager.
	connManager *ConnectionManager

	// topic -> set of connection IDs.
	// Enables fast lookup of all connections subscribed to a topic.
	topicSubscribers map[string]map[string]bool

	// Read-write lock.
	mu sync.RWMutex
}

// NewSubscriptionService creates a subscription service.
func NewSubscriptionService(connManager *ConnectionManager) *SubscriptionService {
	return &SubscriptionService{
		connManager:      connManager,
		topicSubscribers: make(map[string]map[string]bool),
	}
}

// ===== Subscribe and unsubscribe =====

// Subscribe subscribes a connection to topics.
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

		// Add the topic to the connection's subscription list.
		conn.Subscribe(topic)

		// Add the connection to the topic's subscriber list.
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

// Unsubscribe unsubscribes a connection from topics.
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

		// Remove the topic from the connection's subscription list.
		conn.Unsubscribe(topic)

		// Remove the connection from the topic's subscriber list.
		if subscribers, ok := s.topicSubscribers[topic]; ok {
			delete(subscribers, connID)
			// Clean up topics with no subscribers.
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

// UnsubscribeAll removes all subscriptions for a connection when it disconnects.
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

// ===== Subscription queries =====

// GetTopicSubscribers returns all connections subscribed to the specified topic.
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

// GetTopicSubscriberCount returns the number of subscribers to a topic.
func (s *SubscriptionService) GetTopicSubscriberCount(topic string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if subscribers, ok := s.topicSubscribers[topic]; ok {
		return len(subscribers)
	}
	return 0
}

// GetAllTopics returns all topics.
func (s *SubscriptionService) GetAllTopics() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int, len(s.topicSubscribers))
	for topic, subscribers := range s.topicSubscribers {
		result[topic] = len(subscribers)
	}
	return result
}

// ===== Automatic subscriptions =====

// AutoSubscribeForAgent automatically subscribes an agent connection to default topics,
// including the personal, tenant, and global broadcast topics.
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

// AutoSubscribeForAdmin automatically subscribes an administrator connection to default topics,
// including the tenant broadcast topic when tenant-scoped and the global broadcast topic.
func (s *SubscriptionService) AutoSubscribeForAdmin(conn *model.Connection) {
	if conn == nil || conn.UserType() != model.UserTypeAdmin {
		return
	}

	var topics []string
	// Subscribe tenant-scoped administrators to their tenant topic by default.
	if conn.TenantID() > 0 {
		topics = append(topics, s.TenantTopic(conn.TenantID()))
	}
	// All administrators subscribe to the global broadcast topic.
	topics = append(topics, s.BroadcastTopic())

	s.Subscribe(conn, topics)
}

// AutoSubscribeForMonitor automatically subscribes a monitoring dashboard connection to default topics,
// including the tenant monitoring and global broadcast topics.
func (s *SubscriptionService) AutoSubscribeForMonitor(conn *model.Connection) {
	if conn == nil || conn.UserType() != model.UserTypeMonitor {
		return
	}

	var topics []string
	// Monitoring dashboards subscribe to the tenant monitoring topic by default.
	if conn.TenantID() > 0 {
		topics = append(topics, s.TenantMonitorTopic(conn.TenantID()))
	}
	// Global broadcast topic.
	topics = append(topics, s.BroadcastTopic())

	s.Subscribe(conn, topics)
}

// ===== Topic name generation =====

// AgentTopic generates an agent topic name.
func (s *SubscriptionService) AgentTopic(agentID int64) string {
	return fmt.Sprintf("%s%d", model.TopicAgentPrefix, agentID)
}

// TenantTopic generates a tenant topic name.
func (s *SubscriptionService) TenantTopic(tenantID int64) string {
	return fmt.Sprintf("%s%d", model.TopicTenantPrefix, tenantID)
}

// QueueTopic generates a queue topic name.
func (s *SubscriptionService) QueueTopic(queueID int64) string {
	return fmt.Sprintf("%s%d", model.TopicQueuePrefix, queueID)
}

// MonitorTopic generates a monitoring topic name.
func (s *SubscriptionService) MonitorTopic(monitorType string) string {
	return fmt.Sprintf("%s%s", model.TopicMonitorPrefix, monitorType)
}

// TenantMonitorTopic generates a tenant monitoring topic name.
func (s *SubscriptionService) TenantMonitorTopic(tenantID int64) string {
	return fmt.Sprintf("%stenant:%d", model.TopicMonitorPrefix, tenantID)
}

// BroadcastTopic returns the broadcast topic name.
func (s *SubscriptionService) BroadcastTopic() string {
	return model.TopicBroadcast
}

// ===== Topic matching =====

// IsValidTopic reports whether a topic name is valid.
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
		// Content is required after the prefix.
		parts := strings.SplitN(topic, ":", 2)
		return len(parts) == 2 && parts[1] != ""
	}
	return false
}
