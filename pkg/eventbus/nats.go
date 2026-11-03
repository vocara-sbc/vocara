package eventbus

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

// NATSPublisher is an event publisher based on NATS JetStream.
type NATSPublisher struct {
	conn *nats.Conn
}

// NewNATSPublisher creates a NATS event publisher.
func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

// Publish publishes an event to NATS.
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

// Close closes the connection.
func (p *NATSPublisher) Close() error {
	if p.conn != nil {
		p.conn.Close()
	}
	return nil
}

// NATSSubscriber is a NATS-based event subscriber.
type NATSSubscriber struct {
	conn *nats.Conn
	subs map[string]*nats.Subscription
}

// NATSSubscriberConfig contains NATS subscriber configuration.
type NATSSubscriberConfig struct {
	UseJetStream bool // Whether to use JetStream for persistence
	QueueGroup   string // Queue group name; empty selects broadcast mode
}

// NewNATSSubscriber creates a NATS event subscriber.
func NewNATSSubscriber(conn *nats.Conn) *NATSSubscriber {
	return &NATSSubscriber{
		conn: conn,
		subs: make(map[string]*nats.Subscription),
	}
}

// Subscribe subscribes to the specified event type.
// It returns a subscriptionID.
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

// SubscribeAll subscribes to all events using a wildcard.
func (s *NATSSubscriber) SubscribeAll(ctx context.Context, handler Handler) (string, error) {
	// NATS uses > as a multi-level wildcard
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

// SubscribeQueue performs a queue subscription so multiple consumers share messages.
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

// Unsubscribe cancels a subscription.
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

// Close closes all subscriptions and the connection.
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
