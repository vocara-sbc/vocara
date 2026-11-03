// Package main is the ACD service entry point.
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
	// 1. Load configuration.
	configPath := "configs/config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("load config failed: %v", err))
	}

	// 2. Initialize logging.
	if err := logger.Init(cfg.Log); err != nil {
		panic(fmt.Sprintf("init logger failed: %v", err))
	}
	// Silence Hertz framework logs, such as route registration written directly to stderr, to avoid a second log file.
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

	// 3. Initialize Redis early because the license source and installation ID come from Redis and must be ready before license loading.
	if err := pkgredis.Init(cfg.Redis); err != nil {
		logger.Fatal("init redis failed", zap.Error(err))
	}
	rdb := pkgredis.Get().Raw()
	logger.Info("redis connected")

	// License verification runs in process with periodic background checks; errors do not block startup, and verification resumes after Redis or file recovery.
	licenseCtx, licenseCancel := context.WithCancel(context.Background())
	defer licenseCancel()
	licenseManager := license.NewManager(license.Config{
		File:          cfg.License.File,
		CheckInterval: cfg.License.CheckInterval,
		FailOpen:      cfg.License.FailOpen,
		InstallID:     cfg.License.InstallID,
	})
	// The license source and installation ID come from Redis; inject the source before the first load so restarts can restore the license from Redis.
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

	// Subscribe to license changes: admin imports and reloads trigger immediate verification without waiting for the next cycle; go-redis reconnects automatically.
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

	// 4. Initialize the repository layer.
	acdRepo := repository.NewAcdRedisRepo(rdb)
	statsRepo := repository.NewStatsRepo(rdb)

	// 5. Initialize the Redis Pub/Sub event publisher for internal scheduling.
	eventBus := service.NewRedisEventPublisher(acdRepo)

	// 6. Initialize the unified event bus publisher for WebSocket push and other consumers.
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

	// 7. Initialize the agent center client.
	agentClient := createAgentCenterClient(cfg)
	logger.Info("agent center client initialized",
		zap.Bool("useMock", cfg.ACD.UseMockAgentClient),
		zap.String("addr", cfg.ACD.AgentCenterAddr),
	)

	// 8. Initialize the service layer.
	acdConfig := service.ACDConfig{
		MaxQueueLen:        cfg.ACD.MaxQueueLen,
		MaxPriority:        cfg.ACD.MaxPriority,
		PriorityAgingCycle: cfg.ACD.PriorityAgingCycle,
		MaxWaitTime:        cfg.ACD.MaxWaitTime,
	}

	acdService := service.NewAcdService(acdRepo, statsRepo, eventBus, busPublisher, acdConfig)
	monitorService := service.NewMonitorService(acdRepo, statsRepo)

	// 9. Initialize the scheduler.
	schedConfig := service.SchedulerConfig{
		ScheduleInterval:   time.Duration(cfg.ACD.ScheduleInterval) * time.Second,
		LockTimeout:        time.Duration(cfg.ACD.LockTimeout) * time.Second,
		AllocateMaxRetries: cfg.ACD.AllocateMaxRetries,
	}
	scheduler := service.NewScheduler(acdRepo, eventBus, agentClient, acdConfig, schedConfig, busPublisher)

	// 10. Start the scheduler.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		logger.Fatal("start scheduler failed", zap.Error(err))
	}
	logger.Info("scheduler started")

	// 11. Initialize the HTTP server.
	h := initHTTPServer(cfg, acdService, monitorService)

	// 12. Start the HTTP server in the background.
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("http server starting", zap.String("addr", addr))
		h.Spin()
	}()

	// 13. Wait for an exit signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("acd service shutting down...")

	// 14. Shut down gracefully.
	cancel()
	scheduler.Stop()
	logger.Info("scheduler stopped")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer shutdownCancel()
	_ = h.Shutdown(shutdownCtx)

	logger.Info("acd service stopped")
}

// createAgentCenterClient creates an agent center client.
func createAgentCenterClient(cfg *config.Config) service.AgentCenterClient {
	if cfg.ACD.UseMockAgentClient || cfg.ACD.AgentCenterAddr == "" {
		return service.NewMockAgentClient()
	}
	return service.NewHTTPAgentCenterClient(
		cfg.ACD.AgentCenterAddr,
		cfg.ACD.AgentCenterToken,
	)
}

// initBusPublisher initializes the unified event bus publisher.
// NATS is preferred with Redis fallback; nil is returned when neither is available, skipping business event publishing.
func initBusPublisher(cfg *config.Config, rdb redis.UniversalClient) eventbus.Publisher {
	var natsPub *eventbus.NATSPublisher

	// The Redis publisher is always available.
	redisPub := eventbus.NewRedisPublisher(rdb)

	// Attempt the optional NATS connection; failure does not prevent startup.
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

	// Create the dual-channel publisher.
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

// initHTTPServer initializes the HTTP server.
func initHTTPServer(cfg *config.Config, acdService service.AcdService, monitorService service.MonitorService) *server.Hertz {
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		server.WithReadTimeout(time.Duration(cfg.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.Server.WriteTimeout)*time.Second),
		server.WithIdleTimeout(time.Duration(cfg.Server.IdleTimeout)*time.Second),
	)

	// Middleware.
	h.Use(middleware.TraceID())
	h.Use(middleware.Recovery())
	h.Use(middleware.CORS())
	h.Use(middleware.RequestLogger())

	// Health check.
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.Success(c, utils.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// API v1 route group.
	v1 := h.Group("/api/v1/acd")

	// ACD handler.
	acdHandler := handler.NewAcdHandler(acdService)
	{
		v1.POST("/enqueue", acdHandler.Enqueue)
		v1.POST("/dequeue", acdHandler.Dequeue)
		v1.POST("/pause", acdHandler.Pause)
		v1.POST("/resume", acdHandler.Resume)
		v1.GET("/call/:callId", acdHandler.GetCallInfo)
	}

	// Monitoring handler.
	monitorHandler := handler.NewMonitorHandler(monitorService)
	{
		v1.GET("/monitor/queue/:queueId", monitorHandler.MonitorQueue)
		v1.GET("/monitor/queues", monitorHandler.MonitorAllQueues)
	}

	return h
}
