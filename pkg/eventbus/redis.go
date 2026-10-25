package eventbus

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher 基于 Redis Pub/Sub 的事件发布者
type RedisPublisher struct {
	client redis.UniversalClient
}

// NewRedisPublisher 创建 Redis 事件发布者（接受 UniversalClient，单机/集群均可）
func NewRedisPublisher(client redis.UniversalClient) *RedisPublisher {
	return &RedisPublisher{client: client}
}

// Publish 发布事件到 Redis Pub/Sub
func (p *RedisPublisher) Publish(ctx context.Context, event *Event) error {
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}

	topic := event.Topic()
	if err := p.client.Publish(ctx, topic, data).Err(); err != nil {
		return fmt.Errorf("redis publish failed (topic=%s): %w", topic, err)
	}

	return nil
}

// Close 关闭（Redis 发布者不需要额外关闭）
func (p *RedisPublisher) Close() error {
	return nil
}

// RedisSubscriber 基于 Redis Pub/Sub 的事件订阅者
type RedisSubscriber struct {
	client redis.UniversalClient
	pubsub *redis.PubSub
}

// NewRedisSubscriber 创建 Redis 事件订阅者（接受 UniversalClient，单机/集群均可）
func NewRedisSubscriber(client redis.UniversalClient) *RedisSubscriber {
	return &RedisSubscriber{client: client}
}

// Subscribe 订阅指定事件类型
// 返回 subscriptionID（在 Redis 模式下就是 topic）
func (s *RedisSubscriber) Subscribe(ctx context.Context, eventType string, handler Handler) (string, error) {
	topic := GetTopic(eventType)
	pubsub := s.client.Subscribe(ctx, topic)

	// 启动接收协程
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			event, err := ParseEvent([]byte(msg.Payload))
			if err != nil {
				continue
			}
			_ = handler(ctx, event)
		}
	}()

	s.pubsub = pubsub
	return topic, nil
}

// SubscribeAll 订阅所有事件（使用通配符）
func (s *RedisSubscriber) SubscribeAll(ctx context.Context, handler Handler) error {
	pattern := TopicPrefix + "*"
	pubsub := s.client.PSubscribe(ctx, pattern)

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			event, err := ParseEvent([]byte(msg.Payload))
			if err != nil {
				continue
			}
			_ = handler(ctx, event)
		}
	}()

	s.pubsub = pubsub
	return nil
}

// Unsubscribe 取消订阅（Redis 模式下关闭整个 pubsub）
func (s *RedisSubscriber) Unsubscribe(ctx context.Context, subscriptionID string) error {
	if s.pubsub != nil {
		return s.pubsub.Unsubscribe(ctx, subscriptionID)
	}
	return nil
}

// Close 关闭订阅
func (s *RedisSubscriber) Close() error {
	if s.pubsub != nil {
		return s.pubsub.Close()
	}
	return nil
}
