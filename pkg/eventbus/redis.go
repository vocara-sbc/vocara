package eventbus

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher is an event publisher based on Redis Pub/Sub.
type RedisPublisher struct {
	client redis.UniversalClient
}

// NewRedisPublisher creates a Redis event publisher that accepts a UniversalClient for standalone or cluster mode.
func NewRedisPublisher(client redis.UniversalClient) *RedisPublisher {
	return &RedisPublisher{client: client}
}

// Publish publishes an event to Redis Pub/Sub.
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

// Close does nothing because a Redis publisher requires no additional cleanup.
func (p *RedisPublisher) Close() error {
	return nil
}

// RedisSubscriber is an event subscriber based on Redis Pub/Sub.
type RedisSubscriber struct {
	client redis.UniversalClient
	pubsub *redis.PubSub
}

// NewRedisSubscriber creates a Redis event subscriber that accepts a UniversalClient for standalone or cluster mode.
func NewRedisSubscriber(client redis.UniversalClient) *RedisSubscriber {
	return &RedisSubscriber{client: client}
}

// Subscribe subscribes to the specified event type.
// It returns a subscriptionID, which is the topic in Redis mode.
func (s *RedisSubscriber) Subscribe(ctx context.Context, eventType string, handler Handler) (string, error) {
	topic := GetTopic(eventType)
	pubsub := s.client.Subscribe(ctx, topic)

	// Start the receiving goroutine
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

// SubscribeAll subscribes to all events using a wildcard.
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

// Unsubscribe cancels a subscription; Redis mode closes the entire pubsub.
func (s *RedisSubscriber) Unsubscribe(ctx context.Context, subscriptionID string) error {
	if s.pubsub != nil {
		return s.pubsub.Unsubscribe(ctx, subscriptionID)
	}
	return nil
}

// Close closes the subscription.
func (s *RedisSubscriber) Close() error {
	if s.pubsub != nil {
		return s.pubsub.Close()
	}
	return nil
}
