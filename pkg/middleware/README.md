# middleware - HTTP Middleware Package

A collection of HTTP middleware based on the Hertz framework, including authentication, logging, recovery, CORS, TraceID, and more.

## Features

- JWT authentication middleware
- Request logging middleware
- Panic recovery middleware
- CORS middleware
- TraceID middleware
- Internal service authentication middleware

## Usage

### JWT Authentication Middleware

```go
import "vocara/pkg/middleware"

// Basic usage
h.Use(middleware.JWT("your-secret-key"))

// With configuration
h.Use(middleware.JWTWithConfig(middleware.JWTConfig{
    Secret:     "your-secret-key",
    ContextKey: "user",
    SkipPaths:  []string{"/api/login", "/api/health"},
    TokenType:  "Bearer",
}))

// Get user information from the context
func handler(c context.Context, ctx *app.RequestContext) {
    claims := middleware.GetUserClaims(ctx)
    userID := middleware.GetUserID(ctx)
    username := middleware.GetUsername(ctx)
    role := middleware.GetUserRole(ctx)
}
```

#### Generating a Token

```go
import "vocara/pkg/middleware"

token, err := middleware.GenerateToken(
    "user-123",    // userID
    "zhangsan",    // username
    "admin",       // role
    "secret-key",  // secret
)
```

### Request Logging Middleware

```go
import "vocara/pkg/middleware"

// Default configuration
h.Use(middleware.RequestLogger())

// Custom configuration
h.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
    SkipPaths:       []string{"/health", "/metrics"},
    SlowThreshold:   2000,  // Requests taking 2 seconds are considered slow
    MaxBodySize:     2048,  // Log up to 2KB of the body
}))
```

Log fields:
- `method` - Request method
- `path` - Request path
- `status` - Response status code
- `duration` - Request duration
- `ip` - Client IP
- `trace_id` - Trace ID (used with TraceID middleware)

Log levels:
- 5xx errors -> Error level
- 4xx errors -> Warn level
- Slow requests -> Warn level
- Others -> Info level

### Panic Recovery Middleware

```go
import "vocara/pkg/middleware"

// Default configuration
h.Use(middleware.Recovery())

// Custom configuration
h.Use(middleware.RecoveryWithConfig(middleware.RecoveryConfig{
    StackSize:  8192, // 8KB stack
    PrintStack: true,
}))
```

When a panic occurs:
- Logs the error, including stack information
- Returns a 500 system error response
- Keeps the service running

### CORS Middleware

```go
import "vocara/pkg/middleware"

// Default configuration (allows all origins)
h.Use(middleware.CORS())

// Custom configuration
h.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins:     "https://example.com",
    AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
    AllowHeaders:     "Content-Type, Authorization",
    AllowCredentials: true,
    ExposeHeaders:    "X-Trace-ID",
    MaxAge:           86400,
}))
```

### TraceID Middleware

```go
import "vocara/pkg/middleware"

// Default configuration
h.Use(middleware.TraceID())

// Custom configuration
h.Use(middleware.TraceIDWithConfig(middleware.TraceIDConfig{
    HeaderKey:  "X-Request-ID",
    ContextKey: "request_id",
    GenFunc:    func() string { return utils.UUID() },
}))

// Get the TraceID
func handler(c context.Context, ctx *app.RequestContext) {
    traceID := middleware.GetTraceID(ctx)
    // Or get it from the logger context
    l := logger.WithContext(c)
    l.Info("processing request")
}
```

TraceID workflow:
1. Reads the `X-Trace-ID` request header
2. Automatically generates a value (UUID v4) if none is present
3. Sets the `X-Trace-ID` response header
4. Stores the value in the context for automatic output with logger middleware

### Internal Service Authentication

```go
import "vocara/pkg/middleware"

// Internal services authenticate with X-Internal-Token
h.Use(middleware.InternalAuth("your-internal-token"))
```

### Recommended Order

```go
h := server.Default()

// 1. Recovery (outermost layer, catches all panics)
h.Use(middleware.Recovery())

// 2. TraceID (generates a trace ID for subsequent middleware)
h.Use(middleware.TraceID())

// 3. CORS (handles cross-origin preflight requests)
h.Use(middleware.CORS())

// 4. Request logging (logs all requests)
h.Use(middleware.RequestLogger())

// 5. Authentication (route groups requiring login)
authGroup := h.Group("/api")
authGroup.Use(middleware.JWT("secret"))

// 6. Business routes
authGroup.GET("/user", UserHandler)
```
