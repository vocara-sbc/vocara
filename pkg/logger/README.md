# logger - 日志封装包

基于 Zap 的日志封装包，支持不同日志级别、结构化日志、控制台+文件输出、上下文透传。

## 功能特性

- 基于 Uber Zap，高性能结构化日志
- 支持 JSON / Console 两种输出格式
- 支持控制台 / 文件 / 同时输出三种模式
- 基于 Lumberjack 的日志文件切割（按大小、按天保留）
- 上下文透传（trace_id、user_id）
- 丰富的全局日志函数
- 支持 Sugared Logger 便捷调用

## 使用方法

### 初始化

```go
import "vocara/pkg/logger"
import "vocara/pkg/config"

cfg := config.LogConfig{
    Level:      "info",
    Format:     "json",
    Output:     "both",
    FilePath:   "./logs/app.log",
    MaxSize:    100,   // 100MB
    MaxAge:     30,    // 30天
    MaxBackups: 10,    // 10个备份
    Compress:   true,
    Caller:     true,
    CallerSkip: 1,
}

if err := logger.Init(cfg); err != nil {
    log.Fatal(err)
}
defer logger.Sync()
```

### 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| Level | 日志级别: debug/info/warn/error | info |
| Format | 输出格式: json/console | json |
| Output | 输出方式: stdout/file/both | both |
| FilePath | 日志文件路径 | ./logs/app.log |
| MaxSize | 单个文件最大大小(MB) | 100 |
| MaxAge | 文件保留天数 | 30 |
| MaxBackups | 最大备份文件数 | 10 |
| Compress | 是否压缩备份 | true |
| Caller | 是否显示调用位置 | true |
| CallerSkip | 调用栈跳过层数 | 1 |

### 基本日志输出

```go
// 普通日志
logger.Info("server started")
logger.Debug("debug info")
logger.Warn("warning message")
logger.Error("error occurred")

// 带字段的结构化日志
logger.Info("user login",
    zap.String("username", "admin"),
    zap.Int("user_id", 123),
    zap.Duration("duration", time.Second),
)
```

### 格式化日志

```go
logger.Infof("server started on port %d", 8080)
logger.Errorf("failed to connect: %v", err)
logger.Debugf("processing request: %s", reqID)
```

### 带字段的 Logger

```go
// 单个字段
l := logger.WithField("request_id", "req-123")
l.Info("processing request")
l.Info("request completed")

// 多个字段
l := logger.WithFields(map[string]interface{}{
    "user_id": 123,
    "action":  "login",
    "ip":      "192.168.1.1",
})
l.Info("user action")
```

### 上下文透传

```go
import "vocara/pkg/logger"

// 在中间件中设置 trace_id
ctx := context.WithValue(r.Context(), logger.TraceIDKey, "trace-abc123")
ctx = context.WithValue(ctx, logger.UserIDKey, "user-456")

// 在业务代码中使用
logger.InfoCtx(ctx, "processing request")
logger.ErrorCtx(ctx, "something went wrong")

// 或者获取带上下文字段的 logger
l := logger.WithContext(ctx)
l.Info("processing")
l.Warn("slow operation")
```

日志输出示例（JSON 格式）：
```json
{
    "time": "2024-01-01T12:00:00.000+0800",
    "level": "INFO",
    "caller": "service/user.go:42",
    "msg": "user login",
    "trace_id": "trace-abc123",
    "user_id": "user-456",
    "username": "admin"
}
```

### Sugared Logger

```go
sugar := logger.Sugar()
sugar.Infow("failed to fetch URL",
    "url", url,
    "attempt", 3,
    "backoff", time.Second,
)
```
