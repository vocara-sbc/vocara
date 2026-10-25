// Package main ACD 服务启动入口
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
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"vocara/pkg/config"
	"vocara/pkg/eventbus"
	"vocara/pkg/license"
	"vocara/pkg/logger"
	"vocara/pkg/middleware"
	pkgredis "vocara/pkg/redis"
	"vocara/pkg/response"
	"vocara/pkg/version"
	"vocara/services/acd/internal/handler"
	"vocara/services/acd/internal/repository"
	"vocara/services/acd/internal/service"
)

func main() {
	// 1. 加载配置
	configPath := "configs/config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("load config failed: %v", err))
	}

	// 2. 初始化日志
	if err := logger.Init(cfg.Log); err != nil {
		panic(fmt.Sprintf("init logger failed: %v", err))
	}
	// 静音 Hertz 框架日志（路由注册等直写 stderr），避免产生第二套日志文件
	hlog.SetOutput(io.Discard)
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("acd service starting",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("build_version", version.Version),
		zap.String("git_commit", version.GitCommit),
		zap.String("build_time", version.BuildTime),
		zap.String("env", cfg.App.Env),
	)

	// 3. 初始化 Redis（提前：授权原文与安装标识经 Redis 下发，授权加载前须先就绪）
	if err := pkgredis.Init(cfg.Redis); err != nil {
		logger.Fatal("init redis failed", zap.Error(err))
	}
	rdb := pkgredis.Get().Raw()
	logger.Info("redis connected")

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

	// 4. 初始化仓储层
	acdRepo := repository.NewAcdRedisRepo(rdb)
	statsRepo := repository.NewStatsRepo(rdb)

	// 5. 初始化事件发布器（内部调度用，基于 Redis Pub/Sub）
	eventBus := service.NewRedisEventPublisher(acdRepo)

	// 6. 初始化统一事件总线发布者（用于 WebSocket 推送等）
	busPublisher := initBusPublisher(cfg, rdb)
	if busPublisher != nil {
		logger.Info("event bus publisher initialized")
	} else {
		logger.Warn("event bus publisher not available, skip bus events")
	}
	defer func() {
		if busPublisher != nil {
			_ = busPublisher.Close()
		}
	}()

	// 7. 初始化座席中心客户端
	agentClient := createAgentCenterClient(cfg)
	logger.Info("agent center client initialized",
		zap.Bool("useMock", cfg.ACD.UseMockAgentClient),
		zap.String("addr", cfg.ACD.AgentCenterAddr),
	)

	// 8. 初始化服务层
	acdConfig := service.ACDConfig{
		MaxQueueLen:        cfg.ACD.MaxQueueLen,
		MaxPriority:        cfg.ACD.MaxPriority,
		PriorityAgingCycle: cfg.ACD.PriorityAgingCycle,
		MaxWaitTime:        cfg.ACD.MaxWaitTime,
	}

	acdService := service.NewAcdService(acdRepo, statsRepo, eventBus, busPublisher, acdConfig)
	monitorService := service.NewMonitorService(acdRepo, statsRepo)

	// 9. 初始化调度器
	schedConfig := service.SchedulerConfig{
		ScheduleInterval:   time.Duration(cfg.ACD.ScheduleInterval) * time.Second,
		LockTimeout:        time.Duration(cfg.ACD.LockTimeout) * time.Second,
		AllocateMaxRetries: cfg.ACD.AllocateMaxRetries,
	}
	scheduler := service.NewScheduler(acdRepo, eventBus, agentClient, acdConfig, schedConfig, busPublisher)

	// 10. 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		logger.Fatal("start scheduler failed", zap.Error(err))
	}
	logger.Info("scheduler started")

	// 11. 初始化 HTTP 服务
	h := initHTTPServer(cfg, acdService, monitorService)

	// 12. 启动 HTTP 服务（后台）
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("http server starting", zap.String("addr", addr))
		h.Spin()
	}()

	// 13. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("acd service shutting down...")

	// 14. 优雅关闭
	cancel()
	scheduler.Stop()
	logger.Info("scheduler stopped")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer shutdownCancel()
	_ = h.Shutdown(shutdownCtx)

	logger.Info("acd service stopped")
}

// createAgentCenterClient 创建座席中心客户端
func createAgentCenterClient(cfg *config.Config) service.AgentCenterClient {
	if cfg.ACD.UseMockAgentClient || cfg.ACD.AgentCenterAddr == "" {
		return service.NewMockAgentClient()
	}
	return service.NewHTTPAgentCenterClient(
		cfg.ACD.AgentCenterAddr,
		cfg.ACD.AgentCenterToken,
	)
}

// initBusPublisher 初始化统一事件总线发布者
// NATS 优先，Redis 降级；都不可用时返回 nil（跳过业务事件发布）
func initBusPublisher(cfg *config.Config, rdb redis.UniversalClient) eventbus.Publisher {
	var natsPub *eventbus.NATSPublisher

	// Redis 发布者（始终可用）
	redisPub := eventbus.NewRedisPublisher(rdb)

	// 尝试连接 NATS（可选，失败不影响启动）
	if cfg.NATS.URLs != "" {
		opts := []nats.Option{
			nats.Name(cfg.NATS.ClientID),
			nats.MaxReconnects(cfg.NATS.MaxReconnect),
			nats.ReconnectWait(time.Duration(cfg.NATS.ReconnectWait) * time.Second),
			nats.Timeout(time.Duration(cfg.NATS.Timeout) * time.Second),
		}
		if cfg.NATS.User != "" {
			opts = append(opts, nats.UserInfo(cfg.NATS.User, cfg.NATS.Password))
		}
		if cfg.NATS.Token != "" {
			opts = append(opts, nats.Token(cfg.NATS.Token))
		}
		natsUrls := cfg.NATS.URLList()
		nc, err := nats.Connect(natsUrls[0], opts...)
		if err != nil {
			logger.Warn("connect nats failed, fallback to redis pubsub",
				zap.Error(err),
			)
		} else {
			natsPub = eventbus.NewNATSPublisher(nc)
			logger.Info("nats connected for event bus")
		}
	}

	// 创建双通道发布者
	if natsPub != nil && redisPub != nil {
		pub, err := eventbus.NewDualPublisher(natsPub, redisPub)
		if err != nil {
			logger.Warn("create dual publisher failed", zap.Error(err))
			return redisPub
		}
		return pub
	}

	if redisPub != nil {
		return redisPub
	}

	return nil
}

// initHTTPServer 初始化 HTTP 服务
func initHTTPServer(cfg *config.Config, acdService service.AcdService, monitorService service.MonitorService) *server.Hertz {
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		server.WithReadTimeout(time.Duration(cfg.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.Server.WriteTimeout)*time.Second),
		server.WithIdleTimeout(time.Duration(cfg.Server.IdleTimeout)*time.Second),
	)

	// 中间件
	h.Use(middleware.TraceID())
	h.Use(middleware.Recovery())
	h.Use(middleware.CORS())
	h.Use(middleware.RequestLogger())

	// 健康检查
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.Success(c, utils.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// API v1 路由组
	v1 := h.Group("/api/v1/acd")

	// ACD 处理器
	acdHandler := handler.NewAcdHandler(acdService)
	{
		v1.POST("/enqueue", acdHandler.Enqueue)
		v1.POST("/dequeue", acdHandler.Dequeue)
		v1.POST("/pause", acdHandler.Pause)
		v1.POST("/resume", acdHandler.Resume)
		v1.GET("/call/:callId", acdHandler.GetCallInfo)
	}

	// 监控处理器
	monitorHandler := handler.NewMonitorHandler(monitorService)
	{
		v1.GET("/monitor/queue/:queueId", monitorHandler.MonitorQueue)
		v1.GET("/monitor/queues", monitorHandler.MonitorAllQueues)
	}

	return h
}
