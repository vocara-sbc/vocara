// Package redis provides a Redis wrapper based on go-redis v9.
// It supports standalone, cluster, and sentinel modes, wraps common operations, and provides distributed locks.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"vocara/pkg/config"
)

// Client wraps a Redis client.
type Client struct {
	universalClient redis.UniversalClient
	config          config.RedisConfig
}

var (
	globalClient *Client
)

// New creates a Redis client.
// It supports three modes: single / cluster / sentinel.
func New(cfg config.RedisConfig) (*Client, error) {
	opts := &redis.UniversalOptions{
		Addrs:        cfg.AddrList(),
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	}

	// Timeout configuration
	if cfg.DialTimeout > 0 {
		opts.DialTimeout = time.Duration(cfg.DialTimeout) * time.Millisecond
	}
	if cfg.ReadTimeout > 0 {
		opts.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Millisecond
	}
	if cfg.WriteTimeout > 0 {
		opts.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Millisecond
	}
	if cfg.PoolTimeout > 0 {
		opts.PoolTimeout = time.Duration(cfg.PoolTimeout) * time.Millisecond
	}
	if cfg.IdleTimeout > 0 {
		opts.ConnMaxIdleTime = time.Duration(cfg.IdleTimeout) * time.Second
	}

	// Configure settings by mode
	switch cfg.Mode {
	case "cluster":
		// Cluster mode is handled automatically by UniversalClient
	case "sentinel":
		opts.MasterName = cfg.MasterName
	default:
		// Standalone mode
	}

	client := redis.NewUniversalClient(opts)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Client{
		universalClient: client,
		config:          cfg,
	}, nil
}

// Init initializes the global Redis client.
func Init(cfg config.RedisConfig) error {
	client, err := New(cfg)
	if err != nil {
		return err
	}
	globalClient = client
	return nil
}

// Get returns the global Redis client.
func Get() *Client {
	return globalClient
}

// SetGlobal sets the global client for testing.
func SetGlobal(c *Client) {
	globalClient = c
}

// Raw returns the underlying go-redis client.
func (c *Client) Raw() redis.UniversalClient {
	return c.universalClient
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.universalClient.Close()
}

// Ping tests the connection.
func (c *Client) Ping(ctx context.Context) (string, error) {
	return c.universalClient.Ping(ctx).Result()
}

// ===== String operations =====

// Get returns the value of a key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.universalClient.Get(ctx, key).Result()
}

// Set sets a key's value and expiration time.
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.universalClient.Set(ctx, key, value, expiration).Err()
}

// SetNX sets a key only if it does not exist.
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.universalClient.SetNX(ctx, key, value, expiration).Result()
}

// SetEX sets a key's value and expiration time with SETEX semantics.
// The go-redis v9 UniversalClient interface does not expose SetEX, so Set with an expiration time provides equivalent behavior.
func (c *Client) SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.universalClient.Set(ctx, key, value, expiration).Err()
}

// Del deletes keys.
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	return c.universalClient.Del(ctx, keys...).Result()
}

// Exists checks whether keys exist.
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.universalClient.Exists(ctx, keys...).Result()
}

// Expire sets a key's expiration time.
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return c.universalClient.Expire(ctx, key, expiration).Result()
}

// TTL returns a key's remaining expiration time.
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.universalClient.TTL(ctx, key).Result()
}

// Incr increments a key by 1.
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.universalClient.Incr(ctx, key).Result()
}

// IncrBy increments a key by the specified value.
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.universalClient.IncrBy(ctx, key, value).Result()
}

// Decr decrements a key by 1.
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.universalClient.Decr(ctx, key).Result()
}

// DecrBy decrements a key by the specified value.
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.universalClient.DecrBy(ctx, key, value).Result()
}

// Append appends a value.
func (c *Client) Append(ctx context.Context, key, value string) (int64, error) {
	return c.universalClient.Append(ctx, key, value).Result()
}

// StrLen returns the string length.
func (c *Client) StrLen(ctx context.Context, key string) (int64, error) {
	return c.universalClient.StrLen(ctx, key).Result()
}

// MSet sets multiple values.
func (c *Client) MSet(ctx context.Context, values ...interface{}) error {
	return c.universalClient.MSet(ctx, values...).Err()
}

// MGet gets multiple values.
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.universalClient.MGet(ctx, keys...).Result()
}

// ===== Hash operations =====

// HSet sets hash fields.
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.universalClient.HSet(ctx, key, values...).Err()
}

// HGet returns a hash field value.
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.universalClient.HGet(ctx, key, field).Result()
}

// HGetAll returns all hash fields.
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.universalClient.HGetAll(ctx, key).Result()
}

// HDel deletes hash fields.
func (c *Client) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	return c.universalClient.HDel(ctx, key, fields...).Result()
}

// HExists checks whether a hash field exists.
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	return c.universalClient.HExists(ctx, key, field).Result()
}

// HLen returns the number of hash fields.
func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	return c.universalClient.HLen(ctx, key).Result()
}

// HIncrBy increments a hash field.
func (c *Client) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return c.universalClient.HIncrBy(ctx, key, field, incr).Result()
}

// HKeys returns all hash field names.
func (c *Client) HKeys(ctx context.Context, key string) ([]string, error) {
	return c.universalClient.HKeys(ctx, key).Result()
}

// HVals returns all hash field values.
func (c *Client) HVals(ctx context.Context, key string) ([]string, error) {
	return c.universalClient.HVals(ctx, key).Result()
}

// HMSet sets multiple hash fields.
func (c *Client) HMSet(ctx context.Context, key string, values ...interface{}) error {
	return c.universalClient.HMSet(ctx, key, values...).Err()
}

// HMGet gets multiple hash fields.
func (c *Client) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	return c.universalClient.HMGet(ctx, key, fields...).Result()
}

// ===== List operations =====

// LPush pushes values from the left.
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return c.universalClient.LPush(ctx, key, values...).Result()
}

// RPush pushes values from the right.
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return c.universalClient.RPush(ctx, key, values...).Result()
}

// LPop pops a value from the left.
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.universalClient.LPop(ctx, key).Result()
}

// RPop pops a value from the right.
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.universalClient.RPop(ctx, key).Result()
}

// LLen returns the list length.
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.universalClient.LLen(ctx, key).Result()
}

// LRange returns a range of list elements.
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.universalClient.LRange(ctx, key, start, stop).Result()
}

// LIndex returns the element at the specified index.
func (c *Client) LIndex(ctx context.Context, key string, index int64) (string, error) {
	return c.universalClient.LIndex(ctx, key, index).Result()
}

// LRem removes elements.
func (c *Client) LRem(ctx context.Context, key string, count int64, value interface{}) (int64, error) {
	return c.universalClient.LRem(ctx, key, count, value).Result()
}

// LTrim trims a list.
func (c *Client) LTrim(ctx context.Context, key string, start, stop int64) error {
	return c.universalClient.LTrim(ctx, key, start, stop).Err()
}

// ===== Set operations =====

// SAdd adds members.
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return c.universalClient.SAdd(ctx, key, members...).Result()
}

// SRem removes members.
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return c.universalClient.SRem(ctx, key, members...).Result()
}

// SIsMember reports whether a value is a member.
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.universalClient.SIsMember(ctx, key, member).Result()
}

// SCard returns the number of set members.
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	return c.universalClient.SCard(ctx, key).Result()
}

// SMembers returns all members.
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.universalClient.SMembers(ctx, key).Result()
}

// SInter returns the intersection.
func (c *Client) SInter(ctx context.Context, keys ...string) ([]string, error) {
	return c.universalClient.SInter(ctx, keys...).Result()
}

// SUnion returns the union.
func (c *Client) SUnion(ctx context.Context, keys ...string) ([]string, error) {
	return c.universalClient.SUnion(ctx, keys...).Result()
}

// SDiff returns the difference.
func (c *Client) SDiff(ctx context.Context, keys ...string) ([]string, error) {
	return c.universalClient.SDiff(ctx, keys...).Result()
}

// SRandMember returns a random member.
func (c *Client) SRandMember(ctx context.Context, key string) (string, error) {
	return c.universalClient.SRandMember(ctx, key).Result()
}

// SPop pops a random member.
func (c *Client) SPop(ctx context.Context, key string) (string, error) {
	return c.universalClient.SPop(ctx, key).Result()
}

// ===== ZSet operations =====

// ZAdd adds members.
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	return c.universalClient.ZAdd(ctx, key, members...).Result()
}

// ZRem removes members.
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return c.universalClient.ZRem(ctx, key, members...).Result()
}

// ZScore returns a member's score.
func (c *Client) ZScore(ctx context.Context, key, member string) (float64, error) {
	return c.universalClient.ZScore(ctx, key, member).Result()
}

// ZCard returns the number of sorted set members.
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	return c.universalClient.ZCard(ctx, key).Result()
}

// ZRank returns a member's rank in ascending order.
func (c *Client) ZRank(ctx context.Context, key, member string) (int64, error) {
	return c.universalClient.ZRank(ctx, key, member).Result()
}

// ZRevRank returns a member's rank in descending order.
func (c *Client) ZRevRank(ctx context.Context, key, member string) (int64, error) {
	return c.universalClient.ZRevRank(ctx, key, member).Result()
}

// ZRange returns members in the specified rank range in ascending order.
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.universalClient.ZRange(ctx, key, start, stop).Result()
}

// ZRevRange returns members in the specified rank range in descending order.
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.universalClient.ZRevRange(ctx, key, start, stop).Result()
}

// ZRangeByScore returns members in the specified score range.
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return c.universalClient.ZRangeByScore(ctx, key, opt).Result()
}

// ZCount returns the number of members in the specified score range.
func (c *Client) ZCount(ctx context.Context, key, min, max string) (int64, error) {
	return c.universalClient.ZCount(ctx, key, min, max).Result()
}

// ZIncrBy increments a member's score.
func (c *Client) ZIncrBy(ctx context.Context, key, member string, incr float64) (float64, error) {
	return c.universalClient.ZIncrBy(ctx, key, incr, member).Result()
}

// ===== Other operations =====

// Keys returns all matching keys; use with caution in production.
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.universalClient.Keys(ctx, pattern).Result()
}

// Scan iterates over keys and is recommended for large keyspaces.
func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return c.universalClient.Scan(ctx, cursor, match, count).Result()
}

// Type returns a key's type.
func (c *Client) Type(ctx context.Context, key string) (string, error) {
	return c.universalClient.Type(ctx, key).Result()
}

// Rename renames a key.
func (c *Client) Rename(ctx context.Context, key, newkey string) error {
	return c.universalClient.Rename(ctx, key, newkey).Err()
}

// Persist removes a key's expiration time.
func (c *Client) Persist(ctx context.Context, key string) (bool, error) {
	return c.universalClient.Persist(ctx, key).Result()
}

// ===== Lua scripts =====

// Eval executes a Lua script.
func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return c.universalClient.Eval(ctx, script, keys, args...).Result()
}

// EvalSha executes a Lua script by SHA.
func (c *Client) EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	return c.universalClient.EvalSha(ctx, sha, keys, args...).Result()
}

// ScriptLoad loads a Lua script.
func (c *Client) ScriptLoad(ctx context.Context, script string) (string, error) {
	return c.universalClient.ScriptLoad(ctx, script).Result()
}

// ===== Pub/Sub operations =====

// Publish publishes a message to the specified channel for immediate cross-service notifications such as authorization changes.
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.universalClient.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to the specified channel and continuously calls handler; it returns when ctx is canceled or the channel closes permanently.
// go-redis automatically reconnects and resubscribes after a disconnection, so no additional runtime handling is required.
func (c *Client) Subscribe(ctx context.Context, channel string, handler func(message []byte)) error {
	pubsub := c.universalClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Wait for subscription confirmation to ensure the channel is established; return errors for the caller to decide whether to retry
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("subscription channel closed")
			}
			handler([]byte(msg.Payload))
		}
	}
}

// ===== Pipelines =====

// Pipeline creates a pipeline.
func (c *Client) Pipeline() redis.Pipeliner {
	return c.universalClient.Pipeline()
}
