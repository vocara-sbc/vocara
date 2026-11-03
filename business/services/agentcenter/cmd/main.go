package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vocara/pkg/config"
	"vocara/pkg/database"
	"vocara/pkg/eventbus"
	"vocara/pkg/license"
	"vocara/pkg/logger"
	"vocara/pkg/middleware"
	pkgredis "vocara/pkg/redis"
	"vocara/pkg/version"
	"vocara/services/agentcenter/internal/handler"
	"vocara/services/agentcenter/internal/repository"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	// 1. Load the configuration.
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

	logger.Info("starting agent center service...",
		zap.String("build_version", version.Version),
		zap.String("git_commit", version.GitCommit),
		zap.String("build_time", version.BuildTime),
	)

	// 3. Initialize Redis first because license content and the installation ID are delivered through Redis and must be ready before license loading.
	if err := pkgredis.Init(cfg.Redis); err != nil {
		logger.Fatal("init redis failed", zap.Error(err))
	}
	logger.Info("redis initialized")

	// Verify licenses in the process and revalidate them periodically in the background; errors do not block startup, and revalidation recovers after Redis or the file is repaired.
	licenseCtx, licenseCancel := context.WithCancel(context.Background())
	defer licenseCancel()
	licenseManager := license.NewManager(license.Config{
		File:          cfg.License.File,
		CheckInterval: cfg.License.CheckInterval,
		FailOpen:      cfg.License.FailOpen,
		InstallID:     cfg.License.InstallID,
	})
	// Load license content and the installation ID from Redis; configure these sources before the first load so restarts can restore the license from Redis.
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

	// Subscribe to license changes so admin imports or reloads trigger immediate revalidation; go-redis reconnects automatically after disconnects.
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

	// 4. Initialize MySQL.
	if err := database.Init(cfg.MySQL); err != nil {
		logger.Fatal("init mysql failed", zap.Error(err))
	}
	logger.Info("mysql initialized")

	// 5. Initialize optional NATS connectivity without blocking service startup on failure.
	var natsConn *nats.Conn
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
			logger.Warn("init nats failed, will use redis only",
				zap.Error(err),
				zap.Strings("urls", natsUrls),
			)
		} else {
			natsConn = nc
			logger.Info("nats initialized")
			defer func() {
				_ = natsConn.Drain()
			}()
		}
	}

	// 6. Initialize the event publisher with NATS preferred and Redis as fallback.
	rdb := pkgredis.Get().Raw()
	redisPub := eventbus.NewRedisPublisher(rdb)

	var eventPublisher eventbus.Publisher
	if natsConn != nil {
		natsPub := eventbus.NewNATSPublisher(natsConn)
		dualPub, err := eventbus.NewDualPublisher(natsPub, redisPub)
		if err != nil {
			logger.Warn("create dual publisher failed, fallback to redis only", zap.Error(err))
			eventPublisher = redisPub
		} else {
			eventPublisher = dualPub
			logger.Info("dual event publisher initialized (nats + redis)")
		}
	} else {
		eventPublisher = redisPub
		logger.Info("redis event publisher initialized (nats not available)")
	}
	defer func() {
		_ = eventPublisher.Close()
	}()

	// 7. Initialize repositories.
	db := database.Get()
	agentRepo := repository.NewAgentRepo(db.DB)
	dnRepo := repository.NewDnRepo(db.DB)
	skillRepo := repository.NewSkillRepo(db.DB)
	onlineAgentRepo := repository.NewOnlineAgentRepo(db.DB)

	// 8. Initialize services.
	sessionSvc := service.NewSessionService(rdb, getSessionTTL(cfg))
	agentSvc := service.NewAgentService(
		agentRepo, dnRepo, skillRepo, onlineAgentRepo, sessionSvc,
		getJWTSecret(cfg),
		"vocara-agent-center", // issuer
		getJWTExpireHour(cfg),
		eventPublisher,
		licenseManager,
	)
	dnSvc := service.NewDnService(dnRepo, sessionSvc)
	skillSvc := service.NewSkillService(skillRepo, agentRepo, sessionSvc)

	// 9. Initialize handlers.
	agentHandler := handler.NewAgentHandler(agentSvc)
	dnHandler := handler.NewDnHandler(dnSvc)
	skillHandler := handler.NewSkillHandler(skillSvc)
	manageHandler := handler.NewManageHandler(agentSvc)

	// 10. Create the Hertz server.
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	h := server.Default(
		server.WithHostPorts(addr),
		server.WithReadTimeout(time.Duration(cfg.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.Server.WriteTimeout)*time.Second),
	)

	// Middleware
	h.Use(middleware.TraceID())
	h.Use(middleware.Recovery())
	h.Use(middleware.CORS())
	h.Use(middleware.RequestLogger())

	// Register routes.
	registerRoutes(h, agentHandler, dnHandler, skillHandler, manageHandler,
		agentSvc, getInternalToken(cfg))

	// 11. Start the expiration scanner.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startExpiredSessionScanner(ctx, rdb, sessionSvc, getScanInterval(cfg))

	// 12. Shut down gracefully.
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		logger.Info("received signal, shutting down...", zap.String("signal", sig.String()))
		cancel()
		h.Shutdown(ctx)
	}()

	logger.Info("agent center service started", zap.String("addr", addr))
	h.Spin()
}

// getSessionTTL returns the session TTL from configuration, defaulting to 120 seconds.
func getSessionTTL(cfg *config.Config) int {
	// Return the default directly in this simplified implementation instead of reading custom configuration.
	// config.Config can be extended for project-specific settings.
	return 120
}

// getScanInterval returns the scan interval, defaulting to 30000 milliseconds.
func getScanInterval(cfg *config.Config) int {
	return 30000
}

// getJWTSecret returns the JWT secret.
func getJWTSecret(cfg *config.Config) string {
	// Read app.secret_key.
	return cfg.App.SecretKey
}

// getJWTExpireHour returns the JWT expiration time in hours.
func getJWTExpireHour(cfg *config.Config) int {
	return 24 // default to 24 hours
}

// getInternalToken returns the internal service authentication token.
func getInternalToken(cfg *config.Config) string {
	// Use app.secret_key as the internal token; it can also be configured separately.
	return cfg.App.SecretKey
}

// registerRoutes registers HTTP routes.
func registerRoutes(
	h *server.Hertz,
	agentHandler *handler.AgentHandler,
	dnHandler *handler.DnHandler,
	skillHandler *handler.SkillHandler,
	manageHandler *handler.ManageHandler,
	agentSvc *service.AgentService,
	internalToken string,
) {
	// Health check
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"status": "ok", "service": "agent-center"})
	})

	// Agent JWT authentication middleware
	agentAuth := handler.AgentAuth(agentSvc)

	// ===== Agent API =====
	agentGroup := h.Group("/api/v1/agent")
	{
		// Public endpoints that do not require login
		agentGroup.POST("/login", agentHandler.Login)
		agentGroup.POST("/heartbeat/:agentId", agentHandler.Heartbeat)

		// Endpoints that require authentication
		authGroup := agentGroup.Group("", agentAuth)
		{
			authGroup.POST("/logout", agentHandler.Logout)
			authGroup.POST("/checkin", agentHandler.Checkin)
			authGroup.POST("/checkout", agentHandler.Checkout)
			authGroup.POST("/busy", agentHandler.Busy)
			authGroup.POST("/ready", agentHandler.Ready)
			authGroup.GET("/info", agentHandler.GetInfo)
			authGroup.GET("/status", agentHandler.GetStatus)
		}
	}

	// ===== Extension API =====
	dnGroup := h.Group("/api/v1/dn", agentAuth)
	{
		dnGroup.GET("/list", dnHandler.List)
		dnGroup.GET("/:dnId", dnHandler.Get)
		dnGroup.POST("/checkonline", dnHandler.CheckOnline)
	}

	// ===== Skill group API =====
	skillGroup := h.Group("/api/v1/skill", agentAuth)
	{
		skillGroup.GET("/list", skillHandler.List)
		skillGroup.GET("/:id", skillHandler.Get)
		skillGroup.GET("/:id/agents", skillHandler.GetAgents)
	}

	// ===== Internal management API for ACD =====
	internalGroup := h.Group("/api/v1/internal", middleware.InternalAuth(internalToken))
	{
		internalGroup.POST("/agent/try-acquire", manageHandler.TryAcquireAgent)
		internalGroup.POST("/agent/ring", manageHandler.Ring)
		internalGroup.POST("/agent/resume-ready", manageHandler.ResumeReady)
		internalGroup.POST("/agent/call-answer", manageHandler.CallAnswer)
		internalGroup.POST("/agent/call-end", manageHandler.CallEnd)
		internalGroup.POST("/agent/update-status", manageHandler.UpdateStatus)
		internalGroup.POST("/agent/session", manageHandler.EstablishSession)
		internalGroup.POST("/agent/session/refresh", manageHandler.RefreshSession)
		internalGroup.POST("/agent/session/delete", manageHandler.DeleteSession)
	}
}

// startExpiredSessionScanner starts the expired session scanner.
// It periodically finds agents in online sets whose sessions have expired and marks them OFFLINE.
func startExpiredSessionScanner(
	ctx context.Context,
	rdb redis.UniversalClient,
	sessionSvc *service.SessionService,
	scanIntervalMs int,
) {
	if scanIntervalMs <= 0 {
		scanIntervalMs = 30000 // default to 30 seconds
	}

	ticker := time.NewTicker(time.Duration(scanIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	logger.Info("expired session scanner started", zap.Int("interval_ms", scanIntervalMs))

	for {
		select {
		case <-ctx.Done():
			logger.Info("expired session scanner stopped")
			return
		case <-ticker.C:
			scanExpiredSessions(ctx, rdb, sessionSvc)
		}
	}
}

// scanExpiredSessions scans for expired sessions across all tenants.
func scanExpiredSessions(
	ctx context.Context,
	rdb redis.UniversalClient,
	sessionSvc *service.SessionService,
) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("scan expired sessions panic", zap.Any("recover", r))
		}
	}()

	// Scan all online set keys.
	var cursor uint64
	pattern := "vocara:agent:online:*"
	prefixLen := len("vocara:agent:online:")
	tenantsScanned := 0
	agentsOfflined := 0

	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Error("scan online keys failed", zap.Error(err))
			return
		}

		for _, key := range keys {
			if len(key) <= prefixLen {
				continue
			}
			tenantID := key[prefixLen:]

			// Scan for expired agents in this tenant.
			expiredAgents, err := sessionSvc.ScanExpiredSessions(ctx, tenantID)
			if err != nil {
				logger.Warn("scan expired sessions for tenant failed",
					zap.String("tenant_id", tenantID),
					zap.Error(err),
				)
				continue
			}

			for _, agentID := range expiredAgents {
				logger.Warn("agent session expired, force offline",
					zap.String("tenant_id", tenantID),
					zap.String("agent_id", agentID),
				)

				// Clean up resources by reading skill groups from the session and removing the agent from idle queues.
				session, _ := sessionSvc.GetSession(ctx, tenantID, agentID)
				if session != nil && len(session.SkillGroups) > 0 {
					_ = sessionSvc.RemoveIdleBatch(ctx, agentID, session.SkillGroups)
				}

				// Delete the skill group cache.
				_ = sessionSvc.DeleteAgentSkills(ctx, agentID)

				// Remove the agent from the online set.
				_ = sessionSvc.RemoveOnline(ctx, tenantID, agentID)

				// Delete the session.
				_ = sessionSvc.DeleteSession(ctx, tenantID, agentID)

				agentsOfflined++
			}

			tenantsScanned++
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if agentsOfflined > 0 {
		logger.Info("expired session scan complete",
			zap.Int("tenants_scanned", tenantsScanned),
			zap.Int("agents_offlined", agentsOfflined),
		)
	}
}
