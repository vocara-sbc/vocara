// Package main is the entry point for the Vocara push service.
//
// The push service delivers real-time data such as agent status changes,
// call events, and queue monitoring to frontends over WebSocket.
//
// Main features:
// 1. WebSocket connection management for agent, admin, and monitor clients
// 2. Message push for agent status, call events, queue status, system notices, and more
// 3. Topic subscription and unsubscription
// 4. Multi-node routing using a Redis routing table and NATS/Redis PubSub
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
	// 1. Load configuration.
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("load config failed: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logging.
	if err := logger.Init(cfg.Log); err != nil {
		fmt.Printf("init logger failed: %v\n", err)
		os.Exit(1)
	}
	// Silence Hertz framework logs written directly to stderr, such as route registration, to avoid a second set of log files.
	hlog.SetOutput(io.Discard)
	defer logger.Sync()

	logger.Info("vocara push service starting...",
		zap.String("version", cfg.App.Version),
		zap.String("build_version", version.Version),
		zap.String("git_commit", version.GitCommit),
		zap.String("build_time", version.BuildTime),
		zap.String("env", cfg.App.Env),
	)

	// 3. Initialize Redis before license loading because Redis supplies the license text and installation ID.
	if err := initRedis(cfg); err != nil {
		logger.Fatal("init redis failed", zap.Error(err))
	}

	// License authorization verifies signatures in-process and revalidates periodically; errors do not block startup and validation recovers after Redis or file issues are resolved.
	licenseCtx, licenseCancel := context.WithCancel(context.Background())
	defer licenseCancel()
	licenseManager := license.NewManager(license.Config{
		File:          cfg.License.File,
		CheckInterval: cfg.License.CheckInterval,
		FailOpen:      cfg.License.FailOpen,
		InstallID:     cfg.License.InstallID,
	})
	// Redis supplies the license text and installation ID. Set both sources before the initial load so restarts can recover authorization from Redis.
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

	// Subscribe to authorization changes for immediate revalidation after admin imports or reloads; go-redis reconnects automatically after disconnections.
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

	// 4. Initialize core services.
	connManager := service.NewConnectionManager()
	subService := service.NewSubscriptionService(connManager)

	// Node ID used for multi-node routing.
	nodeID := fmt.Sprintf("push-%s", getNodeID())
	nodeAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	routerService := service.NewRouterService(nodeID, nodeAddr, cfg, connManager, subService)
	pushService := service.NewPushService(connManager, subService, routerService)

	// Agent command gateway, enabled when agentcenter_base_url is configured; authentication reuses app.secret_key.
	var agentGateway *service.AgentGateway
	if cfg.Push.AgentCenterBaseURL != "" {
		acClient := service.NewAgentCenterClient(cfg.Push.AgentCenterBaseURL, cfg.App.SecretKey)
		agentGateway = service.NewAgentGateway(acClient, connManager)
	}

	// 5. Initialize handlers.
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
		cfg.App.SecretKey, // Use secretKey as the internal token.
	)

	// 6. Create the context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 7. Start the connection manager.
	connManager.Start(ctx)

	// 8. Start the router service.
	if err := routerService.Start(ctx); err != nil {
		logger.Warn("start router service failed", zap.Error(err))
	}

	// 9. Start message consumers for push events from other services.
	startConsumer(ctx, cfg, connManager, subService, pushService)

	// 10. Start the HTTP/WebSocket service.
	h := setupRouter(cfg, wsHandler, pushHandler)

	// 11. Set up graceful shutdown.
	go handleShutdown(cancel, connManager, routerService, h)

	// 12. Start the service.
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("push service started",
		zap.String("addr", addr),
		zap.String("node_id", nodeID),
	)

	h.Spin()
}

// loadConfig loads the configuration.
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

// initRedis initializes Redis.
func initRedis(cfg *config.Config) error {
	if err := pkgredis.Init(cfg.Redis); err != nil {
		return fmt.Errorf("init redis failed: %w", err)
	}
	logger.Info("redis initialized")
	return nil
}

// setupRouter configures the routes.
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

	// Middleware.
	h.Use(pkgmiddleware.Recovery())
	h.Use(pkgmiddleware.CORS())
	h.Use(pkgmiddleware.RequestLogger())
	h.Use(pkgmiddleware.TraceID())

	// ===== Health checks =====
	h.GET("/health", wsHandler.HealthCheck)
	h.GET("/api/v1/stats", wsHandler.GetStats)

	// ===== WebSocket APIs =====
	wsGroup := h.Group("/ws")
	{
		// Agent WebSocket.
		wsGroup.GET("/agent", wsHandler.AgentWS)
		// Admin WebSocket.
		wsGroup.GET("/admin", wsHandler.AdminWS)
		// Monitoring dashboard WebSocket.
		wsGroup.GET("/monitor", wsHandler.MonitorWS)
	}

	// ===== HTTP push APIs for internal service calls =====
	pushGroup := h.Group("/api/v1/push")
	pushGroup.Use(internalAuth(cfg.App.SecretKey))
	{
		// Push to a specific agent.
		pushGroup.POST("/to-agent", pushHandler.PushToAgent)
		// Push to an entire tenant.
		pushGroup.POST("/to-tenant", pushHandler.PushToTenant)
		// Broadcast to everyone.
		pushGroup.POST("/to-all", pushHandler.PushToAll)
		// Push to a specific topic.
		pushGroup.POST("/topic", pushHandler.PushToTopic)
		// Batch push.
		pushGroup.POST("/batch", pushHandler.PushBatch)
	}

	return h
}

// internalAuth provides internal service authentication middleware.
func internalAuth(token string) app.HandlerFunc {
	return pkgmiddleware.InternalAuth(token)
}

// startConsumer starts message consumers.
func startConsumer(
	ctx context.Context,
	cfg *config.Config,
	connManager *service.ConnectionManager,
	subService *service.SubscriptionService,
	pushService *service.PushService,
) {
	// ===== 1. Start the HTTP push API consumer for push commands on vocara.ws.push.* topics =====
	// Select the consumer based on configuration.
	// Prefer NATS with Redis as an alternative.
	if cfg.NATS.URLs != "" {
		// NATS consumer.
		natsConsumer := consumer.NewNATSConsumer(connManager, subService, pushService, cfg.NATS)
		if err := natsConsumer.Start(ctx); err != nil {
			logger.Warn("start nats consumer failed, fallback to redis consumer", zap.Error(err))
			// Fall back to the Redis consumer on failure.
			startRedisConsumer(ctx, connManager, subService, pushService)
		}
	} else {
		// Redis Pub/Sub consumer.
		startRedisConsumer(ctx, connManager, subService, pushService)
	}

	// ===== 2. Start the business event consumer to route eventbus events to WebSocket connections =====
	startBusinessConsumer(ctx, cfg, subService, pushService)
}

// startBusinessConsumer starts the business event consumer.
// It consumes business events such as agent.*, call.*, queue.*, and system.notice,
// and routes them to WebSocket connections by event type.
// It prefers NATS and falls back to Redis Pub/Sub when NATS is unavailable.
// Consumer startup failures only log warnings and do not prevent the main service from starting.
func startBusinessConsumer(
	ctx context.Context,
	cfg *config.Config,
	subService *service.SubscriptionService,
	pushService *service.PushService,
) {
	var businessConsumer *consumer.BusinessConsumer
	var startErr error

	if cfg.NATS.URLs != "" {
		// Prefer NATS.
		businessConsumer = consumer.NewBusinessConsumerWithNATS(pushService, subService, cfg.NATS)
		startErr = businessConsumer.Start(ctx)
		if startErr == nil {
			return
		}
		logger.Warn("start business consumer with nats failed, fallback to redis",
			zap.Error(startErr),
		)
	}

	// Fall back to Redis.
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

// startRedisConsumer starts the Redis consumer.
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

// handleShutdown handles graceful shutdown.
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

	// Cancel the context to notify all goroutines to stop.
	cancel()

	// Stop the router service.
	routerService.Stop()

	// Stop the connection manager.
	connManager.Stop()

	// Shut down the Hertz service with a timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	_ = h.Shutdown(shutdownCtx)

	logger.Info("push service shutdown complete")
	os.Exit(0)
}

// getNodeID returns the node identifier.
// It prefers the NODE_ID environment variable and otherwise uses the hostname.
func getNodeID() string {
	nodeID := os.Getenv("NODE_ID")
	if nodeID != "" {
		return nodeID
	}

	hostname, err := os.Hostname()
	if err == nil && hostname != "" {
		return hostname
	}

	// Use a timestamp as the final fallback.
	return fmt.Sprintf("%d", time.Now().Unix())
}
