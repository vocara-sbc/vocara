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

// RedisConsumer Redis Pub/Sub 消费者
// 使用 Redis Pub/Sub 消费推送消息，是 NATS 消费者的备选方案
// 适用于部署简单、消息量不大的场景
//
// 订阅的频道：
// - vocara:push:agent:*       推送给指定座席
// - vocara:push:tenant:*      推送给指定租户
// - vocara:push:topic:*       推送到指定主题
// - vocara:push:broadcast     全局广播
type RedisConsumer struct {
	connManager *service.ConnectionManager
	subService  *service.SubscriptionService
	pushService *service.PushService

	redisClient *pkgredis.Client

	// 订阅客户端
	pubsub *redis.PubSub

	// 停止相关
	stopCh chan struct{}
	once   sync.Once
}

// NewRedisConsumer 创建 Redis Pub/Sub 消费者
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

// Start 启动 Redis 消费者
func (r *RedisConsumer) Start(ctx context.Context) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client is nil")
	}

	// 使用 PSUBSCRIBE 订阅模式匹配的频道
	patterns := []string{
		"vocara:push:agent:*",
		"vocara:push:tenant:*",
		"vocara:push:topic:*",
		"vocara:push:broadcast",
	}

	r.pubsub = r.redisClient.Raw().PSubscribe(ctx, patterns...)

	// 确认订阅
	_, err := r.pubsub.Receive(ctx)
	if err != nil {
		_ = r.pubsub.Close()
		return fmt.Errorf("redis psubscribe failed: %w", err)
	}

	logger.Info("redis pubsub consumer started",
		zap.Strings("patterns", patterns),
	)

	// 启动消息处理循环
	go r.messageLoop(ctx)

	return nil
}

// Stop 停止 Redis 消费者
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

// messageLoop 消息处理循环
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

// handleMessage 处理收到的消息
func (r *RedisConsumer) handleMessage(channel string, data []byte) {
	logger.Debug("received redis pubsub message",
		zap.String("channel", channel),
	)

	// 解析消息
	var pushMsg model.Message
	if err := json.Unmarshal(data, &pushMsg); err != nil {
		logger.Error("unmarshal redis push message failed",
			zap.String("channel", channel),
			zap.Error(err),
		)
		return
	}

	// 根据频道类型处理
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

// handleAgentMessage 处理座席推送消息
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

	// 从消息 data 中提取 tenantId
	tenantID := extractTenantIDFromData(msg.Data)

	// 通过推送服务推送
	_, err = r.pushService.PushToAgent(tenantID, agentID, msg.Event, msg.Data)
	if err != nil {
		logger.Error("push to agent failed",
			zap.Int64("agent_id", agentID),
			zap.Error(err),
		)
	}
}

// handleTenantMessage 处理租户推送消息
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

// handleTopicMessage 处理主题推送消息
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

// handleBroadcastMessage 处理广播消息
func (r *RedisConsumer) handleBroadcastMessage(msg *model.Message) {
	logger.Debug("redis push broadcast",
		zap.String("event", msg.Event),
	)

	_, err := r.pushService.PushToAll(msg.Event, msg.Data)
	if err != nil {
		logger.Error("push to all failed", zap.Error(err))
	}
}
