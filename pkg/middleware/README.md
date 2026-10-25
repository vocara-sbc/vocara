# middleware - HTTP 中间件包

基于 Hertz 框架的 HTTP 中间件集合，包括鉴权、日志、恢复、CORS、TraceID 等。

## 功能特性

- JWT 鉴权中间件
- 请求日志中间件
- Panic 恢复中间件
- CORS 跨域中间件
- TraceID 追踪中间件
- 内部服务鉴权中间件

## 使用方法

### JWT 鉴权中间件

```go
import "vocara/pkg/middleware"

// 简单使用
h.Use(middleware.JWT("your-secret-key"))

// 带配置
h.Use(middleware.JWTWithConfig(middleware.JWTConfig{
    Secret:     "your-secret-key",
    ContextKey: "user",
    SkipPaths:  []string{"/api/login", "/api/health"},
    TokenType:  "Bearer",
}))

// 从上下文获取用户信息
func handler(c context.Context, ctx *app.RequestContext) {
    claims := middleware.GetUserClaims(ctx)
    userID := middleware.GetUserID(ctx)
    username := middleware.GetUsername(ctx)
    role := middleware.GetUserRole(ctx)
}
```

#### 生成 Token

```go
import "vocara/pkg/middleware"

token, err := middleware.GenerateToken(
    "user-123",    // userID
    "zhangsan",    // username
    "admin",       // role
    "secret-key",  // secret
)
```

### 请求日志中间件

```go
import "vocara/pkg/middleware"

// 默认配置
h.Use(middleware.RequestLogger())

// 自定义配置
h.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
    SkipPaths:       []string{"/health", "/metrics"},
    SlowThreshold:   2000,  // 2秒视为慢请求
    MaxBodySize:     2048,  // 最大记录 2KB body
}))
```

日志字段：
- `method` - 请求方法
- `path` - 请求路径
- `status` - 响应状态码
- `duration` - 请求耗时
- `ip` - 客户端 IP
- `trace_id` - 追踪 ID（配合 TraceID 中间件）

日志级别：
- 5xx 错误 -> Error 级别
- 4xx 错误 -> Warn 级别
- 慢请求 -> Warn 级别
- 其他 -> Info 级别

### Panic 恢复中间件

```go
import "vocara/pkg/middleware"

// 默认配置
h.Use(middleware.Recovery())

// 自定义配置
h.Use(middleware.RecoveryWithConfig(middleware.RecoveryConfig{
    StackSize:  8192, // 8KB 堆栈
    PrintStack: true,
}))
```

发生 panic 时：
- 记录错误日志（包含堆栈信息）
- 返回 500 系统错误响应
- 保证服务不崩溃

### CORS 跨域中间件

```go
import "vocara/pkg/middleware"

// 默认配置（允许所有源）
h.Use(middleware.CORS())

// 自定义配置
h.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins:     "https://example.com",
    AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
    AllowHeaders:     "Content-Type, Authorization",
    AllowCredentials: true,
    ExposeHeaders:    "X-Trace-ID",
    MaxAge:           86400,
}))
```

### TraceID 追踪中间件

```go
import "vocara/pkg/middleware"

// 默认配置
h.Use(middleware.TraceID())

// 自定义配置
h.Use(middleware.TraceIDWithConfig(middleware.TraceIDConfig{
    HeaderKey:  "X-Request-ID",
    ContextKey: "request_id",
    GenFunc:    func() string { return utils.UUID() },
}))

// 获取 TraceID
func handler(c context.Context, ctx *app.RequestContext) {
    traceID := middleware.GetTraceID(ctx)
    // 或从 logger 上下文获取
    l := logger.WithContext(c)
    l.Info("processing request")
}
```

TraceID 工作流程：
1. 从请求头 `X-Trace-ID` 获取
2. 如果没有则自动生成（UUID v4）
3. 设置到响应头 `X-Trace-ID`
4. 存入上下文，配合 logger 中间件自动输出

### 内部服务鉴权

```go
import "vocara/pkg/middleware"

// 内部服务使用 X-Internal-Token 鉴权
h.Use(middleware.InternalAuth("your-internal-token"))
```

### 推荐使用顺序

```go
h := server.Default()

// 1. Recovery（最外层，捕获所有 panic）
h.Use(middleware.Recovery())

// 2. TraceID（生成追踪 ID，后续中间件都能用到）
h.Use(middleware.TraceID())

// 3. CORS（处理跨域预检请求）
h.Use(middleware.CORS())

// 4. 请求日志（记录所有请求）
h.Use(middleware.RequestLogger())

// 5. 鉴权（需要登录的路由组）
authGroup := h.Group("/api")
authGroup.Use(middleware.JWT("secret"))

// 6. 业务路由
authGroup.GET("/user", UserHandler)
```
