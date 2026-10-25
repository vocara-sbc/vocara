package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"vocara/pkg/utils"
)

// 分布式锁 Lua 脚本
// 解锁脚本：只有持有锁的人才能解锁
var unlockScript = `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
`

// 续期脚本：只有持有锁的人才能续期
var renewScript = `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("pexpire", KEYS[1], ARGV[2])
	else
		return 0
	end
`

// DistributedLock 分布式锁
type DistributedLock struct {
	client    redis.UniversalClient
	key       string
	value     string
	expiration time.Duration
	renewCh   chan struct{} // 停止续期信号
}

// NewDistributedLock 创建分布式锁
func (c *Client) NewDistributedLock(key string, expiration time.Duration) *DistributedLock {
	return &DistributedLock{
		client:     c.universalClient,
		key:        key,
		value:      utils.UUID(), // 生成唯一值，防止误解锁
		expiration: expiration,
	}
}

// NewLock 创建分布式锁（使用全局客户端）
func NewLock(key string, expiration time.Duration) *DistributedLock {
	return globalClient.NewDistributedLock(key, expiration)
}

// Lock 尝试获取锁
func (l *DistributedLock) Lock(ctx context.Context) (bool, error) {
	ok, err := l.client.SetNX(ctx, l.key, l.value, l.expiration).Result()
	if err != nil {
		return false, fmt.Errorf("setnx failed: %w", err)
	}
	return ok, nil
}

// TryLock 尝试获取锁（带重试）
// retryCount: 重试次数
// retryInterval: 重试间隔
func (l *DistributedLock) TryLock(ctx context.Context, retryCount int, retryInterval time.Duration) (bool, error) {
	for i := 0; i < retryCount; i++ {
		ok, err := l.Lock(ctx)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		// 最后一次不等待
		if i < retryCount-1 {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(retryInterval):
			}
		}
	}
	return false, nil
}

// Unlock 释放锁
func (l *DistributedLock) Unlock(ctx context.Context) (bool, error) {
	// 停止自动续期
	if l.renewCh != nil {
		close(l.renewCh)
		l.renewCh = nil
	}

	result, err := l.client.Eval(ctx, unlockScript, []string{l.key}, l.value).Result()
	if err != nil {
		return false, fmt.Errorf("unlock script failed: %w", err)
	}

	return result.(int64) == 1, nil
}

// Renew 手动续期
func (l *DistributedLock) Renew(ctx context.Context) (bool, error) {
	result, err := l.client.Eval(ctx, renewScript, []string{l.key},
		l.value, fmt.Sprintf("%d", l.expiration.Milliseconds())).Result()
	if err != nil {
		return false, fmt.Errorf("renew script failed: %w", err)
	}

	return result.(int64) == 1, nil
}

// AutoRenew 启动自动续期
// 续期间隔为过期时间的 1/3
func (l *DistributedLock) AutoRenew(ctx context.Context) {
	if l.renewCh != nil {
		return // 已经在续期
	}

	l.renewCh = make(chan struct{})
	renewInterval := l.expiration / 3

	go func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 续期，忽略错误（失败会自动过期）
				_, _ = l.Renew(ctx)
			case <-l.renewCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Key 获取锁的 key
func (l *DistributedLock) Key() string {
	return l.key
}

// Value 获取锁的唯一值
func (l *DistributedLock) Value() string {
	return l.value
}
