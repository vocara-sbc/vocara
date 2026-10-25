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

// BusinessConsumer 业务事件消费者
// 消费业务服务通过事件总线发布的业务事件（座席状态、通话事件、队列事件、系统通知等），
// 根据事件类型路由到对应的 WebSocket 连接。
//
// 与现有 NATSConsumer / RedisConsumer 的区别：
// - 现有消费者消费的是"推送指令"（其他服务直接指定推送给谁），通过 vocara.ws.push.* 主题
// - 本消费者消费的是"业务事件"（agent.status_change / call.ringing 等），
//   由 push 服务自身根据事件类型决定推送目标
type BusinessConsumer struct {
	pushService *service.PushService
	subService  *service.SubscriptionService

	// 后端类型: nats / redis
	backend string

	// NATS 相关
	natsConfig config.NATSConfig
	natsConn   *nats.Conn

	// Redis 相关
	redisClient *redis.Client
	redisPubsub *redis.PubSub

	// 停止相关
	stopCh chan struct{}
	once   sync.Once
}

// NewBusinessConsumerWithNATS 使用 NATS 创建业务事件消费者
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

// NewBusinessConsumerWithRedis 使用 Redis 创建业务事件消费者
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

// Start 启动业务事件消费者（异步，非阻塞）
func (c *BusinessConsumer) Start(ctx context.Context) error {
	if c.backend == "nats" {
		return c.startNATS(ctx)
	}
	return c.startRedis(ctx)
}

// Stop 停止消费者
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

// ===== NATS 实现 =====

func (c *BusinessConsumer) startNATS(ctx context.Context) error {
	// 连接 NATS
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

	// 使用 eventbus 包的 NATSSubscriber 订阅所有事件
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

	// 监听停止信号
	go func() {
		select {
		case <-ctx.Done():
		case <-c.stopCh:
		}
		c.Stop()
	}()

	return nil
}

// ===== Redis 实现 =====

func (c *BusinessConsumer) startRedis(ctx context.Context) error {
	if c.redisClient == nil {
		return fmt.Errorf("redis client is nil")
	}

	// 使用 eventbus 包的 RedisSubscriber 订阅所有事件
	subscriber := eventbus.NewRedisSubscriber(c.redisClient)
	err := subscriber.SubscribeAll(ctx, func(_ context.Context, event *eventbus.Event) error {
		c.handleEvent(event)
		return nil
	})
	if err != nil {
		return fmt.Errorf("business consumer redis subscribe all failed: %w", err)
	}

	logger.Info("business consumer started (redis)")

	// 监听停止信号
	go func() {
		select {
		case <-ctx.Done():
		case <-c.stopCh:
		}
		c.Stop()
	}()

	return nil
}

// ===== 事件处理核心逻辑 =====

// handleEvent 处理业务事件，根据事件类型路由到对应的 WebSocket 连接
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

	// 构建 WebSocket 消息（保持原始 event_id 和 timestamp）
	msg := model.NewMessageWithID(eventType, event.EventID, event.Timestamp, data)

	switch {
	// ===== 座席相关事件 =====
	case strings.HasPrefix(eventType, "agent."):
		c.handleAgentEvent(tenantID, msg, data)

	// ===== 通话相关事件 =====
	case strings.HasPrefix(eventType, "call."):
		c.handleCallEvent(tenantID, msg, data)

	// ===== 排队相关事件 =====
	case strings.HasPrefix(eventType, "queue."):
		c.handleQueueEvent(tenantID, msg, data)

	// ===== 系统通知 =====
	case eventType == eventbus.EventSystemNotice:
		c.handleSystemNotice(msg)

	default:
		// 其他事件忽略（或可扩展）
		logger.Debug("business consumer: skip unknown event type",
			zap.String("event_type", eventType),
		)
	}
}

// handleAgentEvent 处理座席相关事件
// 推送给：座席本人 + 管理员 + 监控大屏
func (c *BusinessConsumer) handleAgentEvent(tenantID int64, msg *model.Message, data interface{}) {
	// 1. 推送给座席本人
	agentID := extractIntField(data, "agent_id")
	if agentID > 0 && tenantID > 0 {
		_, _ = c.pushService.PushToAgent(tenantID, agentID, msg.Event, msg.Data)
	}

	// 2. 推送给租户管理员和监控大屏（通过 PushToTenant，指定用户类型）
	if tenantID > 0 {
		userTypes := []string{model.UserTypeAdmin, model.UserTypeMonitor}
		_, _ = c.pushService.PushToTenant(tenantID, msg.Event, msg.Data, userTypes)
	}
}

// handleCallEvent 处理通话相关事件
// 推送给：座席本人 + 管理员 + 监控大屏
func (c *BusinessConsumer) handleCallEvent(tenantID int64, msg *model.Message, data interface{}) {
	// 1. 推送给座席本人
	agentID := extractIntField(data, "agent_id")
	if agentID > 0 && tenantID > 0 {
		_, _ = c.pushService.PushToAgent(tenantID, agentID, msg.Event, msg.Data)
	}

	// 2. 推送给租户管理员和监控大屏
	if tenantID > 0 {
		userTypes := []string{model.UserTypeAdmin, model.UserTypeMonitor}
		_, _ = c.pushService.PushToTenant(tenantID, msg.Event, msg.Data, userTypes)
	}
}

// handleQueueEvent 处理排队相关事件
// 队列事件分为两类：
//   - 单通话排队事件（queue.enqueue / queue.dequeue / queue.abandoned / queue.timeout）：
//     data 为 CallQueueEventData，围绕单个通话的入队/出队/放弃/超时
//   - 队列整体统计事件（queue.stats_update）：
//     data 为 QueueStatusData，包含等待人数、座席数、SLA 等整体统计
//
// 推送给：队列监控主题（queue:{queue_id}） + 租户管理员
func (c *BusinessConsumer) handleQueueEvent(tenantID int64, msg *model.Message, data interface{}) {
	queueID := extractIntField(data, "queue_id")

	// 1. 推送给队列监控主题（queue:{queue_id}）
	//    数据完整透传（CallQueueEventData 或 QueueStatusData）
	if queueID > 0 {
		queueTopic := c.subService.QueueTopic(queueID)
		_, _ = c.pushService.PushToTopic(queueTopic, msg.Event, msg.Data)
	}

	// 2. 推送给租户管理员
	if tenantID > 0 {
		userTypes := []string{model.UserTypeAdmin}
		_, _ = c.pushService.PushToTenant(tenantID, msg.Event, msg.Data, userTypes)
	}
}

// handleSystemNotice 处理系统通知
// 广播给所有人
func (c *BusinessConsumer) handleSystemNotice(msg *model.Message) {
	_, _ = c.pushService.PushToAll(msg.Event, msg.Data)
}

// ===== 辅助方法 =====

// extractIntField 从 data 中提取 int64 类型的字段
// 支持 float64（JSON 反序列化后的数字）、int64、int、string
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

// ===== 从 pkgredis.Client 获取底层 *redis.Client 的辅助函数 =====
// 由于 pkgredis.Client.Raw() 返回 UniversalClient，
// 在单机模式下它实际是 *redis.Client，需要做类型断言。

// GetRedisClientFromPkg 从 pkgredis.Client 获取底层 *redis.Client
// 如果是集群模式，返回 nil（集群模式下 Pub/Sub 行为不同，由调用方处理）
func GetRedisClientFromPkg(pkgClient *pkgredis.Client) *redis.Client {
	if pkgClient == nil {
		return nil
	}
	raw := pkgClient.Raw()
	if client, ok := raw.(*redis.Client); ok {
		return client
	}
	// ClusterClient / SentinelClient 也支持 Pub/Sub，
	// 但 eventbus.RedisSubscriber 需要 *redis.Client。
	// 生产环境推荐使用 NATS 作为事件总线，Redis 仅作降级。
	logger.Warn("pkg redis client is not *redis.Client (maybe cluster mode), " +
		"business consumer redis backend may not work properly")
	return nil
}
