package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	pkgredis "vocara/pkg/redis"
	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
	"vocara/services/push/internal/service"
)

// RedisConsumer consumes push messages through Redis Pub/Sub.
// It is an alternative to the NATS consumer.
// It suits simple deployments with moderate message volumes.
//
// Subscribed channels:
// - vocara:push:agent:*       Push to a specific agent
// - vocara:push:tenant:*      Push to a specific tenant
// - vocara:push:topic:*       Push to a specific topic
// - vocara:push:broadcast     Global broadcast
type RedisConsumer struct {
	connManager *service.ConnectionManager
	subService  *service.SubscriptionService
	pushService *service.PushService

	redisClient *pkgredis.Client

	// Subscription client.
	pubsub *redis.PubSub

	// Stop state.
	stopCh chan struct{}
	once   sync.Once
}

// NewRedisConsumer creates a Redis Pub/Sub consumer.
func NewRedisConsumer(
	connManager *service.ConnectionManager,
	subService *service.SubscriptionService,
	pushService *service.PushService,
	redisClient *pkgredis.Client,
) *RedisConsumer {
	return &RedisConsumer{
		connManager: connManager,
		subService:  subService,
		pushService: pushService,
		redisClient: redisClient,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the Redis consumer.
func (r *RedisConsumer) Start(ctx context.Context) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client is nil")
	}

	// Use PSUBSCRIBE to subscribe to channels matching the patterns.
	patterns := []string{
		"vocara:push:agent:*",
		"vocara:push:tenant:*",
		"vocara:push:topic:*",
		"vocara:push:broadcast",
	}

	r.pubsub = r.redisClient.Raw().PSubscribe(ctx, patterns...)

	// Confirm the subscription.
	_, err := r.pubsub.Receive(ctx)
	if err != nil {
		_ = r.pubsub.Close()
		return fmt.Errorf("redis psubscribe failed: %w", err)
	}

	logger.Info("redis pubsub consumer started",
		zap.Strings("patterns", patterns),
	)

	// Start the message processing loop.
	go r.messageLoop(ctx)

	return nil
}

// Stop stops the Redis consumer.
func (r *RedisConsumer) Stop() {
	r.once.Do(func() {
		close(r.stopCh)
		if r.pubsub != nil {
			_ = r.pubsub.Close()
			r.pubsub = nil
		}
		logger.Info("redis pubsub consumer stopped")
	})
}

// messageLoop processes messages in a loop.
func (r *RedisConsumer) messageLoop(ctx context.Context) {
	ch := r.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			logger.Info("redis consumer stopped by context")
			return
		case <-r.stopCh:
			return
		case msg, ok := <-ch:
			if !ok {
				logger.Warn("redis pubsub channel closed")
				return
			}
			if msg == nil {
				continue
			}
			r.handleMessage(msg.Channel, []byte(msg.Payload))
		}
	}
}

// handleMessage processes a received message.
func (r *RedisConsumer) handleMessage(channel string, data []byte) {
	logger.Debug("received redis pubsub message",
		zap.String("channel", channel),
	)

	// Parse the message.
	var pushMsg model.Message
	if err := json.Unmarshal(data, &pushMsg); err != nil {
		logger.Error("unmarshal redis push message failed",
			zap.String("channel", channel),
			zap.Error(err),
		)
		return
	}

	// Handle the message by channel type.
	switch {
	case strings.HasPrefix(channel, "vocara:push:agent:"):
		r.handleAgentMessage(channel, &pushMsg)

	case strings.HasPrefix(channel, "vocara:push:tenant:"):
		r.handleTenantMessage(channel, &pushMsg)

	case strings.HasPrefix(channel, "vocara:push:topic:"):
		r.handleTopicMessage(channel, &pushMsg)

	case channel == "vocara:push:broadcast":
		r.handleBroadcastMessage(&pushMsg)

	default:
		logger.Warn("unknown channel", zap.String("channel", channel))
	}
}

// handleAgentMessage processes an agent push message.
func (r *RedisConsumer) handleAgentMessage(channel string, msg *model.Message) {
	agentIDStr := strings.TrimPrefix(channel, "vocara:push:agent:")
	agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
	if err != nil || agentID <= 0 {
		logger.Warn("invalid agent id in channel",
			zap.String("channel", channel),
			zap.String("agent_id", agentIDStr),
		)
		return
	}

	logger.Debug("redis push to agent",
		zap.Int64("agent_id", agentID),
		zap.String("event", msg.Event),
	)

	// Extract tenantId from the message data.
	tenantID := extractTenantIDFromData(msg.Data)

	// Push through the push service.
	_, err = r.pushService.PushToAgent(tenantID, agentID, msg.Event, msg.Data)
	if err != nil {
		logger.Error("push to agent failed",
			zap.Int64("agent_id", agentID),
			zap.Error(err),
		)
	}
}

// handleTenantMessage processes a tenant push message.
func (r *RedisConsumer) handleTenantMessage(channel string, msg *model.Message) {
	tenantIDStr := strings.TrimPrefix(channel, "vocara:push:tenant:")
	tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64)
	if err != nil || tenantID <= 0 {
		logger.Warn("invalid tenant id in channel",
			zap.String("channel", channel),
			zap.String("tenant_id", tenantIDStr),
		)
		return
	}

	logger.Debug("redis push to tenant",
		zap.Int64("tenant_id", tenantID),
		zap.String("event", msg.Event),
	)

	_, err = r.pushService.PushToTenant(tenantID, msg.Event, msg.Data, nil)
	if err != nil {
		logger.Error("push to tenant failed",
			zap.Int64("tenant_id", tenantID),
			zap.Error(err),
		)
	}
}

// handleTopicMessage processes a topic push message.
func (r *RedisConsumer) handleTopicMessage(channel string, msg *model.Message) {
	topic := strings.TrimPrefix(channel, "vocara:push:topic:")
	if topic == "" {
		logger.Warn("empty topic in channel", zap.String("channel", channel))
		return
	}

	logger.Debug("redis push to topic",
		zap.String("topic", topic),
		zap.String("event", msg.Event),
	)

	_, err := r.pushService.PushToTopic(topic, msg.Event, msg.Data)
	if err != nil {
		logger.Error("push to topic failed",
			zap.String("topic", topic),
			zap.Error(err),
		)
	}
}

// handleBroadcastMessage processes a broadcast message.
func (r *RedisConsumer) handleBroadcastMessage(msg *model.Message) {
	logger.Debug("redis push broadcast",
		zap.String("event", msg.Event),
	)

	_, err := r.pushService.PushToAll(msg.Event, msg.Data)
	if err != nil {
		logger.Error("push to all failed", zap.Error(err))
	}
}
