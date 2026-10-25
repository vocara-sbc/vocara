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

	logger.Info("starting agent center service...",
		zap.String("build_version", version.Version),
		zap.String("git_commit", version.GitCommit),
		zap.String("build_time", version.BuildTime),
	)

	// 3. 初始化 Redis（提前：授权原文与安装标识经 Redis 下发，授权加载前须先就绪）
	if err := pkgredis.Init(cfg.Redis); err != nil {
		logger.Fatal("init redis failed", zap.Error(err))
	}
	logger.Info("redis initialized")

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

	// 4. 初始化 MySQL
	if err := database.Init(cfg.MySQL); err != nil {
		logger.Fatal("init mysql failed", zap.Error(err))
	}
	logger.Info("mysql initialized")

	// 5. 初始化 NATS（可选，失败不影响服务启动）
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

	// 6. 初始化事件发布者（双通道：NATS 优先，Redis 降级）
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

	// 7. 初始化 Repository
	db := database.Get()
	agentRepo := repository.NewAgentRepo(db.DB)
	dnRepo := repository.NewDnRepo(db.DB)
	skillRepo := repository.NewSkillRepo(db.DB)
	onlineAgentRepo := repository.NewOnlineAgentRepo(db.DB)

	// 8. 初始化 Service
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

	// 9. 初始化 Handler
	agentHandler := handler.NewAgentHandler(agentSvc)
	dnHandler := handler.NewDnHandler(dnSvc)
	skillHandler := handler.NewSkillHandler(skillSvc)
	manageHandler := handler.NewManageHandler(agentSvc)

	// 10. 创建 Hertz 服务
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	h := server.Default(
		server.WithHostPorts(addr),
		server.WithReadTimeout(time.Duration(cfg.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.Server.WriteTimeout)*time.Second),
	)

	// 中间件
	h.Use(middleware.TraceID())
	h.Use(middleware.Recovery())
	h.Use(middleware.CORS())
	h.Use(middleware.RequestLogger())

	// 路由注册
	registerRoutes(h, agentHandler, dnHandler, skillHandler, manageHandler,
		agentSvc, getInternalToken(cfg))

	// 11. 启动过期扫描器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startExpiredSessionScanner(ctx, rdb, sessionSvc, getScanInterval(cfg))

	// 12. 优雅关闭
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

// getSessionTTL 获取会话 TTL，从配置中读取，默认 120 秒
func getSessionTTL(cfg *config.Config) int {
	// 尝试从自定义配置读取，这里简化直接返回默认值
	// 实际项目中可以扩展 config.Config
	return 120
}

// getScanInterval 获取扫描间隔，默认 30000 毫秒
func getScanInterval(cfg *config.Config) int {
	return 30000
}

// getJWTSecret 获取 JWT 密钥
func getJWTSecret(cfg *config.Config) string {
	// 从 app.secret_key 读取
	return cfg.App.SecretKey
}

// getJWTExpireHour 获取 JWT 过期时间（小时）
func getJWTExpireHour(cfg *config.Config) int {
	return 24 // 默认 24 小时
}

// getInternalToken 获取内部服务鉴权 Token
func getInternalToken(cfg *config.Config) string {
	// 使用 app.secret_key 作为内部 token，实际可单独配置
	return cfg.App.SecretKey
}

// registerRoutes 注册路由
func registerRoutes(
	h *server.Hertz,
	agentHandler *handler.AgentHandler,
	dnHandler *handler.DnHandler,
	skillHandler *handler.SkillHandler,
	manageHandler *handler.ManageHandler,
	agentSvc *service.AgentService,
	internalToken string,
) {
	// 健康检查
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"status": "ok", "service": "agent-center"})
	})

	// 座席 JWT 鉴权中间件
	agentAuth := handler.AgentAuth(agentSvc)

	// ===== 座席接口 =====
	agentGroup := h.Group("/api/v1/agent")
	{
		// 公开接口（不需要登录）
		agentGroup.POST("/login", agentHandler.Login)
		agentGroup.POST("/heartbeat/:agentId", agentHandler.Heartbeat)

		// 需要鉴权的接口
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

	// ===== 分机接口 =====
	dnGroup := h.Group("/api/v1/dn", agentAuth)
	{
		dnGroup.GET("/list", dnHandler.List)
		dnGroup.GET("/:dnId", dnHandler.Get)
		dnGroup.POST("/checkonline", dnHandler.CheckOnline)
	}

	// ===== 技能组接口 =====
	skillGroup := h.Group("/api/v1/skill", agentAuth)
	{
		skillGroup.GET("/list", skillHandler.List)
		skillGroup.GET("/:id", skillHandler.Get)
		skillGroup.GET("/:id/agents", skillHandler.GetAgents)
	}

	// ===== 内部管理接口（供 ACD 调用） =====
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

// startExpiredSessionScanner 启动过期会话扫描器
// 定时扫描在线集合中 session 已过期的座席，将其置为 OFFLINE
func startExpiredSessionScanner(
	ctx context.Context,
	rdb redis.UniversalClient,
	sessionSvc *service.SessionService,
	scanIntervalMs int,
) {
	if scanIntervalMs <= 0 {
		scanIntervalMs = 30000 // 默认 30 秒
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

// scanExpiredSessions 扫描所有租户下的过期会话
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

	// 扫描所有在线集合 key
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

			// 扫描该租户下过期的座席
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

				// 清理资源：尝试获取 session 中的技能组信息来清理空闲队列
				session, _ := sessionSvc.GetSession(ctx, tenantID, agentID)
				if session != nil && len(session.SkillGroups) > 0 {
					_ = sessionSvc.RemoveIdleBatch(ctx, agentID, session.SkillGroups)
				}

				// 删除技能组缓存
				_ = sessionSvc.DeleteAgentSkills(ctx, agentID)

				// 从在线集合移除
				_ = sessionSvc.RemoveOnline(ctx, tenantID, agentID)

				// 删除 session
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
