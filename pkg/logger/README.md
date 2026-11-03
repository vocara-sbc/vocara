# logger - Logging Wrapper Package

A logging wrapper based on Zap, supporting multiple log levels, structured logging, console and file output, and context propagation.

## Features

- High-performance structured logging based on Uber Zap
- Supports JSON and Console output formats
- Supports console, file, and simultaneous output modes
- Log file rotation with Lumberjack (by size and retention days)
- Context propagation (trace_id, user_id)
- Extensive global logging functions
- Supports convenient Sugared Logger calls

## Usage

### Initialization

```go
import "vocara/pkg/logger"
import "vocara/pkg/config"

cfg := config.LogConfig{
    Level:      "info",
    Format:     "json",
    Output:     "both",
    FilePath:   "./logs/app.log",
    MaxSize:    100,   // 100MB
    MaxAge:     30,    // 30 days
    MaxBackups: 10,    // 10 backups
    Compress:   true,
    Caller:     true,
    CallerSkip: 1,
}

if err := logger.Init(cfg); err != nil {
    log.Fatal(err)
}
defer logger.Sync()
```

### Configuration Reference

| Configuration Item | Description | Default Value |
|--------|------|--------|
| Level | Log level: debug/info/warn/error | info |
| Format | Output format: json/console | json |
| Output | Output mode: stdout/file/both | both |
| FilePath | Log file path | ./logs/app.log |
| MaxSize | Maximum size of a single file (MB) | 100 |
| MaxAge | File retention period in days | 30 |
| MaxBackups | Maximum number of backup files | 10 |
| Compress | Whether to compress backups | true |
| Caller | Whether to show the caller location | true |
| CallerSkip | Number of call stack frames to skip | 1 |

### Basic Logging

```go
// Standard logs
logger.Info("server started")
logger.Debug("debug info")
logger.Warn("warning message")
logger.Error("error occurred")

// Structured logs with fields
logger.Info("user login",
    zap.String("username", "admin"),
    zap.Int("user_id", 123),
    zap.Duration("duration", time.Second),
)
```

### Formatted Logging

```go
logger.Infof("server started on port %d", 8080)
logger.Errorf("failed to connect: %v", err)
logger.Debugf("processing request: %s", reqID)
```

### Logger with Fields

```go
// Single field
l := logger.WithField("request_id", "req-123")
l.Info("processing request")
l.Info("request completed")

// Multiple fields
l := logger.WithFields(map[string]interface{}{
    "user_id": 123,
    "action":  "login",
    "ip":      "192.168.1.1",
})
l.Info("user action")
```

### Context Propagation

```go
import "vocara/pkg/logger"

// Set trace_id in middleware
ctx := context.WithValue(r.Context(), logger.TraceIDKey, "trace-abc123")
ctx = context.WithValue(ctx, logger.UserIDKey, "user-456")

// Use in business code
logger.InfoCtx(ctx, "processing request")
logger.ErrorCtx(ctx, "something went wrong")

// Or get a logger with context fields
l := logger.WithContext(ctx)
l.Info("processing")
l.Warn("slow operation")
```

Example log output (JSON format):
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
