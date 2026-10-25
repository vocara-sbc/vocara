# redis - Redis 封装包

基于 go-redis v9 的 Redis 封装，支持单机/集群/哨兵模式，封装常用操作，提供分布式锁。

## 功能特性

- 支持单机 / 集群 / 哨兵 三种部署模式
- 完整的 String / Hash / List / Set / ZSet 数据结构操作
- Lua 脚本执行支持
- 分布式锁（带自动续期）
- 连接池配置
- 管道 / 事务支持
- 全局单例 + 实例化两种使用方式

## 使用方法

### 初始化

```go
import "vocara/pkg/config"
import "vocara/pkg/redis"

cfg := config.RedisConfig{
    Mode:         "single",      // single / cluster / sentinel
    Addrs:        "127.0.0.1:6379",
    Password:     "",
    DB:           0,
    PoolSize:     50,
    MinIdleConns: 10,
    DialTimeout:  5000,          // 毫秒
    ReadTimeout:  3000,
    WriteTimeout: 3000,
}

// 方式一：初始化全局客户端
if err := redis.Init(cfg); err != nil {
    log.Fatal(err)
}

// 方式二：创建独立客户端
client, err := redis.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 基本操作

```go
ctx := context.Background()
client := redis.Get()

// String 操作
client.Set(ctx, "key", "value", time.Hour)
val, err := client.Get(ctx, "key")
client.Del(ctx, "key")
client.Incr(ctx, "counter")
client.IncrBy(ctx, "counter", 10)

// Hash 操作
client.HSet(ctx, "user:1", "name", "张三", "age", 25)
name, _ := client.HGet(ctx, "user:1", "name")
user, _ := client.HGetAll(ctx, "user:1")
client.HDel(ctx, "user:1", "age")

// List 操作
client.LPush(ctx, "list", "a", "b", "c")
client.RPush(ctx, "list", "d")
val, _ := client.LPop(ctx, "list")
items, _ := client.LRange(ctx, "list", 0, -1)

// Set 操作
client.SAdd(ctx, "set", "a", "b", "c")
client.SIsMember(ctx, "set", "a")
members, _ := client.SMembers(ctx, "set")

// ZSet 操作
client.ZAdd(ctx, "zset", redis.Z{Score: 1, Member: "a"})
client.ZAdd(ctx, "zset", redis.Z{Score: 2, Member: "b"})
rank, _ := client.ZRank(ctx, "zset", "a")
topItems, _ := client.ZRevRange(ctx, "zset", 0, 9)
```

### Lua 脚本

```go
// 执行脚本
result, err := client.Eval(ctx, `
    return redis.call("incr", KEYS[1])
`, []string{"counter"})

// 加载脚本（可复用）
sha, err := client.ScriptLoad(ctx, script)
result, err = client.EvalSha(ctx, sha, []string{"counter"})
```

### 分布式锁

```go
import "vocara/pkg/redis"

// 创建锁
lock := client.NewDistributedLock("order:123", 10*time.Second)

// 尝试获取锁
ok, err := lock.Lock(ctx)
if err != nil {
    return err
}
if !ok {
    return errors.New("获取锁失败")
}
defer lock.Unlock(ctx)

// 启动自动续期（长任务使用）
lock.AutoRenew(ctx)

// 执行业务逻辑
doBusinessLogic()

// 或者使用 TryLock 带重试
ok, err = lock.TryLock(ctx, 3, 100*time.Millisecond)
```

#### 分布式锁特性

- **原子加锁**：使用 SET NX PX 保证原子性
- **防误解锁**：每个锁有唯一 value，只有持有者才能解锁
- **Lua 原子解锁**：使用 Lua 脚本保证检查和删除的原子性
- **自动续期**：长任务可启动自动续期，防止任务未完成锁过期
- **安全释放**：Unlock 会自动停止续期协程

### 管道操作

```go
pipe := client.Pipeline()
pipe.Set(ctx, "key1", "val1", 0)
pipe.Set(ctx, "key2", "val2", 0)
pipe.Get(ctx, "key1")
cmds, err := pipe.Exec(ctx)
```

### 其他操作

```go
// 检查 key 是否存在
exists, _ := client.Exists(ctx, "key")

// 设置过期时间
client.Expire(ctx, "key", time.Hour)

// 获取剩余过期时间
ttl, _ := client.TTL(ctx, "key")

// 扫描 key（推荐替代 keys）
var cursor uint64
for {
    keys, next, err := client.Scan(ctx, cursor, "prefix:*", 100)
    // 处理 keys
    cursor = next
    if cursor == 0 {
        break
    }
}
```

## 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| Mode | 模式: single/cluster/sentinel | single |
| Addrs | 地址列表（逗号分隔） | 127.0.0.1:6379 |
| Username | 用户名 | - |
| Password | 密码 | - |
| DB | 数据库编号 | 0 |
| MasterName | 哨兵主节点名称 | - |
| PoolSize | 连接池大小 | 50 |
| MinIdleConns | 最小空闲连接数 | 10 |
| MaxRetries | 最大重试次数 | 3 |
| DialTimeout | 连接超时(毫秒) | 5000 |
| ReadTimeout | 读取超时(毫秒) | 3000 |
| WriteTimeout | 写入超时(毫秒) | 3000 |
| PoolTimeout | 连接池超时(毫秒) | 4000 |
| IdleTimeout | 空闲超时(秒) | 300 |
