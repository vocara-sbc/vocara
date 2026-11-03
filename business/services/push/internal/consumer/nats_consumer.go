// Package consumer provides message consumers.
// It consumes push messages from NATS or Redis Pub/Sub and forwards them to local WebSocket connections.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"vocara/pkg/config"
	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
	"vocara/services/push/internal/service"
)

// NATSConsumer consumes messages from NATS.
// It subscribes to push-related topics and pushes received messages to local WebSocket connections.
//
// Subscribed topics:
// - vocara.ws.push.agent.{agentId}   Push to a specific agent
// - vocara.ws.push.tenant.{tenantId} Push to a specific tenant
// - vocara.ws.push.topic.{topic}     Push to a specific topic
// - vocara.ws.push.broadcast         Global broadcast
//
// This consumer differs from the RouterService consumer.
// RouterService routes messages between nodes, while this consumer receives push events from business services
// such as the agent and call services.
type NATSConsumer struct {
	connManager *service.ConnectionManager
	subService  *service.SubscriptionService
	pushService *service.PushService
	natsConfig  config.NATSConfig

	conn *nats.Conn

	// Subscriptions used for graceful shutdown.
	subs []*nats.Subscription

	// Stop state.
	stopCh chan struct{}
	once   sync.Once
}

// NewNATSConsumer creates a NATS consumer.
func NewNATSConsumer(
	connManager *service.ConnectionManager,
	subService *service.SubscriptionService,
	pushService *service.PushService,
	natsConfig config.NATSConfig,
) *NATSConsumer {
	return &NATSConsumer{
		connManager: connManager,
		subService:  subService,
		pushService: pushService,
		natsConfig:  natsConfig,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the NATS consumer.
func (n *NATSConsumer) Start(ctx context.Context) error {
	// Connect to NATS.
	if err := n.connect(); err != nil {
		return fmt.Errorf("connect nats failed: %w", err)
	}

	// Subscribe to all topics.
	if err := n.subscribeAll(); err != nil {
		// Close the connection.
		n.conn.Close()
		return fmt.Errorf("subscribe nats topics failed: %w", err)
	}

	logger.Info("nats consumer started",
		zap.Strings("urls", n.natsConfig.URLList()),
	)

	// Listen for the stop signal.
	go func() {
		select {
		case <-ctx.Done():
		case <-n.stopCh:
		}
		n.stop()
	}()

	return nil
}

// Stop stops the NATS consumer.
func (n *NATSConsumer) Stop() {
	n.once.Do(func() {
		close(n.stopCh)
	})
}

// stop performs the actual shutdown.
func (n *NATSConsumer) stop() {
	// Cancel all subscriptions.
	for _, sub := range n.subs {
		_ = sub.Unsubscribe()
	}
	n.subs = nil

	// Close the connection.
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}

	logger.Info("nats consumer stopped")
}

// connect connects to NATS.
func (n *NATSConsumer) connect() error {
	urls := n.natsConfig.URLs
	if urls == "" {
		urls = nats.DefaultURL
	}

	opts := []nats.Option{
		nats.Name("vocara-push-consumer"),
		nats.MaxReconnects(n.natsConfig.MaxReconnect),
		nats.ReconnectWait(time.Duration(n.natsConfig.ReconnectWait) * time.Second),
		nats.Timeout(time.Duration(n.natsConfig.Timeout) * time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			logger.Warn("nats disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			logger.Info("nats reconnected", zap.String("url", conn.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			logger.Info("nats connection closed")
		}),
	}

	// User authentication.
	if n.natsConfig.User != "" {
		opts = append(opts, nats.UserInfo(n.natsConfig.User, n.natsConfig.Password))
	}
	if n.natsConfig.Token != "" {
		opts = append(opts, nats.Token(n.natsConfig.Token))
	}

	var err error
	n.conn, err = nats.Connect(urls, opts...)
	if err != nil {
		return fmt.Errorf("nats connect failed: %w", err)
	}

	return nil
}

// subscribeAll subscribes to all topics.
func (n *NATSConsumer) subscribeAll() error {
	// Agent push topic using a wildcard.
	sub, err := n.conn.Subscribe("vocara.ws.push.agent.*", n.handleAgentPush)
	if err != nil {
		return fmt.Errorf("subscribe agent push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	// Tenant push topic.
	sub, err = n.conn.Subscribe("vocara.ws.push.tenant.*", n.handleTenantPush)
	if err != nil {
		return fmt.Errorf("subscribe tenant push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	// Topic push.
	sub, err = n.conn.Subscribe("vocara.ws.push.topic.>", n.handleTopicPush)
	if err != nil {
		return fmt.Errorf("subscribe topic push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	// Broadcast.
	sub, err = n.conn.Subscribe("vocara.ws.push.broadcast", n.handleBroadcastPush)
	if err != nil {
		return fmt.Errorf("subscribe broadcast push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	return nil
}

// ===== Message handling =====

// handleAgentPush processes an agent push message.
func (n *NATSConsumer) handleAgentPush(msg *nats.Msg) {
	// Extract agentId from the topic.
	agentIDStr := extractLastPart(msg.Subject)
	agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
	if err != nil || agentID <= 0 {
		logger.Warn("invalid agent push subject",
			zap.String("subject", msg.Subject),
			zap.String("agent_id", agentIDStr),
		)
		return
	}

	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal agent push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received agent push from nats",
		zap.Int64("agent_id", agentID),
		zap.String("event", pushMsg.Event),
	)

	// Push through the push service, which handles local and cross-node delivery.
	// Try to extract tenantId from the message data.
	tenantID := extractTenantIDFromData(pushMsg.Data)

	_, err = n.pushService.PushToAgent(tenantID, agentID, pushMsg.Event, pushMsg.Data)
	if err != nil {
		logger.Error("push to agent failed",
			zap.Int64("agent_id", agentID),
			zap.Error(err),
		)
	}
}

// handleTenantPush processes a tenant push message.
func (n *NATSConsumer) handleTenantPush(msg *nats.Msg) {
	tenantIDStr := extractLastPart(msg.Subject)
	tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64)
	if err != nil || tenantID <= 0 {
		logger.Warn("invalid tenant push subject",
			zap.String("subject", msg.Subject),
			zap.String("tenant_id", tenantIDStr),
		)
		return
	}

	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal tenant push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received tenant push from nats",
		zap.Int64("tenant_id", tenantID),
		zap.String("event", pushMsg.Event),
	)

	// Push to the tenant through the push service.
	_, err = n.pushService.PushToTenant(tenantID, pushMsg.Event, pushMsg.Data, nil)
	if err != nil {
		logger.Error("push to tenant failed",
			zap.Int64("tenant_id", tenantID),
			zap.Error(err),
		)
	}
}

// handleTopicPush processes a topic push message.
func (n *NATSConsumer) handleTopicPush(msg *nats.Msg) {
	// Topics can contain multiple path segments; extract everything after vocara.ws.push.topic.
	topic := strings.TrimPrefix(msg.Subject, "vocara.ws.push.topic.")
	if topic == "" {
		logger.Warn("invalid topic push subject", zap.String("subject", msg.Subject))
		return
	}

	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal topic push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received topic push from nats",
		zap.String("topic", topic),
		zap.String("event", pushMsg.Event),
	)

	_, err := n.pushService.PushToTopic(topic, pushMsg.Event, pushMsg.Data)
	if err != nil {
		logger.Error("push to topic failed",
			zap.String("topic", topic),
			zap.Error(err),
		)
	}
}

// handleBroadcastPush processes a broadcast message.
func (n *NATSConsumer) handleBroadcastPush(msg *nats.Msg) {
	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal broadcast push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received broadcast push from nats",
		zap.String("event", pushMsg.Event),
	)

	_, err := n.pushService.PushToAll(pushMsg.Event, pushMsg.Data)
	if err != nil {
		logger.Error("push to all failed", zap.Error(err))
	}
}

// ===== Helper methods =====

// extractLastPart extracts the last part of a topic.
func extractLastPart(subject string) string {
	idx := strings.LastIndex(subject, ".")
	if idx < 0 || idx == len(subject)-1 {
		return ""
	}
	return subject[idx+1:]
}

// extractTenantIDFromData extracts tenantId from message data.
func extractTenantIDFromData(data interface{}) int64 {
	if data == nil {
		return 0
	}

	// Try to read it from a map.
	if m, ok := data.(map[string]interface{}); ok {
		if tid, exists := m["tenant_id"]; exists {
			switch v := tid.(type) {
			case float64:
				return int64(v)
			case int64:
				return v
			case int:
				return int64(v)
			case string:
				if id, err := strconv.ParseInt(v, 10, 64); err == nil {
					return id
				}
			}
		}
	}

	return 0
}
