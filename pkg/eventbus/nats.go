package eventbus

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

// NATSPublisher 基于 NATS JetStream 的事件发布者
type NATSPublisher struct {
	conn *nats.Conn
}

// NewNATSPublisher 创建 NATS 事件发布者
func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

// Publish 发布事件到 NATS
func (p *NATSPublisher) Publish(ctx context.Context, event *Event) error {
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}

	topic := event.Topic()
	if err := p.conn.Publish(topic, data); err != nil {
		return fmt.Errorf("nats publish failed (topic=%s): %w", topic, err)
	}

	return nil
}

// Close 关闭连接
func (p *NATSPublisher) Close() error {
	if p.conn != nil {
		p.conn.Close()
	}
	return nil
}

// NATSSubscriber 基于 NATS 的事件订阅者
type NATSSubscriber struct {
	conn *nats.Conn
	subs map[string]*nats.Subscription
}

// NATSSubscriberConfig NATS 订阅者配置
type NATSSubscriberConfig struct {
	UseJetStream bool // 是否使用 JetStream（持久化）
	QueueGroup   string // 队列组名（空则广播模式）
}

// NewNATSSubscriber 创建 NATS 事件订阅者
func NewNATSSubscriber(conn *nats.Conn) *NATSSubscriber {
	return &NATSSubscriber{
		conn: conn,
		subs: make(map[string]*nats.Subscription),
	}
}

// Subscribe 订阅指定事件类型
// 返回 subscriptionID
func (s *NATSSubscriber) Subscribe(ctx context.Context, eventType string, handler Handler) (string, error) {
	topic := GetTopic(eventType)

	sub, err := s.conn.Subscribe(topic, func(msg *nats.Msg) {
		event, err := ParseEvent(msg.Data)
		if err != nil {
			return
		}
		_ = handler(ctx, event)
	})
	if err != nil {
		return "", fmt.Errorf("nats subscribe failed (topic=%s): %w", topic, err)
	}

	s.subs[topic] = sub
	return topic, nil
}

// SubscribeAll 订阅所有事件（使用通配符）
func (s *NATSSubscriber) SubscribeAll(ctx context.Context, handler Handler) (string, error) {
	// NATS 使用 > 作为多层通配符
	topic := TopicPrefix + ">"

	sub, err := s.conn.Subscribe(topic, func(msg *nats.Msg) {
		event, err := ParseEvent(msg.Data)
		if err != nil {
			return
		}
		_ = handler(ctx, event)
	})
	if err != nil {
		return "", fmt.Errorf("nats subscribe all failed: %w", err)
	}

	s.subs[topic] = sub
	return topic, nil
}

// SubscribeQueue 队列订阅（多个消费者分摊消息）
func (s *NATSSubscriber) SubscribeQueue(ctx context.Context, eventType string, queueGroup string, handler Handler) (string, error) {
	topic := GetTopic(eventType)

	sub, err := s.conn.QueueSubscribe(topic, queueGroup, func(msg *nats.Msg) {
		event, err := ParseEvent(msg.Data)
		if err != nil {
			return
		}
		_ = handler(ctx, event)
	})
	if err != nil {
		return "", fmt.Errorf("nats queue subscribe failed (topic=%s): %w", topic, err)
	}

	s.subs[topic+":"+queueGroup] = sub
	return topic + ":" + queueGroup, nil
}

// Unsubscribe 取消订阅
func (s *NATSSubscriber) Unsubscribe(ctx context.Context, subscriptionID string) error {
	sub, ok := s.subs[subscriptionID]
	if !ok {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("unsubscribe failed: %w", err)
	}

	delete(s.subs, subscriptionID)
	return nil
}

// Close 关闭所有订阅和连接
func (s *NATSSubscriber) Close() error {
	var errs []string
	for id, sub := range s.subs {
		if err := sub.Unsubscribe(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("unsubscribe errors: %s", strings.Join(errs, "; "))
	}

	return nil
}
