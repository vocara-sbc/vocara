// Package main provides the admin service entry point.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"vocara/admin/internal/handler"
	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/config"
	"vocara/pkg/database"
	"vocara/pkg/eventbus"
	"vocara/pkg/license"
	"vocara/pkg/logger"
	"vocara/pkg/middleware"
	rdb "vocara/pkg/redis"
	"vocara/pkg/response"
	"vocara/pkg/version"
)

func main() {
	// 1. Load configuration; CONFIG_PATH overrides the configuration path.
	configPath := "./configs/config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("load config failed: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logging.
	if err := logger.Init(cfg.Log); err != nil {
		fmt.Printf("init logger failed: %v\n", err)
		os.Exit(1)
	}
	// Discard Hertz framework logs written directly to stderr to avoid a second log file.
	hlog.SetOutput(io.Discard)
	defer logger.Sync()

	logger.Info("vocara admin service starting",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("build_version", version.Version),
		zap.String("git_commit", version.GitCommit),
		zap.String("build_time", version.BuildTime),
		zap.String("env", cfg.App.Env),
	)

	// License verification runs in-process with periodic background checks and does not block startup on errors.
	licenseCtx, licenseCancel := context.WithCancel(context.Background())
	defer licenseCancel()
	licenseManager := license.Bootstrap(licenseCtx, license.Config{
		File:          cfg.License.File,
		CheckInterval: cfg.License.CheckInterval,
		FailOpen:      cfg.License.FailOpen,
		InstallID:     cfg.License.InstallID,
	})
	lr := licenseManager.Result()
	logger.Info("license status",
		zap.String("state", string(lr.State)),
		zap.String("message", lr.Message),
	)

	// 3. Initialize MySQL.
	if err := database.Init(cfg.MySQL); err != nil {
		logger.Fatal("init mysql failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("mysql connected")

	// Automatically migrate database models.
	db := database.Get()
	if cfg.MySQL.AutoMigrate {
		// GORM AutoMigrate stops at the first model error, so form tables migrate independently
		// to remain unaffected by errors in other models such as FSInstance.
		if err := db.AutoMigrate(
			&model.Tenant{},
			&model.Agent{},
			&model.Organization{},
			&model.SkillGroup{},
			&model.AgentSkill{},
			&model.DN{},
			&model.Queue{},
			&model.InboundRoute{},
			&model.HolidaySchedule{},
			&model.CallRecord{},
			&model.Line{},
			&model.FSInstance{},
			&model.SystemConfig{},
			&model.OperationLog{},
		); err != nil {
			logger.Error("auto migrate failed", zap.Error(err))
		} else {
			logger.Info("auto migrate completed")
		}

		// Form, customer, and ticket tables migrate independently of errors in the main model group.
		if err := db.AutoMigrate(
			&model.FormTemplate{},
			&model.FormTemplateVersion{},
			&model.FormRecord{},
			&model.Customer{},
		); err != nil {
			logger.Error("form auto migrate failed", zap.Error(err))
		} else {
			logger.Info("form auto migrate completed")
		}
	}

	// 4. Initialize Redis.
	if err := rdb.Init(cfg.Redis); err != nil {
		logger.Warn("init redis failed", zap.Error(err))
	} else {
		logger.Info("redis connected")
	}

	// 4.5 Initialize the optional NATS event publisher without blocking startup on failure.
	// It publishes route.route_change events so callctrl immediately refreshes its inbound route cache.
	var eventPub eventbus.Publisher
	if natsConn := connectNATS(cfg); natsConn != nil {
		defer natsConn.Close()
		eventPub = eventbus.NewNATSPublisher(natsConn)
	}

	// 5. Initialize services.
	agentSvc := service.NewAgentService()
	orgSvc := service.NewOrganizationService()
	skillSvc := service.NewSkillGroupService()
	dnSvc := service.NewDNService()
	queueSvc := service.NewQueueService()
	inboundRouteSvc := service.NewInboundRouteService()
	inboundRouteSvc.SetEventPublisher(eventPub)
	recordSvc := service.NewRecordService()
	lineSvc := service.NewLineService(licenseManager)
	statsSvc := service.NewStatsService()
	systemSvc := service.NewSystemService()
	tenantSvc := service.NewTenantService(licenseManager)
	licenseSvc := service.NewLicenseService(licenseManager, repository.NewSystemRepository())

	// The system configuration table is authoritative for the admin-generated installation ID; uploaded licenses are saved and hot-loaded.
	licenseManager.SetInstallIDProvider(licenseSvc)
	if err := licenseSvc.EnsureInstallID(context.Background()); err != nil {
		logger.Error("ensure install id failed", zap.Error(err))
	}
	if err := licenseSvc.Reload(); err != nil {
		logger.Info("license not present in database, waiting for import", zap.Error(err))
	}
	categorySvc := service.NewFormCategoryService()
	formSvc := service.NewFormService()

	// Populate template v1 data and version pointers idempotently.
	if err := formSvc.EnsureTemplateVersions(context.Background()); err != nil {
		logger.Error("ensure template versions failed", zap.Error(err))
	} else {
		logger.Info("ensure template versions completed")
	}

	// Initialize built-in customer and ticket business templates idempotently.
	if err := formSvc.EnsureBuiltinTemplates(context.Background()); err != nil {
		logger.Error("ensure builtin templates failed", zap.Error(err))
	} else {
		logger.Info("ensure builtin templates completed")
	}

	// 6. Initialize handlers.
	authHandler := handler.NewAuthHandler(cfg, agentSvc, tenantSvc)
	agentHandler := handler.NewAgentHandler(agentSvc, licenseManager)
	orgHandler := handler.NewOrganizationHandler(orgSvc)
	skillHandler := handler.NewSkillGroupHandler(skillSvc)
	dnHandler := handler.NewDNHandler(dnSvc)
	queueHandler := handler.NewQueueHandler(queueSvc)
	inboundRouteHandler := handler.NewInboundRouteHandler(inboundRouteSvc)
	recordHandler := handler.NewRecordHandler(recordSvc, licenseManager)
	lineHandler := handler.NewLineHandler(lineSvc)
	statsHandler := handler.NewStatsHandler(statsSvc)
	systemHandler := handler.NewSystemHandler(systemSvc, tenantSvc)
	licenseHandler := handler.NewLicenseHandler(licenseSvc)
	formHandler := handler.NewFormHandler(formSvc, categorySvc, licenseManager)
	customerHandler := handler.NewCustomerHandler(formSvc)
	ticketHandler := handler.NewTicketHandler(formSvc)
	// Serve the FS xml_curl directory and the agent's own extension SIP credentials.
	fsDirHandler := handler.NewFSDirectoryHandler(cfg.App.SecretKey, 1)
	// Serve FS xml_curl dialplan rules in real time and control outbound calling by license status.
	fsDialplanSvc := service.NewFSDialplanService(licenseManager, cfg.FSDialplan)
	fsDialplanHandler := handler.NewFSDialplanHandler(fsDialplanSvc, cfg.App.SecretKey)

	// 7. Initialize the HTTP server.
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		server.WithReadTimeout(time.Duration(cfg.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.Server.WriteTimeout)*time.Second),
		server.WithIdleTimeout(time.Duration(cfg.Server.IdleTimeout)*time.Second),
	)

	// Middleware chain: TraceID → Recovery → CORS → RequestLogger → JWT
	h.Use(
		middleware.TraceID(),
		middleware.Recovery(),
		middleware.CORS(),
		middleware.RequestLogger(),
	)

	// Health check.
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.OK(c, utils.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// Authentication routes that do not require authentication.
	authGroup := h.Group("/api/v1/admin/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/logout", authHandler.Logout)
	}

	// Routes requiring JWT authentication.
	api := h.Group("/api/v1/admin")
	api.Use(middleware.JWT(cfg.App.SecretKey))
	{
		// Authentication.
		auth := api.Group("/auth")
		{
			auth.PUT("/profile", authHandler.UpdateProfile)
			auth.POST("/change_password", authHandler.ChangePassword)
			auth.GET("/userinfo", authHandler.GetUserInfo)
		}

		// Agent management.
		agent := api.Group("/agents")
		{
			agent.GET("/page", agentHandler.Page)
			agent.GET("/list", agentHandler.List)
			agent.GET("/:id", agentHandler.Get)
			agent.POST("/save", agentHandler.Save)
			agent.POST("/update", agentHandler.Update)
			agent.DELETE("/:id", agentHandler.Delete)
			agent.POST("/import", agentHandler.Import)
			agent.GET("/export", agentHandler.Export)
			agent.POST("/:id/skill", agentHandler.UpdateSkill)
			agent.POST("/:id/bind_dn", agentHandler.BindDn)
			agent.POST("/:id/reset_password", agentHandler.ResetPassword)
			agent.POST("/:id/enable", agentHandler.Enable)
			agent.POST("/:id/disable", agentHandler.Disable)
			agent.GET("/status/count", agentHandler.GetStatusCount)
		}

		// Organization management.
		org := api.Group("/organizations")
		{
			org.GET("/list", orgHandler.List)
			org.GET("/:id", orgHandler.Get)
			org.POST("/save", orgHandler.Save)
			org.POST("/update", orgHandler.Update)
			org.DELETE("/:id", orgHandler.Delete)
		}

		// Skill group management.
		skillGroup := api.Group("/skill_groups")
		{
			skillGroup.GET("/page", skillHandler.Page)
			skillGroup.GET("/list", skillHandler.List)
			skillGroup.GET("/:id", skillHandler.Get)
			skillGroup.POST("/save", skillHandler.Save)
			skillGroup.POST("/update", skillHandler.Update)
			skillGroup.DELETE("/:id", skillHandler.Delete)
			skillGroup.GET("/:id/agents", skillHandler.Agents)
			skillGroup.POST("/:id/agents", skillHandler.SetAgents)
		}

		// Extension management.
		dn := api.Group("/dns")
		{
			// The current agent's bound extension and SIP registration password for phone bar sign-in.
			dn.GET("/my", fsDirHandler.MyDn)
			dn.GET("/page", dnHandler.Page)
			dn.GET("/list", dnHandler.List)
			dn.GET("/:id", dnHandler.Get)
			dn.POST("/save", dnHandler.Save)
			dn.POST("/update", dnHandler.Update)
			dn.DELETE("/:id", dnHandler.Delete)
			dn.POST("/status_check", dnHandler.StatusCheck)
			dn.POST("/batch_import", dnHandler.BatchImport)
			dn.GET("/available", dnHandler.Available)
			dn.POST("/:id/bind_agent", dnHandler.BindDnAgent)
			dn.POST("/:id/unbind_agent", dnHandler.UnbindAgent)
		}

		// Queue management.
		queue := api.Group("/queues")
		{
			queue.GET("/page", queueHandler.Page)
			queue.GET("/list", queueHandler.List)
			queue.GET("/:id", queueHandler.Get)
			queue.POST("/save", queueHandler.Save)
			queue.POST("/update", queueHandler.Update)
			queue.DELETE("/:id", queueHandler.Delete)
			queue.GET("/monitor/list", queueHandler.Monitor)
			queue.GET("/:id/agents", queueHandler.Agents)
		}

		// Dynamic forms.
		form := api.Group("/forms")
		{
			form.GET("/categories", formHandler.CategoryList)
			form.POST("/categories", formHandler.CategorySave)

			// Customer and ticket management for all authenticated roles.
			form.GET("/customers", customerHandler.List)
			form.GET("/customers/:id", customerHandler.Get)
			form.GET("/tickets", ticketHandler.List)
			form.GET("/tickets/:id", ticketHandler.Get)
			form.POST("/tickets/:id/status", ticketHandler.UpdateStatus)

			templates := form.Group("/templates")
			{
				templates.GET("/page", formHandler.TemplatePage)
				templates.GET("/list", formHandler.TemplateList)
				templates.GET("/:id/versions", formHandler.TemplateVersionList)
				templates.GET("/version/:versionId", formHandler.TemplateVersionGet)
				templates.GET("/key/:key", formHandler.TemplateSchemaByKey)
				templates.GET("/:id", formHandler.TemplateGet)
				templates.POST("/save", formHandler.TemplateSave)
				templates.POST("/update", formHandler.TemplateUpdate)
				templates.POST("/:id/status", formHandler.TemplateUpdateStatus)
				templates.DELETE("/:id", formHandler.TemplateDelete)
			}
			records := form.Group("/records")
			{
				records.GET("/page", formHandler.RecordPage)
				// Register export before /:id so the path parameter does not capture it.
				records.GET("/export", formHandler.RecordExport)
				records.GET("/:id", formHandler.RecordGet)
				records.POST("/save", formHandler.RecordSave)
				records.POST("/update", formHandler.RecordUpdate)
				records.DELETE("/:id", formHandler.RecordDelete)
			}
		}

		// Inbound reception management.
		route := api.Group("/inbound-routes")
		{
			route.GET("/page", inboundRouteHandler.Page)
			route.GET("/list", inboundRouteHandler.List)
			route.GET("/:id", inboundRouteHandler.Get)
			route.POST("/save", inboundRouteHandler.Save)
			route.POST("/update", inboundRouteHandler.Update)
			route.DELETE("/:id", inboundRouteHandler.Delete)
			// Holidays.
			route.GET("/holiday/page", inboundRouteHandler.HolidayPage)
			route.GET("/holiday/list", inboundRouteHandler.HolidayList)
			route.GET("/holiday/:id", inboundRouteHandler.HolidayGet)
			route.POST("/holiday/save", inboundRouteHandler.HolidaySave)
			route.POST("/holiday/update", inboundRouteHandler.HolidayUpdate)
			route.DELETE("/holiday/:id", inboundRouteHandler.HolidayDelete)
		}

		// Call records.
		record := api.Group("/records")
		{
			record.GET("/inbound/page", recordHandler.InboundPage)
			record.GET("/outbound/page", recordHandler.OutboundPage)
			record.GET("/:id", recordHandler.Detail)
			record.GET("/export", recordHandler.Export)
			record.GET("/today", recordHandler.Today)
			record.GET("/uncalled", recordHandler.Uncalled)
			record.GET("/:id/recording", recordHandler.Recording)
		}

		// Line resource management.
		line := api.Group("/lines")
		{
			line.GET("/page", lineHandler.Page)
			line.GET("/list", lineHandler.List)
			line.GET("/:id", lineHandler.Get)
			line.POST("/save", lineHandler.Save)
			line.POST("/update", lineHandler.Update)
			line.DELETE("/:id", lineHandler.Delete)
			line.GET("/status/info", lineHandler.Status)
		}

		// Statistics reports.
		stats := api.Group("/stats")
		{
			stats.GET("/agent", statsHandler.AgentStats)
			stats.GET("/queue", statsHandler.QueueStats)
			stats.GET("/line", statsHandler.LineStats)
			stats.GET("/trend", statsHandler.TrendStats)
			stats.GET("/realtime", statsHandler.RealtimeStats)
		}

		// System management.
		sys := api.Group("/system")
		{
			// Parameter configuration.
			sys.GET("/param/list", systemHandler.ConfigList)
			sys.POST("/param/save", systemHandler.ConfigSave)
			sys.DELETE("/param/:id", systemHandler.ConfigDelete)

			// Operation logs.
			sys.GET("/operation_log/page", systemHandler.OperationLogPage)

			// FS instances.
			sys.GET("/fs/page", systemHandler.FSInstancePage)
			sys.GET("/fs/list", systemHandler.FSInstanceList)
			sys.GET("/fs/:id", systemHandler.FSInstanceGet)
			sys.POST("/fs/save", systemHandler.FSInstanceSave)
			sys.POST("/fs/update", systemHandler.FSInstanceUpdate)
			sys.DELETE("/fs/:id", systemHandler.FSInstanceDelete)

			// Tenant management.
			sys.GET("/tenant/page", systemHandler.TenantPage)
			sys.GET("/tenant/list", systemHandler.TenantList)
			sys.GET("/tenant/:id", systemHandler.TenantGet)
			sys.POST("/tenant/save", systemHandler.TenantSave)
			sys.POST("/tenant/update", systemHandler.TenantUpdate)
			sys.DELETE("/tenant/:id", systemHandler.TenantDelete)
		}

		// License management.
		licenseGroup := api.Group("/license")
		{
			licenseGroup.GET("/info", licenseHandler.Info)
			licenseGroup.POST("/import", licenseHandler.Import)
			licenseGroup.POST("/reload", licenseHandler.Reload)
		}
	}

	// Internal service APIs authenticate with X-Internal-Token instead of JWT.
	internalGroup := h.Group("/api/v1/internal", middleware.InternalAuth(cfg.App.SecretKey))
	{
		// Inbound routes for callctrl to retrieve DID-to-queue and skill group mappings periodically.
		internalGroup.GET("/inbound-routes", inboundRouteHandler.InboundRoutes)
	}

	// The FS xml_curl directory callback uses a query token instead of JWT.
	h.POST("/api/v1/fs/directory", fsDirHandler.Directory)
	// The FS xml_curl dialplan callback uses a query token instead of JWT.
	h.POST("/api/v1/fs/dialplan", fsDialplanHandler.Dialplan)

	// Start the HTTP server in the background.
	go func() {
		logger.Info("http server starting",
			zap.String("addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		)
		h.Spin()
	}()

	// Wait for an exit signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("admin service shutting down...")

	// Shut down the HTTP server gracefully.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer shutdownCancel()

	if err := h.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", zap.Error(err))
	}

	logger.Info("admin service stopped")
}

// connectNATS initializes an optional NATS connection and returns nil on failure without blocking startup.
func connectNATS(cfg *config.Config) *nats.Conn {
	natsCfg := cfg.NATS
	if natsCfg.URLs == "" {
		return nil
	}

	opts := []nats.Option{
		nats.Name(natsCfg.ClientID),
		nats.Timeout(time.Duration(natsCfg.Timeout) * time.Second),
		nats.FlusherTimeout(time.Duration(natsCfg.FlushTimeout) * time.Second),
		nats.ReconnectWait(time.Duration(natsCfg.ReconnectWait) * time.Second),
		nats.MaxReconnects(natsCfg.MaxReconnect),
		nats.ReconnectBufSize(natsCfg.ReconnectBuf),
	}
	if natsCfg.User != "" {
		opts = append(opts, nats.UserInfo(natsCfg.User, natsCfg.Password))
	}
	if natsCfg.Token != "" {
		opts = append(opts, nats.Token(natsCfg.Token))
	}

	urls := natsCfg.URLList()
	conn, err := nats.Connect(strings.Join(urls, ","), opts...)
	if err != nil {
		logger.Warn("init nats client failed, event publishing disabled",
			zap.Strings("urls", urls),
			zap.Error(err))
		return nil
	}
	logger.Info("nats connected", zap.Strings("urls", urls))
	return conn
}
