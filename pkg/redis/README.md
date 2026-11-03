# redis - Redis Wrapper Package

A Redis wrapper based on go-redis v9, supporting standalone, cluster, and sentinel modes, wrapping common operations, and providing distributed locks.

## Features

- Supports standalone, cluster, and sentinel deployment modes
- Comprehensive String / Hash / List / Set / ZSet data structure operations
- Lua script execution support
- Distributed locks with automatic renewal
- Connection pool configuration
- Pipeline and transaction support
- Both global singleton and instance-based usage

## Usage

### Initialization

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
    DialTimeout:  5000,          // milliseconds
    ReadTimeout:  3000,
    WriteTimeout: 3000,
}

// Option 1: Initialize the global client
if err := redis.Init(cfg); err != nil {
    log.Fatal(err)
}

// Option 2: Create an independent client
client, err := redis.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Basic Operations

```go
ctx := context.Background()
client := redis.Get()

// String operations
client.Set(ctx, "key", "value", time.Hour)
val, err := client.Get(ctx, "key")
client.Del(ctx, "key")
client.Incr(ctx, "counter")
client.IncrBy(ctx, "counter", 10)

// Hash operations
client.HSet(ctx, "user:1", "name", "John Doe", "age", 25)
name, _ := client.HGet(ctx, "user:1", "name")
user, _ := client.HGetAll(ctx, "user:1")
client.HDel(ctx, "user:1", "age")

// List operations
client.LPush(ctx, "list", "a", "b", "c")
client.RPush(ctx, "list", "d")
val, _ := client.LPop(ctx, "list")
items, _ := client.LRange(ctx, "list", 0, -1)

// Set operations
client.SAdd(ctx, "set", "a", "b", "c")
client.SIsMember(ctx, "set", "a")
members, _ := client.SMembers(ctx, "set")

// ZSet operations
client.ZAdd(ctx, "zset", redis.Z{Score: 1, Member: "a"})
client.ZAdd(ctx, "zset", redis.Z{Score: 2, Member: "b"})
rank, _ := client.ZRank(ctx, "zset", "a")
topItems, _ := client.ZRevRange(ctx, "zset", 0, 9)
```

### Lua Scripts

```go
// Execute a script
result, err := client.Eval(ctx, `
    return redis.call("incr", KEYS[1])
`, []string{"counter"})

// Load a script (reusable)
sha, err := client.ScriptLoad(ctx, script)
result, err = client.EvalSha(ctx, sha, []string{"counter"})
```

### Distributed Locks

```go
import "vocara/pkg/redis"

// Create a lock
lock := client.NewDistributedLock("order:123", 10*time.Second)

// Attempt to acquire the lock
ok, err := lock.Lock(ctx)
if err != nil {
    return err
}
if !ok {
    return errors.New("failed to acquire lock")
}
defer lock.Unlock(ctx)

// Start automatic renewal (for long-running tasks)
lock.AutoRenew(ctx)

// Execute business logic
doBusinessLogic()

// Or use TryLock with retries
ok, err = lock.TryLock(ctx, 3, 100*time.Millisecond)
```

#### Distributed Lock Features

- **Atomic locking**: Uses SET NX PX to ensure atomicity
- **Ownership-safe unlocking**: Each lock has a unique value, and only its owner can unlock it
- **Atomic unlocking with Lua**: Uses a Lua script to make checking and deletion atomic
- **Automatic renewal**: Long-running tasks can start automatic renewal to prevent lock expiration before completion
- **Safe release**: Unlock automatically stops the renewal goroutine

### Pipeline Operations

```go
pipe := client.Pipeline()
pipe.Set(ctx, "key1", "val1", 0)
pipe.Set(ctx, "key2", "val2", 0)
pipe.Get(ctx, "key1")
cmds, err := pipe.Exec(ctx)
```

### Other Operations

```go
// Check whether a key exists
exists, _ := client.Exists(ctx, "key")

// Set an expiration time
client.Expire(ctx, "key", time.Hour)

// Get the remaining time to live
ttl, _ := client.TTL(ctx, "key")

// Scan keys (preferred over keys)
var cursor uint64
for {
    keys, next, err := client.Scan(ctx, cursor, "prefix:*", 100)
    // Process keys
    cursor = next
    if cursor == 0 {
        break
    }
}
```

## Configuration Reference

| Configuration Item | Description | Default Value |
|--------|------|--------|
| Mode | Mode: single/cluster/sentinel | single |
| Addrs | Address list (comma-separated) | 127.0.0.1:6379 |
| Username | Username | - |
| Password | Password | - |
| DB | Database number | 0 |
| MasterName | Sentinel master node name | - |
| PoolSize | Connection pool size | 50 |
| MinIdleConns | Minimum number of idle connections | 10 |
| MaxRetries | Maximum number of retries | 3 |
| DialTimeout | Connection timeout (milliseconds) | 5000 |
| ReadTimeout | Read timeout (milliseconds) | 3000 |
| WriteTimeout | Write timeout (milliseconds) | 3000 |
| PoolTimeout | Connection pool timeout (milliseconds) | 4000 |
| IdleTimeout | Idle timeout (seconds) | 300 |
