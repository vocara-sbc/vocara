// Package redis 提供基于 go-redis v9 的 Redis 封装
// 支持单机/集群/哨兵模式，封装常用操作，提供分布式锁
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"vocara/pkg/config"
)

// Client Redis 客户端封装
type Client struct {
	universalClient redis.UniversalClient
	config          config.RedisConfig
}

var (
	globalClient *Client
)

// New 创建新的 Redis 客户端
// 支持三种模式: single / cluster / sentinel
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

	// 超时配置
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

	// 根据模式设置
	switch cfg.Mode {
	case "cluster":
		// 集群模式，UniversalClient 会自动处理
	case "sentinel":
		opts.MasterName = cfg.MasterName
	default:
		// 单机模式
	}

	client := redis.NewUniversalClient(opts)

	// 测试连接
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

// Init 初始化全局 Redis 客户端
func Init(cfg config.RedisConfig) error {
	client, err := New(cfg)
	if err != nil {
		return err
	}
	globalClient = client
	return nil
}

// Get 获取全局 Redis 客户端
func Get() *Client {
	return globalClient
}

// SetGlobal 设置全局客户端（用于测试）
func SetGlobal(c *Client) {
	globalClient = c
}

// Raw 获取底层的 go-redis 客户端
func (c *Client) Raw() redis.UniversalClient {
	return c.universalClient
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.universalClient.Close()
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) (string, error) {
	return c.universalClient.Ping(ctx).Result()
}

// ===== String 操作 =====

// Get 获取 key 的值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.universalClient.Get(ctx, key).Result()
}

// Set 设置 key 的值和过期时间
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.universalClient.Set(ctx, key, value, expiration).Err()
}

// SetNX 设置 key（仅当 key 不存在时）
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.universalClient.SetNX(ctx, key, value, expiration).Result()
}

// SetEX 设置 key 的值和过期时间（SETEX 语义）。
// go-redis v9 的 UniversalClient 接口未暴露 SetEX，使用 Set 携带过期时间等价实现。
func (c *Client) SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.universalClient.Set(ctx, key, value, expiration).Err()
}

// Del 删除 key
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	return c.universalClient.Del(ctx, keys...).Result()
}

// Exists 检查 key 是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.universalClient.Exists(ctx, keys...).Result()
}

// Expire 设置 key 的过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return c.universalClient.Expire(ctx, key, expiration).Result()
}

// TTL 获取 key 的剩余过期时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.universalClient.TTL(ctx, key).Result()
}

// Incr key 自增 1
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.universalClient.Incr(ctx, key).Result()
}

// IncrBy key 自增指定值
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.universalClient.IncrBy(ctx, key, value).Result()
}

// Decr key 自减 1
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.universalClient.Decr(ctx, key).Result()
}

// DecrBy key 自减指定值
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.universalClient.DecrBy(ctx, key, value).Result()
}

// Append 追加值
func (c *Client) Append(ctx context.Context, key, value string) (int64, error) {
	return c.universalClient.Append(ctx, key, value).Result()
}

// StrLen 获取字符串长度
func (c *Client) StrLen(ctx context.Context, key string) (int64, error) {
	return c.universalClient.StrLen(ctx, key).Result()
}

// MSet 批量设置
func (c *Client) MSet(ctx context.Context, values ...interface{}) error {
	return c.universalClient.MSet(ctx, values...).Err()
}

// MGet 批量获取
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.universalClient.MGet(ctx, keys...).Result()
}

// ===== Hash 操作 =====

// HSet 设置 hash 字段
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.universalClient.HSet(ctx, key, values...).Err()
}

// HGet 获取 hash 字段值
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.universalClient.HGet(ctx, key, field).Result()
}

// HGetAll 获取 hash 所有字段
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.universalClient.HGetAll(ctx, key).Result()
}

// HDel 删除 hash 字段
func (c *Client) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	return c.universalClient.HDel(ctx, key, fields...).Result()
}

// HExists 检查 hash 字段是否存在
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	return c.universalClient.HExists(ctx, key, field).Result()
}

// HLen 获取 hash 字段数量
func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	return c.universalClient.HLen(ctx, key).Result()
}

// HIncrBy hash 字段自增
func (c *Client) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return c.universalClient.HIncrBy(ctx, key, field, incr).Result()
}

// HKeys 获取 hash 所有字段名
func (c *Client) HKeys(ctx context.Context, key string) ([]string, error) {
	return c.universalClient.HKeys(ctx, key).Result()
}

// HVals 获取 hash 所有字段值
func (c *Client) HVals(ctx context.Context, key string) ([]string, error) {
	return c.universalClient.HVals(ctx, key).Result()
}

// HMSet 批量设置 hash 字段
func (c *Client) HMSet(ctx context.Context, key string, values ...interface{}) error {
	return c.universalClient.HMSet(ctx, key, values...).Err()
}

// HMGet 批量获取 hash 字段
func (c *Client) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	return c.universalClient.HMGet(ctx, key, fields...).Result()
}

// ===== List 操作 =====

// LPush 从左边推入
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return c.universalClient.LPush(ctx, key, values...).Result()
}

// RPush 从右边推入
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return c.universalClient.RPush(ctx, key, values...).Result()
}

// LPop 从左边弹出
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.universalClient.LPop(ctx, key).Result()
}

// RPop 从右边弹出
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.universalClient.RPop(ctx, key).Result()
}

// LLen 获取列表长度
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.universalClient.LLen(ctx, key).Result()
}

// LRange 获取列表范围
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.universalClient.LRange(ctx, key, start, stop).Result()
}

// LIndex 获取指定索引的元素
func (c *Client) LIndex(ctx context.Context, key string, index int64) (string, error) {
	return c.universalClient.LIndex(ctx, key, index).Result()
}

// LRem 移除元素
func (c *Client) LRem(ctx context.Context, key string, count int64, value interface{}) (int64, error) {
	return c.universalClient.LRem(ctx, key, count, value).Result()
}

// LTrim 修剪列表
func (c *Client) LTrim(ctx context.Context, key string, start, stop int64) error {
	return c.universalClient.LTrim(ctx, key, start, stop).Err()
}

// ===== Set 操作 =====

// SAdd 添加成员
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return c.universalClient.SAdd(ctx, key, members...).Result()
}

// SRem 移除成员
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return c.universalClient.SRem(ctx, key, members...).Result()
}

// SIsMember 判断是否是成员
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.universalClient.SIsMember(ctx, key, member).Result()
}

// SCard 获取集合成员数
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	return c.universalClient.SCard(ctx, key).Result()
}

// SMembers 获取所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.universalClient.SMembers(ctx, key).Result()
}

// SInter 交集
func (c *Client) SInter(ctx context.Context, keys ...string) ([]string, error) {
	return c.universalClient.SInter(ctx, keys...).Result()
}

// SUnion 并集
func (c *Client) SUnion(ctx context.Context, keys ...string) ([]string, error) {
	return c.universalClient.SUnion(ctx, keys...).Result()
}

// SDiff 差集
func (c *Client) SDiff(ctx context.Context, keys ...string) ([]string, error) {
	return c.universalClient.SDiff(ctx, keys...).Result()
}

// SRandMember 随机获取成员
func (c *Client) SRandMember(ctx context.Context, key string) (string, error) {
	return c.universalClient.SRandMember(ctx, key).Result()
}

// SPop 随机弹出成员
func (c *Client) SPop(ctx context.Context, key string) (string, error) {
	return c.universalClient.SPop(ctx, key).Result()
}

// ===== ZSet 操作 =====

// ZAdd 添加成员
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	return c.universalClient.ZAdd(ctx, key, members...).Result()
}

// ZRem 移除成员
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return c.universalClient.ZRem(ctx, key, members...).Result()
}

// ZScore 获取成员分数
func (c *Client) ZScore(ctx context.Context, key, member string) (float64, error) {
	return c.universalClient.ZScore(ctx, key, member).Result()
}

// ZCard 获取有序集合成员数
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	return c.universalClient.ZCard(ctx, key).Result()
}

// ZRank 获取成员排名（从小到大）
func (c *Client) ZRank(ctx context.Context, key, member string) (int64, error) {
	return c.universalClient.ZRank(ctx, key, member).Result()
}

// ZRevRank 获取成员排名（从大到小）
func (c *Client) ZRevRank(ctx context.Context, key, member string) (int64, error) {
	return c.universalClient.ZRevRank(ctx, key, member).Result()
}

// ZRange 获取指定排名范围的成员（从小到大）
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.universalClient.ZRange(ctx, key, start, stop).Result()
}

// ZRevRange 获取指定排名范围的成员（从大到小）
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.universalClient.ZRevRange(ctx, key, start, stop).Result()
}

// ZRangeByScore 获取指定分数范围的成员
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return c.universalClient.ZRangeByScore(ctx, key, opt).Result()
}

// ZCount 获取指定分数范围的成员数
func (c *Client) ZCount(ctx context.Context, key, min, max string) (int64, error) {
	return c.universalClient.ZCount(ctx, key, min, max).Result()
}

// ZIncrBy 成员分数自增
func (c *Client) ZIncrBy(ctx context.Context, key, member string, incr float64) (float64, error) {
	return c.universalClient.ZIncrBy(ctx, key, incr, member).Result()
}

// ===== 其他操作 =====

// Keys 获取所有匹配的 key（生产环境慎用）
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.universalClient.Keys(ctx, pattern).Result()
}

// Scan 迭代 key（推荐用于大 key 空间）
func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return c.universalClient.Scan(ctx, cursor, match, count).Result()
}

// Type 获取 key 类型
func (c *Client) Type(ctx context.Context, key string) (string, error) {
	return c.universalClient.Type(ctx, key).Result()
}

// Rename 重命名 key
func (c *Client) Rename(ctx context.Context, key, newkey string) error {
	return c.universalClient.Rename(ctx, key, newkey).Err()
}

// Persist 移除过期时间
func (c *Client) Persist(ctx context.Context, key string) (bool, error) {
	return c.universalClient.Persist(ctx, key).Result()
}

// ===== Lua 脚本 =====

// Eval 执行 Lua 脚本
func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return c.universalClient.Eval(ctx, script, keys, args...).Result()
}

// EvalSha 执行 Lua 脚本（通过 SHA）
func (c *Client) EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	return c.universalClient.EvalSha(ctx, sha, keys, args...).Result()
}

// ScriptLoad 加载 Lua 脚本
func (c *Client) ScriptLoad(ctx context.Context, script string) (string, error) {
	return c.universalClient.ScriptLoad(ctx, script).Result()
}

// ===== Pub/Sub 操作 =====

// Publish 向指定频道发布消息，用于跨服务即时通知（如授权变更）。
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.universalClient.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅指定频道并持续调用 handler；ctx 取消或频道永久关闭时返回。
// 连接断开由 go-redis 自动重连并重新订阅，运行期无需额外处理。
func (c *Client) Subscribe(ctx context.Context, channel string, handler func(message []byte)) error {
	pubsub := c.universalClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	// 等待订阅确认，确保频道建立；失败则返回错误，由上层决定是否重试
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

// ===== 管道 =====

// Pipeline 创建管道
func (c *Client) Pipeline() redis.Pipeliner {
	return c.universalClient.Pipeline()
}
