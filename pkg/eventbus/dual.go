package eventbus

import (
	"context"
	"fmt"
)

// DualPublisher publishes through two channels, preferring NATS with Redis as fallback.
// It publishes to NATS when the NATS connection is healthy and falls back to Redis Pub/Sub when NATS fails.
type DualPublisher struct {
	natsPub  *NATSPublisher
	redisPub *RedisPublisher
	natsOK   bool // Whether NATS is available
}

// DualPublisherConfig contains dual-channel publisher configuration.
type DualPublisherConfig struct {
	// NatsPublisher  *NATSPublisher  // Optional; nil uses only Redis
	// RedisPublisher *RedisPublisher // Optional; nil uses only NATS
}

// NewDualPublisher creates a dual-channel publisher.
// At least one backend is required; NATS + Redis is recommended.
func NewDualPublisher(natsPub *NATSPublisher, redisPub *RedisPublisher) (*DualPublisher, error) {
	if natsPub == nil && redisPub == nil {
		return nil, fmt.Errorf("at least one publisher is required")
	}

	return &DualPublisher{
		natsPub:  natsPub,
		redisPub: redisPub,
		natsOK:   natsPub != nil,
	}, nil
}

// Publish publishes an event.
// It prefers NATS and falls back to Redis when NATS publication fails and Redis is available.
func (p *DualPublisher) Publish(ctx context.Context, event *Event) error {
	// Prefer NATS
	if p.natsOK && p.natsPub != nil {
		err := p.natsPub.Publish(ctx, event)
		if err == nil {
			return nil
		}
		// Mark NATS as unavailable and fall back to Redis
		p.natsOK = false
	}

	// Fall back to Redis
	if p.redisPub != nil {
		return p.redisPub.Publish(ctx, event)
	}

	// Neither backend is available
	return fmt.Errorf("no available publisher (nats failed and redis not available")
}

// Close closes all backends.
func (p *DualPublisher) Close() error {
	if p.natsPub != nil {
		_ = p.natsPub.Close()
	}
	if p.redisPub != nil {
		_ = p.redisPub.Close()
	}
	return nil
}
