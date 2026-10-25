# Vocara PKG - Go 公共库模块

Vocara 项目的公共库模块，提供配置管理、日志、错误码、Redis、数据库、工具函数、中间件、统一响应等基础能力。

## 模块信息

- **模块名**: `vocara/pkg`
- **Go 版本**: Go 1.22
- **框架依赖**: Hertz (HTTP), GORM (数据库), go-redis (Redis)

## 目录结构

```
pkg/
├── go.mod
├── README.md
├── config/          # 配置管理
│   ├── config.go
│   ├── config_test.go
│   └── README.md
├── logger/          # 日志封装
│   ├── logger.go
│   ├── logger_test.go
│   └── README.md
├── errors/          # 错误码封装
│   ├── errors.go
│   ├── code.go
│   ├── errors_test.go
│   └── README.md
├── redis/           # Redis 封装
│   ├── redis.go
│   ├── lock.go
│   └── README.md
├── eventbus/        # 事件总线封装
│   ├── event.go
│   ├── nats.go
│   ├── redis.go
│   ├── dual.go
│   └── README.md
├── database/        # 数据库封装
│   ├── database.go
│   ├── pagination.go
│   ├── database_test.go
│   └── README.md
├── utils/           # 工具函数
│   ├── string.go
│   ├── time.go
│   ├── uuid.go
│   ├── crypto.go
│   ├── number.go
│   ├── copier.go
│   ├── string_test.go
│   ├── utils_test.go
│   └── README.md
├── middleware/      # HTTP 中间件（Hertz）
│   ├── auth.go
│   ├── auth_test.go
│   ├── logger.go
│   ├── recovery.go
│   ├── cors.go
│   ├── trace.go
│   ├── rbac/        # 角色与权限码定义
│   └── README.md
├── response/        # 统一响应封装
│   ├── response.go
│   ├── response_test.go
│   └── README.md
├── license/         # License 校验运行时
│   ├── manager.go
│   ├── bootstrap.go
│   ├── binding.go
│   ├── claims.go
│   ├── envelope.go
│   ├── fingerprint.go
│   └── ...
└── version/         # 构建期版本信息
    └── version.go
```

## 包说明

### [config](./config/README.md) - 配置管理

基于 Viper 的配置管理，支持 YAML 配置文件和环境变量覆盖。

- 完整的配置结构体（App/Server/MySQL/Redis/NATS/Log/ESL/CallCtrl/FSDialplan/ACD/Push/License）
- 丰富的默认值
- 环境变量覆盖（前缀 `VOCARA_`）
- DSN/地址列表等工具方法

### [logger](./logger/README.md) - 日志封装

基于 Zap 的高性能结构化日志封装。

- JSON/Console 两种输出格式
- 控制台 + 文件输出（Lumberjack 日志切割）
- 上下文透传（trace_id、user_id）
- 丰富的全局日志函数

### [errors](./errors/README.md) - 错误码封装

自定义业务错误类型，包含错误码、消息、HTTP 状态码。

- `BizError` 业务错误结构体
- 预定义常用错误码（按业务分类）
- 错误包装 / 解包
- 自动推断 HTTP 状态码

### [redis](./redis/README.md) - Redis 封装

基于 go-redis v9 的 Redis 封装。

- 单机 / 集群 / 哨兵 三种模式
- String/Hash/List/Set/ZSet 常用操作
- Lua 脚本执行
- 分布式锁（带自动续期）
- 连接池配置

### [database](./database/README.md) - 数据库封装

基于 GORM v2 + MySQL 的数据库封装。

- 连接池配置
- 自动迁移（可选）
- 分页查询封装
- 软删除支持
- 上下文透传

### [utils](./utils/README.md) - 工具函数

常用工具函数集合。

- 字符串工具（随机字符串、格式转换）
- UUID 生成
- 时间工具（格式化、计算）
- 数字工具（转换、四舍五入）
- IP 工具（本机 IP、客户端 IP）
- 加密工具（MD5、SHA256）
- 结构体拷贝

### [middleware](./middleware/README.md) - HTTP 中间件

基于 Hertz 框架的中间件集合。

- JWT 鉴权中间件
- 请求日志中间件
- Panic 恢复中间件
- CORS 跨域中间件
- TraceID 追踪中间件
- 内部服务鉴权

### [response](./response/README.md) - 统一响应封装

统一的 HTTP 响应格式。

- 成功/失败响应
- 分页响应
- 与 errors 包无缝集成
- 自动设置 HTTP 状态码
- 丰富的快捷方法

### [eventbus](./eventbus/README.md) - 事件总线封装

统一事件信封与三种发布者实现。

- 信封字段（event_id/event_type/timestamp/tenant_id/source/data）
- NATS、Redis、Dual（NATS 优先、Redis 降级）三种发布者
- 主题命名 `vocara.event.{type}`

### [license](./license/) - License 校验运行时

离线 License 的加载、验签与状态管理。

- 机器指纹绑定
- 到期 / 容量 / 模块开关拦截
- 与厂商侧 `license-cli`（核心私有仓库 `tools/license-cli`）签发体系配套

### [version](./version/) - 版本信息

暴露 `Version`、`BuildTime`、`GitCommit`，由构建期 `-ldflags "-X"` 注入。

## 快速开始

### 安装

```bash
go get vocara/pkg
```

### 基本使用

```go
import (
    "vocara/pkg/config"
    "vocara/pkg/logger"
    "vocara/pkg/errors"
    "vocara/pkg/redis"
    "vocara/pkg/database"
)

// 加载配置
cfg, err := config.Load("./config.yaml")
if err != nil {
    log.Fatal(err)
}

// 初始化日志
logger.Init(cfg.Log)
defer logger.Sync()

// 初始化 Redis
if err := redis.Init(cfg.Redis); err != nil {
    logger.Fatal("redis init failed", zap.Error(err))
}

// 初始化数据库
if err := database.Init(cfg.MySQL); err != nil {
    logger.Fatal("mysql init failed", zap.Error(err))
}

// 使用
logger.Info("server started")
```

### Hertz 服务示例

```go
import (
    "github.com/cloudwego/hertz/pkg/app/server"
    "vocara/pkg/middleware"
)

func main() {
    h := server.Default()

    // 中间件
    h.Use(middleware.Recovery())
    h.Use(middleware.TraceID())
    h.Use(middleware.CORS())
    h.Use(middleware.RequestLogger())

    // 路由
    h.GET("/health", func(c context.Context, ctx *app.RequestContext) {
        response.OK(ctx, map[string]string{"status": "ok"})
    })

    h.Spin()
}
```

## 依赖列表

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/spf13/viper | v1.19.0 | 配置管理 |
| go.uber.org/zap | v1.27.0 | 日志 |
| github.com/natefinch/lumberjack | v2.0.0 | 日志切割 |
| github.com/redis/go-redis/v9 | v9.7.0 | Redis |
| gorm.io/gorm | v1.25.12 | ORM |
| gorm.io/driver/mysql | v1.5.7 | MySQL 驱动 |
| github.com/cloudwego/hertz | v0.10.1 | HTTP 框架 |
| github.com/nats-io/nats.go | v1.36.0 | NATS 客户端 |
| github.com/golang-jwt/jwt/v5 | v5.2.1 | JWT |
| github.com/google/uuid | v1.6.0 | UUID |
| github.com/jinzhu/copier | v0.4.0 | 结构体拷贝 |
| golang.org/x/crypto | v0.28.0 | 加密 |

## 开发规范

- 每个包提供清晰的 README 文档
- 公共函数需要添加注释
- 核心功能需要单元测试
- 错误处理使用 `errors.BizError`
- 日志使用 `logger` 包
- 配置使用 `config` 包

## License

MIT
