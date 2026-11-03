package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"vocara/pkg/utils"
)

// Lua scripts for distributed locks.
// The unlock script allows only the lock holder to unlock it.
var unlockScript = `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
`

// The renewal script allows only the lock holder to renew it.
var renewScript = `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("pexpire", KEYS[1], ARGV[2])
	else
		return 0
	end
`

// DistributedLock is a distributed lock.
type DistributedLock struct {
	client    redis.UniversalClient
	key       string
	value     string
	expiration time.Duration
	renewCh   chan struct{} // Signal to stop renewal
}

// NewDistributedLock creates a distributed lock.
func (c *Client) NewDistributedLock(key string, expiration time.Duration) *DistributedLock {
	return &DistributedLock{
		client:     c.universalClient,
		key:        key,
		value:      utils.UUID(), // Generate a unique value to prevent accidental unlocking
		expiration: expiration,
	}
}

// NewLock creates a distributed lock using the global client.
func NewLock(key string, expiration time.Duration) *DistributedLock {
	return globalClient.NewDistributedLock(key, expiration)
}

// Lock attempts to acquire the lock.
func (l *DistributedLock) Lock(ctx context.Context) (bool, error) {
	ok, err := l.client.SetNX(ctx, l.key, l.value, l.expiration).Result()
	if err != nil {
		return false, fmt.Errorf("setnx failed: %w", err)
	}
	return ok, nil
}

// TryLock attempts to acquire the lock with retries.
// retryCount: number of retries
// retryInterval: interval between retries
func (l *DistributedLock) TryLock(ctx context.Context, retryCount int, retryInterval time.Duration) (bool, error) {
	for i := 0; i < retryCount; i++ {
		ok, err := l.Lock(ctx)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		// Do not wait after the final attempt
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

// Unlock releases the lock.
func (l *DistributedLock) Unlock(ctx context.Context) (bool, error) {
	// Stop automatic renewal
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

// Renew manually renews the lock.
func (l *DistributedLock) Renew(ctx context.Context) (bool, error) {
	result, err := l.client.Eval(ctx, renewScript, []string{l.key},
		l.value, fmt.Sprintf("%d", l.expiration.Milliseconds())).Result()
	if err != nil {
		return false, fmt.Errorf("renew script failed: %w", err)
	}

	return result.(int64) == 1, nil
}

// AutoRenew starts automatic renewal.
// The renewal interval is one-third of the expiration time.
func (l *DistributedLock) AutoRenew(ctx context.Context) {
	if l.renewCh != nil {
		return // Renewal is already running
	}

	l.renewCh = make(chan struct{})
	renewInterval := l.expiration / 3

	go func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Renew and ignore errors; the lock expires automatically on failure
				_, _ = l.Renew(ctx)
			case <-l.renewCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Key returns the lock key.
func (l *DistributedLock) Key() string {
	return l.key
}

// Value returns the lock's unique value.
func (l *DistributedLock) Value() string {
	return l.value
}
