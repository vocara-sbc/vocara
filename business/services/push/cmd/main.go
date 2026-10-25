// Package main Vocara 推送服务入口
//
// 推送服务负责将座席状态变更、通话事件、队列监控等实时数据
// 通过 WebSocket 实时推送给前端。
//
// 主要功能：
// 1. WebSocket 连接管理（座席端、管理端、监控端）
// 2. 消息推送（座席状态、通话事件、队列状态、系统通知等）
// 3. 订阅机制（主题订阅/取消订阅）
// 4. 多节点路由（Redis 路由表 + NATS/Redis PubSub）
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.uber.org/zap"

	"vocara/pkg/config"
	"vocara/pkg/license"
	"vocara/pkg/logger"
	pkgmiddleware "vocara/pkg/middleware"
	pkgredis "vocara/pkg/redis"
	"vocara/pkg/version"
	"vocara/services/push/internal/consumer"
	"vocara/services/push/internal/handler"
	"vocara/services/push/internal/service"
)

func main() {
	// 1. 加载配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("load config failed: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	if err := logger.Init(cfg.Log); err != nil {
		fmt.Printf("init logger failed: %v\n", err)
		os.Exit(1)
	}
	// 静音 Hertz 框架日志（路由注册等直写 stderr），避免产生第二套日志文件
	hlog.SetOutput(io.Discard)
	defer logger.Sync()

	logger.Info("vocara push service starting...",
		zap.String("version", cfg.App.Version),
		zap.String("build_version", version.Version),
		zap.String("git_commit", version.GitCommit),
		zap.String("build_time", version.BuildTime),
		zap.String("env", cfg.App.Env),
	)

	// 3. 初始化 Redis（提前：授权原文与安装标识经 Redis 下发，授权加载前须先就绪）
	if err := initRedis(cfg); err != nil {
		logger.Fatal("init redis failed", zap.Error(err))
	}

	// License 授权：进程内验签，后台周期复校；异常不阻断启动，Redis/文件修复后于复校恢复
	licenseCtx, licenseCancel := context.WithCancel(context.Background())
	defer licenseCancel()
	licenseManager := license.NewManager(license.Config{
		File:          cfg.License.File,
		CheckInterval: cfg.License.CheckInterval,
		FailOpen:      cfg.License.FailOpen,
		InstallID:     cfg.License.InstallID,
	})
	// License 原文与安装标识来自 Redis；先注入来源再首次加载，保证重启后即可从 Redis 恢复授权
	licenseManager.SetSource(func() ([]byte, error) {
		cli := pkgredis.Get()
		if cli == nil {
			return nil, fmt.Errorf("redis not ready")
		}
		val, err := cli.Get(context.Background(), license.LicenseSourceKey)
		if err != nil {
			return nil, err
		}
		return []byte(val), nil
	})
	licenseManager.SetInstallIDProvider(license.InstallIDProviderFunc(func() (string, error) {
		cli := pkgredis.Get()
		if cli == nil {
			return "", nil
		}
		val, err := cli.Get(context.Background(), license.InstallIDSourceKey)
		if err != nil {
			return "", nil
		}
		return val, nil
	}))
	_ = licenseManager.Load()
	go licenseManager.Start(licenseCtx)

	// 订阅授权变更：admin 导入/重载后即时复校，无需等待下个复校周期；断线由 go-redis 自动重连
	go func() {
		cli := pkgredis.Get()
		if cli == nil {
			return
		}
		if err := cli.Subscribe(licenseCtx, license.LicenseChangedChannel, func(_ []byte) {
			if err := licenseManager.Load(); err != nil {
				logger.Warn("reload license on change failed", zap.Error(err))
				return
			}
			logger.Info("license reloaded on change notification")
		}); err != nil {
			logger.Warn("license change subscription ended", zap.Error(err))
		}
	}()

	lr := licenseManager.Result()
	logger.Info("license status",
		zap.String("state", string(lr.State)),
		zap.String("message", lr.Message),
	)

	// 4. 初始化核心服务
	connManager := service.NewConnectionManager()
	subService := service.NewSubscriptionService(connManager)

	// 节点ID（用于多节点路由）
	nodeID := fmt.Sprintf("push-%s", getNodeID())
	nodeAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	routerService := service.NewRouterService(nodeID, nodeAddr, cfg, connManager, subService)
	pushService := service.NewPushService(connManager, subService, routerService)

	// 座席命令网关（配置 agentcenter_base_url 时启用；鉴权复用 app.secret_key）
	var agentGateway *service.AgentGateway
	if cfg.Push.AgentCenterBaseURL != "" {
		acClient := service.NewAgentCenterClient(cfg.Push.AgentCenterBaseURL, cfg.App.SecretKey)
		agentGateway = service.NewAgentGateway(acClient, connManager)
	}

	// 5. 初始化 Handler
	wsHandler := handler.NewWebSocketHandler(
		connManager,
		subService,
		pushService,
		routerService,
		cfg.App.SecretKey,
		agentGateway,
	)

	pushHandler := handler.NewPushHandler(
		pushService,
		cfg.App.SecretKey, // 简化：使用 secretKey 作为内部 token
	)

	// 6. 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 7. 启动连接管理器
	connManager.Start(ctx)

	// 8. 启动路由服务
	if err := routerService.Start(ctx); err != nil {
		logger.Warn("start router service failed", zap.Error(err))
	}

	// 9. 启动消息消费者（消费其他服务发来的推送事件）
	startConsumer(ctx, cfg, connManager, subService, pushService)

	// 10. 启动 HTTP/WebSocket 服务
	h := setupRouter(cfg, wsHandler, pushHandler)

	// 11. 优雅关闭
	go handleShutdown(cancel, connManager, routerService, h)

	// 12. 启动服务
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("push service started",
		zap.String("addr", addr),
		zap.String("node_id", nodeID),
	)

	h.Spin()
}

// loadConfig 加载配置
func loadConfig() (*config.Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config failed: %w", err)
	}

	return cfg, nil
}

// initRedis 初始化 Redis
func initRedis(cfg *config.Config) error {
	if err := pkgredis.Init(cfg.Redis); err != nil {
		return fmt.Errorf("init redis failed: %w", err)
	}
	logger.Info("redis initialized")
	return nil
}

// setupRouter 设置路由
func setupRouter(
	cfg *config.Config,
	wsHandler *handler.WebSocketHandler,
	pushHandler *handler.PushHandler,
) *server.Hertz {
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		server.WithReadTimeout(time.Duration(cfg.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.Server.WriteTimeout)*time.Second),
		server.WithIdleTimeout(time.Duration(cfg.Server.IdleTimeout)*time.Second),
	)

	// 中间件
	h.Use(pkgmiddleware.Recovery())
	h.Use(pkgmiddleware.CORS())
	h.Use(pkgmiddleware.RequestLogger())
	h.Use(pkgmiddleware.TraceID())

	// ===== 健康检查 =====
	h.GET("/health", wsHandler.HealthCheck)
	h.GET("/api/v1/stats", wsHandler.GetStats)

	// ===== WebSocket 接口 =====
	wsGroup := h.Group("/ws")
	{
		// 座席端 WebSocket
		wsGroup.GET("/agent", wsHandler.AgentWS)
		// 管理端 WebSocket
		wsGroup.GET("/admin", wsHandler.AdminWS)
		// 监控大屏 WebSocket
		wsGroup.GET("/monitor", wsHandler.MonitorWS)
	}

	// ===== HTTP 推送接口（内部服务调用）=====
	pushGroup := h.Group("/api/v1/push")
	pushGroup.Use(internalAuth(cfg.App.SecretKey))
	{
		// 推送给指定座席
		pushGroup.POST("/to-agent", pushHandler.PushToAgent)
		// 推送给整个租户
		pushGroup.POST("/to-tenant", pushHandler.PushToTenant)
		// 广播给所有人
		pushGroup.POST("/to-all", pushHandler.PushToAll)
		// 推送到指定主题
		pushGroup.POST("/topic", pushHandler.PushToTopic)
		// 批量推送
		pushGroup.POST("/batch", pushHandler.PushBatch)
	}

	return h
}

// internalAuth 内部服务鉴权中间件
func internalAuth(token string) app.HandlerFunc {
	return pkgmiddleware.InternalAuth(token)
}

// startConsumer 启动消息消费者
func startConsumer(
	ctx context.Context,
	cfg *config.Config,
	connManager *service.ConnectionManager,
	subService *service.SubscriptionService,
	pushService *service.PushService,
) {
	// ===== 1. 启动 HTTP 推送接口消费者（消费 vocara.ws.push.* 主题的推送指令）=====
	// 根据配置决定使用哪种消费者
	// 优先使用 NATS，Redis 作为备选
	if cfg.NATS.URLs != "" {
		// NATS 消费者
		natsConsumer := consumer.NewNATSConsumer(connManager, subService, pushService, cfg.NATS)
		if err := natsConsumer.Start(ctx); err != nil {
			logger.Warn("start nats consumer failed, fallback to redis consumer", zap.Error(err))
			// 失败回退到 Redis 消费者
			startRedisConsumer(ctx, connManager, subService, pushService)
		}
	} else {
		// Redis Pub/Sub 消费者
		startRedisConsumer(ctx, connManager, subService, pushService)
	}

	// ===== 2. 启动业务事件消费者（消费 eventbus 上的业务事件，自动路由到 WebSocket 连接）=====
	startBusinessConsumer(ctx, cfg, subService, pushService)
}

// startBusinessConsumer 启动业务事件消费者
// 消费 agent.* / call.* / queue.* / system.notice 等业务事件，
// 根据事件类型自动路由到对应的 WebSocket 连接。
// 优先使用 NATS，如果 NATS 不可用则降级到 Redis Pub/Sub。
// 消费者启动失败只打警告日志，不影响主服务启动。
func startBusinessConsumer(
	ctx context.Context,
	cfg *config.Config,
	subService *service.SubscriptionService,
	pushService *service.PushService,
) {
	var businessConsumer *consumer.BusinessConsumer
	var startErr error

	if cfg.NATS.URLs != "" {
		// 优先使用 NATS
		businessConsumer = consumer.NewBusinessConsumerWithNATS(pushService, subService, cfg.NATS)
		startErr = businessConsumer.Start(ctx)
		if startErr == nil {
			return
		}
		logger.Warn("start business consumer with nats failed, fallback to redis",
			zap.Error(startErr),
		)
	}

	// 降级到 Redis
	redisClient := consumer.GetRedisClientFromPkg(pkgredis.Get())
	if redisClient == nil {
		logger.Warn("redis client not available, business consumer disabled")
		return
	}

	businessConsumer = consumer.NewBusinessConsumerWithRedis(pushService, subService, redisClient)
	if err := businessConsumer.Start(ctx); err != nil {
		logger.Warn("start business consumer with redis failed", zap.Error(err))
	}
}

// startRedisConsumer 启动 Redis 消费者
func startRedisConsumer(
	ctx context.Context,
	connManager *service.ConnectionManager,
	subService *service.SubscriptionService,
	pushService *service.PushService,
) {
	redisClient := pkgredis.Get()
	if redisClient == nil {
		logger.Warn("redis client not available, consumer disabled")
		return
	}

	redisConsumer := consumer.NewRedisConsumer(connManager, subService, pushService, redisClient)
	if err := redisConsumer.Start(ctx); err != nil {
		logger.Error("start redis consumer failed", zap.Error(err))
	}
}

// handleShutdown 处理优雅关闭
func handleShutdown(
	cancel context.CancelFunc,
	connManager *service.ConnectionManager,
	routerService *service.RouterService,
	h *server.Hertz,
) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// 取消上下文，通知所有 goroutine 停止
	cancel()

	// 停止路由服务
	routerService.Stop()

	// 停止连接管理器
	connManager.Stop()

	// 关闭 Hertz 服务（给一个超时时间）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	_ = h.Shutdown(shutdownCtx)

	logger.Info("push service shutdown complete")
	os.Exit(0)
}

// getNodeID 获取节点标识
// 优先使用环境变量 NODE_ID，否则使用主机名
func getNodeID() string {
	nodeID := os.Getenv("NODE_ID")
	if nodeID != "" {
		return nodeID
	}

	hostname, err := os.Hostname()
	if err == nil && hostname != "" {
		return hostname
	}

	// 最后兜底：使用时间戳
	return fmt.Sprintf("%d", time.Now().Unix())
}
