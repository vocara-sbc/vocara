# Vocara PKG - Go Shared Library Modules

The shared library modules for the Vocara project provide foundational capabilities for configuration management, logging, error codes, Redis, databases, utility functions, middleware, unified responses, and more.

## Module Information

- **Module Name**: `vocara/pkg`
- **Go Version**: Go 1.22
- **Framework Dependencies**: Hertz (HTTP), GORM (database), go-redis (Redis)

## Directory Structure

```
pkg/
├── go.mod
├── README.md
├── config/          # Configuration management
│   ├── config.go
│   ├── config_test.go
│   └── README.md
├── logger/          # Logging wrapper
│   ├── logger.go
│   ├── logger_test.go
│   └── README.md
├── errors/          # Error code wrapper
│   ├── errors.go
│   ├── code.go
│   ├── errors_test.go
│   └── README.md
├── redis/           # Redis wrapper
│   ├── redis.go
│   ├── lock.go
│   └── README.md
├── eventbus/        # Event bus wrapper
│   ├── event.go
│   ├── nats.go
│   ├── redis.go
│   ├── dual.go
│   └── README.md
├── database/        # Database wrapper
│   ├── database.go
│   ├── pagination.go
│   ├── database_test.go
│   └── README.md
├── utils/           # Utility functions
│   ├── string.go
│   ├── time.go
│   ├── uuid.go
│   ├── crypto.go
│   ├── number.go
│   ├── copier.go
│   ├── string_test.go
│   ├── utils_test.go
│   └── README.md
├── middleware/      # HTTP middleware (Hertz)
│   ├── auth.go
│   ├── auth_test.go
│   ├── logger.go
│   ├── recovery.go
│   ├── cors.go
│   ├── trace.go
│   ├── rbac/        # Role and permission code definitions
│   └── README.md
├── response/        # Unified response wrapper
│   ├── response.go
│   ├── response_test.go
│   └── README.md
├── license/         # License validation runtime
│   ├── manager.go
│   ├── bootstrap.go
│   ├── binding.go
│   ├── claims.go
│   ├── envelope.go
│   ├── fingerprint.go
│   └── ...
└── version/         # Build-time version information
    └── version.go
```

## Package Overview

### [config](./config/README.md) - Configuration Management

Configuration management based on Viper, with support for YAML configuration files and environment variable overrides.

- Comprehensive configuration structs (App/Server/MySQL/Redis/NATS/Log/ESL/CallCtrl/FSDialplan/ACD/Push/License)
- Extensive default values
- Environment variable overrides (prefix `VOCARA_`)
- Utility methods for DSNs, address lists, and more

### [logger](./logger/README.md) - Logging Wrapper

High-performance structured logging based on Zap.

- JSON and Console output formats
- Console and file output (log rotation with Lumberjack)
- Context propagation (trace_id, user_id)
- Extensive global logging functions

### [errors](./errors/README.md) - Error Code Wrapper

Custom business error types containing an error code, message, and HTTP status code.

- `BizError` business error struct
- Predefined common error codes organized by business domain
- Error wrapping and unwrapping
- Automatic HTTP status code inference

### [redis](./redis/README.md) - Redis Wrapper

Redis wrapper based on go-redis v9.

- Standalone, cluster, and sentinel modes
- Common String/Hash/List/Set/ZSet operations
- Lua script execution
- Distributed locks with automatic renewal
- Connection pool configuration

### [database](./database/README.md) - Database Wrapper

Database wrapper based on GORM v2 and MySQL.

- Connection pool configuration
- Optional automatic migration
- Paginated query wrapper
- Soft delete support
- Context propagation

### [utils](./utils/README.md) - Utility Functions

A collection of common utility functions.

- String utilities (random strings and format conversion)
- UUID generation
- Time utilities (formatting and calculation)
- Number utilities (conversion and rounding)
- IP utilities (local IP and client IP)
- Encryption utilities (MD5 and SHA256)
- Struct copying

### [middleware](./middleware/README.md) - HTTP Middleware

A collection of middleware based on the Hertz framework.

- JWT authentication middleware
- Request logging middleware
- Panic recovery middleware
- CORS middleware
- TraceID middleware
- Internal service authentication

### [response](./response/README.md) - Unified Response Wrapper

A unified HTTP response format.

- Success and failure responses
- Paginated responses
- Seamless integration with the errors package
- Automatic HTTP status code setting
- Extensive convenience methods

### [eventbus](./eventbus/README.md) - Event Bus Wrapper

A unified event envelope and three publisher implementations.

- Envelope fields (event_id/event_type/timestamp/tenant_id/source/data)
- NATS, Redis, and Dual (NATS first with Redis fallback) publishers
- Topic naming: `vocara.event.{type}`

### [license](./license/) - License Validation Runtime

Loading, signature verification, and state management for offline licenses.

- Machine fingerprint binding
- Expiration, capacity, and module feature gating
- Works with the vendor-side `license-cli` issuance system (core private repository `tools/license-cli`)

### [version](./version/) - Version Information

Exposes `Version`, `BuildTime`, and `GitCommit`, injected at build time with `-ldflags "-X"`.

## Quick Start

### Installation

```bash
go get vocara/pkg
```

### Basic Usage

```go
import (
    "vocara/pkg/config"
    "vocara/pkg/logger"
    "vocara/pkg/errors"
    "vocara/pkg/redis"
    "vocara/pkg/database"
)

// Load configuration
cfg, err := config.Load("./config.yaml")
if err != nil {
    log.Fatal(err)
}

// Initialize logging
logger.Init(cfg.Log)
defer logger.Sync()

// Initialize Redis
if err := redis.Init(cfg.Redis); err != nil {
    logger.Fatal("redis init failed", zap.Error(err))
}

// Initialize the database
if err := database.Init(cfg.MySQL); err != nil {
    logger.Fatal("mysql init failed", zap.Error(err))
}

// Usage
logger.Info("server started")
```

### Hertz Service Example

```go
import (
    "github.com/cloudwego/hertz/pkg/app/server"
    "vocara/pkg/middleware"
)

func main() {
    h := server.Default()

    // Middleware
    h.Use(middleware.Recovery())
    h.Use(middleware.TraceID())
    h.Use(middleware.CORS())
    h.Use(middleware.RequestLogger())

    // Routes
    h.GET("/health", func(c context.Context, ctx *app.RequestContext) {
        response.OK(ctx, map[string]string{"status": "ok"})
    })

    h.Spin()
}
```

## Dependency List

| Dependency | Version | Purpose |
|------|------|------|
| github.com/spf13/viper | v1.19.0 | Configuration management |
| go.uber.org/zap | v1.27.0 | Logging |
| github.com/natefinch/lumberjack | v2.0.0 | Log rotation |
| github.com/redis/go-redis/v9 | v9.7.0 | Redis |
| gorm.io/gorm | v1.25.12 | ORM |
| gorm.io/driver/mysql | v1.5.7 | MySQL driver |
| github.com/cloudwego/hertz | v0.10.1 | HTTP framework |
| github.com/nats-io/nats.go | v1.36.0 | NATS client |
| github.com/golang-jwt/jwt/v5 | v5.2.1 | JWT |
| github.com/google/uuid | v1.6.0 | UUID |
| github.com/jinzhu/copier | v0.4.0 | Struct copying |
| golang.org/x/crypto | v0.28.0 | Encryption |

## Development Guidelines

- Provide clear README documentation for each package
- Add comments to public functions
- Provide unit tests for core functionality
- Use `errors.BizError` for error handling
- Use the `logger` package for logging
- Use the `config` package for configuration

## License

MIT
