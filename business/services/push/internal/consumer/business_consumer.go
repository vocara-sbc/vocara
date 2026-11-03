package consumer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"vocara/pkg/config"
	pkgredis "vocara/pkg/redis"
	"vocara/pkg/eventbus"
	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
	"vocara/services/push/internal/service"
)

// BusinessConsumer consumes business events published by business services through the event bus.
// These events include agent status, call, queue, and system notice events.
// It routes them to WebSocket connections by event type.
//
// Differences from NATSConsumer and RedisConsumer:
// - Those consumers receive "push commands" through vocara.ws.push.* topics, where other services specify the recipients.
// - This consumer receives "business events" such as agent.status_change and call.ringing;
//   the push service determines the recipients from the event type.
type BusinessConsumer struct {
	pushService *service.PushService
	subService  *service.SubscriptionService

	// Backend type: nats / redis.
	backend string

	// NATS state.
	natsConfig config.NATSConfig
	natsConn   *nats.Conn

	// Redis state.
	redisClient *redis.Client
	redisPubsub *redis.PubSub

	// Stop state.
	stopCh chan struct{}
	once   sync.Once
}

// NewBusinessConsumerWithNATS creates a business event consumer that uses NATS.
func NewBusinessConsumerWithNATS(
	pushService *service.PushService,
	subService *service.SubscriptionService,
	natsConfig config.NATSConfig,
) *BusinessConsumer {
	return &BusinessConsumer{
		pushService: pushService,
		subService:  subService,
		backend:     "nats",
		natsConfig:  natsConfig,
		stopCh:      make(chan struct{}),
	}
}

// NewBusinessConsumerWithRedis creates a business event consumer that uses Redis.
func NewBusinessConsumerWithRedis(
	pushService *service.PushService,
	subService *service.SubscriptionService,
	redisClient *redis.Client,
) *BusinessConsumer {
	return &BusinessConsumer{
		pushService: pushService,
		subService:  subService,
		backend:     "redis",
		redisClient: redisClient,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the business event consumer asynchronously without blocking.
func (c *BusinessConsumer) Start(ctx context.Context) error {
	if c.backend == "nats" {
		return c.startNATS(ctx)
	}
	return c.startRedis(ctx)
}

// Stop stops the consumer.
func (c *BusinessConsumer) Stop() {
	c.once.Do(func() {
		close(c.stopCh)

		if c.backend == "nats" && c.natsConn != nil {
			c.natsConn.Close()
			c.natsConn = nil
		}
		if c.backend == "redis" && c.redisPubsub != nil {
			_ = c.redisPubsub.Close()
			c.redisPubsub = nil
		}

		logger.Info("business consumer stopped", zap.String("backend", c.backend))
	})
}

// ===== NATS implementation =====

func (c *BusinessConsumer) startNATS(ctx context.Context) error {
	// Connect to NATS.
	urls := c.natsConfig.URLs
	if urls == "" {
		urls = nats.DefaultURL
	}

	opts := []nats.Option{
		nats.Name("vocara-push-business-consumer"),
		nats.MaxReconnects(c.natsConfig.MaxReconnect),
		nats.ReconnectWait(time.Duration(c.natsConfig.ReconnectWait) * time.Second),
		nats.Timeout(time.Duration(c.natsConfig.Timeout) * time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			logger.Warn("business consumer nats disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			logger.Info("business consumer nats reconnected", zap.String("url", conn.ConnectedUrl()))
		}),
	}

	if c.natsConfig.User != "" {
		opts = append(opts, nats.UserInfo(c.natsConfig.User, c.natsConfig.Password))
	}
	if c.natsConfig.Token != "" {
		opts = append(opts, nats.Token(c.natsConfig.Token))
	}

	conn, err := nats.Connect(urls, opts...)
	if err != nil {
		return fmt.Errorf("business consumer nats connect failed: %w", err)
	}
	c.natsConn = conn

	// Use the eventbus package's NATSSubscriber to subscribe to all events.
	subscriber := eventbus.NewNATSSubscriber(conn)
	_, err = subscriber.SubscribeAll(ctx, func(_ context.Context, event *eventbus.Event) error {
		c.handleEvent(event)
		return nil
	})
	if err != nil {
		conn.Close()
		return fmt.Errorf("business consumer subscribe all failed: %w", err)
	}

	logger.Info("business consumer started (nats)",
		zap.Strings("urls", c.natsConfig.URLList()),
	)

	// Listen for the stop signal.
	go func() {
		select {
		case <-ctx.Done():
		case <-c.stopCh:
		}
		c.Stop()
	}()

	return nil
}

// ===== Redis implementation =====

func (c *BusinessConsumer) startRedis(ctx context.Context) error {
	if c.redisClient == nil {
		return fmt.Errorf("redis client is nil")
	}

	// Use the eventbus package's RedisSubscriber to subscribe to all events.
	subscriber := eventbus.NewRedisSubscriber(c.redisClient)
	err := subscriber.SubscribeAll(ctx, func(_ context.Context, event *eventbus.Event) error {
		c.handleEvent(event)
		return nil
	})
	if err != nil {
		return fmt.Errorf("business consumer redis subscribe all failed: %w", err)
	}

	logger.Info("business consumer started (redis)")

	// Listen for the stop signal.
	go func() {
		select {
		case <-ctx.Done():
		case <-c.stopCh:
		}
		c.Stop()
	}()

	return nil
}

// ===== Core event handling logic =====

// handleEvent processes a business event and routes it to WebSocket connections by event type.
func (c *BusinessConsumer) handleEvent(event *eventbus.Event) {
	if event == nil {
		return
	}

	eventType := event.EventType
	tenantID := int64(event.TenantID)
	data := event.Data

	logger.Debug("business consumer received event",
		zap.String("event_type", eventType),
		zap.String("event_id", event.EventID),
		zap.Uint64("tenant_id", event.TenantID),
	)

	// Construct the WebSocket message while preserving the original event_id and timestamp.
	msg := model.NewMessageWithID(eventType, event.EventID, event.Timestamp, data)

	switch {
	// ===== Agent events =====
	case strings.HasPrefix(eventType, "agent."):
		c.handleAgentEvent(tenantID, msg, data)

	// ===== Call events =====
	case strings.HasPrefix(eventType, "call."):
		c.handleCallEvent(tenantID, msg, data)

	// ===== Queue events =====
	case strings.HasPrefix(eventType, "queue."):
		c.handleQueueEvent(tenantID, msg, data)

	// ===== System notices =====
	case eventType == eventbus.EventSystemNotice:
		c.handleSystemNotice(msg)

	default:
		// Ignore other event types; support can be extended as needed.
		logger.Debug("business consumer: skip unknown event type",
			zap.String("event_type", eventType),
		)
	}
}

// handleAgentEvent processes agent-related events.
// It pushes to the agent, administrators, and monitoring dashboards.
func (c *BusinessConsumer) handleAgentEvent(tenantID int64, msg *model.Message, data interface{}) {
	// 1. Push to the agent.
	agentID := extractIntField(data, "agent_id")
	if agentID > 0 && tenantID > 0 {
		_, _ = c.pushService.PushToAgent(tenantID, agentID, msg.Event, msg.Data)
	}

	// 2. Push to tenant administrators and monitoring dashboards through PushToTenant with user types specified.
	if tenantID > 0 {
		userTypes := []string{model.UserTypeAdmin, model.UserTypeMonitor}
		_, _ = c.pushService.PushToTenant(tenantID, msg.Event, msg.Data, userTypes)
	}
}

// handleCallEvent processes call-related events.
// It pushes to the agent, administrators, and monitoring dashboards.
func (c *BusinessConsumer) handleCallEvent(tenantID int64, msg *model.Message, data interface{}) {
	// 1. Push to the agent.
	agentID := extractIntField(data, "agent_id")
	if agentID > 0 && tenantID > 0 {
		_, _ = c.pushService.PushToAgent(tenantID, agentID, msg.Event, msg.Data)
	}

	// 2. Push to tenant administrators and monitoring dashboards.
	if tenantID > 0 {
		userTypes := []string{model.UserTypeAdmin, model.UserTypeMonitor}
		_, _ = c.pushService.PushToTenant(tenantID, msg.Event, msg.Data, userTypes)
	}
}

// handleQueueEvent processes queue-related events.
// Queue events fall into two categories:
//   - Single-call queue events (queue.enqueue / queue.dequeue / queue.abandoned / queue.timeout):
//     data is CallQueueEventData and describes a call entering or leaving a queue, being abandoned, or timing out.
//   - Aggregate queue statistics events (queue.stats_update):
//     data is QueueStatusData with aggregate statistics such as waiting callers, agent counts, and SLA.
//
// It pushes to the queue monitoring topic (queue:{queue_id}) and tenant administrators.
func (c *BusinessConsumer) handleQueueEvent(tenantID int64, msg *model.Message, data interface{}) {
	queueID := extractIntField(data, "queue_id")

	// 1. Push to the queue monitoring topic (queue:{queue_id}).
	//    Forward the complete CallQueueEventData or QueueStatusData payload.
	if queueID > 0 {
		queueTopic := c.subService.QueueTopic(queueID)
		_, _ = c.pushService.PushToTopic(queueTopic, msg.Event, msg.Data)
	}

	// 2. Push to tenant administrators.
	if tenantID > 0 {
		userTypes := []string{model.UserTypeAdmin}
		_, _ = c.pushService.PushToTenant(tenantID, msg.Event, msg.Data, userTypes)
	}
}

// handleSystemNotice processes a system notice.
// It broadcasts the notice to everyone.
func (c *BusinessConsumer) handleSystemNotice(msg *model.Message) {
	_, _ = c.pushService.PushToAll(msg.Event, msg.Data)
}

// ===== Helper methods =====

// extractIntField extracts an int64 field from data.
// It supports float64 for JSON-decoded numbers, int64, int, and string.
func extractIntField(data interface{}, field string) int64 {
	if data == nil {
		return 0
	}

	m, ok := data.(map[string]interface{})
	if !ok {
		return 0
	}

	v, exists := m[field]
	if !exists || v == nil {
		return 0
	}

	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case uint64:
		return int64(val)
	case string:
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			return id
		}
	}
	return 0
}

// ===== Helper to obtain the underlying *redis.Client from pkgredis.Client =====
// pkgredis.Client.Raw() returns UniversalClient.
// In standalone mode, the concrete value is *redis.Client and requires a type assertion.

// GetRedisClientFromPkg returns the underlying *redis.Client from pkgredis.Client.
// It returns nil in cluster mode because Pub/Sub behavior differs and must be handled by the caller.
func GetRedisClientFromPkg(pkgClient *pkgredis.Client) *redis.Client {
	if pkgClient == nil {
		return nil
	}
	raw := pkgClient.Raw()
	if client, ok := raw.(*redis.Client); ok {
		return client
	}
	// ClusterClient and SentinelClient also support Pub/Sub,
	// but eventbus.RedisSubscriber requires *redis.Client.
	// Production deployments should use NATS as the event bus, with Redis only as a fallback.
	logger.Warn("pkg redis client is not *redis.Client (maybe cluster mode), " +
		"business consumer redis backend may not work properly")
	return nil
}
