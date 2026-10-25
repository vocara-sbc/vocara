package eventbus

import (
	"context"
	"fmt"
)

// DualPublisher 双通道发布者（NATS 优先，Redis 降级
// 如果 NATS 连接正常，发布到 NATS；如果 NATS 失败，降级到 Redis Pub/Sub
type DualPublisher struct {
	natsPub  *NATSPublisher
	redisPub *RedisPublisher
	natsOK   bool // NATS 是否可用
}

// DualPublisherConfig 双通道发布者配置
type DualPublisherConfig struct {
	// NatsPublisher  *NATSPublisher  // 可选，nil 则只用 Redis
	// RedisPublisher *RedisPublisher // 可选，nil 则只用 NATS
}

// NewDualPublisher 创建双通道发布者
// 至少需要一个后端，推荐 NATS + Redis 组合
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

// Publish 发布事件
// 优先发布到 NATS，如果 NATS 发布失败且有 Redis 时降级到 Redis
func (p *DualPublisher) Publish(ctx context.Context, event *Event) error {
	// 优先 NATS
	if p.natsOK && p.natsPub != nil {
		err := p.natsPub.Publish(ctx, event)
		if err == nil {
			return nil
		}
		// NATS 失败，标记为不可用，降级到 Redis
		p.natsOK = false
	}

	// 降级到 Redis
	if p.redisPub != nil {
		return p.redisPub.Publish(ctx, event)
	}

	// 两个都没有可用的
	return fmt.Errorf("no available publisher (nats failed and redis not available")
}

// Close 关闭所有后端
func (p *DualPublisher) Close() error {
	if p.natsPub != nil {
		_ = p.natsPub.Close()
	}
	if p.redisPub != nil {
		_ = p.redisPub.Close()
	}
	return nil
}
