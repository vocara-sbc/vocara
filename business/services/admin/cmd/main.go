// Package main 管理后台服务入口
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
	// 1. 加载配置（可用 CONFIG_PATH 环境变量覆盖配置路径）
	configPath := "./configs/config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}
	cfg, err := config.Load(configPath)
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

	logger.Info("vocara admin service starting",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("build_version", version.Version),
		zap.String("git_commit", version.GitCommit),
		zap.String("build_time", version.BuildTime),
		zap.String("env", cfg.App.Env),
	)

	// License 授权：进程内验签，后台周期复校；异常不阻断启动，文件修复后于复校恢复
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

	// 3. 初始化 MySQL
	if err := database.Init(cfg.MySQL); err != nil {
		logger.Fatal("init mysql failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("mysql connected")

	// 自动迁移
	db := database.Get()
	if cfg.MySQL.AutoMigrate {
		// 主迁移：GORM AutoMigrate 在首个模型出错时即中断，故 form 三表拆出单独迁移，
		// 避免被其他模型（如 FSInstance）的历史错误阻断
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

		// 表单与客户/工单相关表独立迁移（必须成功，不被主迁移的历史错误阻断）
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

	// 4. 初始化 Redis
	if err := rdb.Init(cfg.Redis); err != nil {
		logger.Warn("init redis failed", zap.Error(err))
	} else {
		logger.Info("redis connected")
	}

	// 4.5 初始化 NATS 与事件发布者（可选，失败不阻塞启动；
	// 用于呼入接待变更事件 route.route_change 的发布，callctrl 订阅后即时刷新路由缓存）
	var eventPub eventbus.Publisher
	if natsConn := connectNATS(cfg); natsConn != nil {
		defer natsConn.Close()
		eventPub = eventbus.NewNATSPublisher(natsConn)
	}

	// 5. 初始化服务层
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

	// 安装标识以系统配置表为权威，由 admin 生成并经页面展示；License 经页面上传后写库并热加载
	licenseManager.SetInstallIDProvider(licenseSvc)
	if err := licenseSvc.EnsureInstallID(context.Background()); err != nil {
		logger.Error("ensure install id failed", zap.Error(err))
	}
	if err := licenseSvc.Reload(); err != nil {
		logger.Info("license not present in database, waiting for import", zap.Error(err))
	}
	categorySvc := service.NewFormCategoryService()
	formSvc := service.NewFormService()

	// 存量模板版本化迁移（幂等）：补 v1、回填版本指针
	if err := formSvc.EnsureTemplateVersions(context.Background()); err != nil {
		logger.Error("ensure template versions failed", zap.Error(err))
	} else {
		logger.Info("ensure template versions completed")
	}

	// 内置业务模板初始化（customer biz_type、ticket 模板，幂等）
	if err := formSvc.EnsureBuiltinTemplates(context.Background()); err != nil {
		logger.Error("ensure builtin templates failed", zap.Error(err))
	} else {
		logger.Info("ensure builtin templates completed")
	}

	// 6. 初始化 Handler 层
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
	// FS 动态目录（xml_curl directory binding）+ 座席"我的分机"（SIP 凭据下发）
	fsDirHandler := handler.NewFSDirectoryHandler(cfg.App.SecretKey, 1)
	// FS 动态拨号计划（xml_curl dialplan binding）：规则实时下发，外呼随 License 状态控制
	fsDialplanSvc := service.NewFSDialplanService(licenseManager, cfg.FSDialplan)
	fsDialplanHandler := handler.NewFSDialplanHandler(fsDialplanSvc, cfg.App.SecretKey)

	// 7. 初始化 HTTP 服务
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		server.WithReadTimeout(time.Duration(cfg.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.Server.WriteTimeout)*time.Second),
		server.WithIdleTimeout(time.Duration(cfg.Server.IdleTimeout)*time.Second),
	)

	// 中间件链: TraceID → Recovery → CORS → RequestLogger → JWT
	h.Use(
		middleware.TraceID(),
		middleware.Recovery(),
		middleware.CORS(),
		middleware.RequestLogger(),
	)

	// 健康检查
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.OK(c, utils.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// 认证相关（不需要鉴权）
	authGroup := h.Group("/api/v1/admin/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/logout", authHandler.Logout)
	}

	// 需要 JWT 鉴权的路由
	api := h.Group("/api/v1/admin")
	api.Use(middleware.JWT(cfg.App.SecretKey))
	{
		// 认证相关
		auth := api.Group("/auth")
		{
			auth.PUT("/profile", authHandler.UpdateProfile)
			auth.POST("/change_password", authHandler.ChangePassword)
			auth.GET("/userinfo", authHandler.GetUserInfo)
		}

		// 座席管理
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

		// 组织机构管理
		org := api.Group("/organizations")
		{
			org.GET("/list", orgHandler.List)
			org.GET("/:id", orgHandler.Get)
			org.POST("/save", orgHandler.Save)
			org.POST("/update", orgHandler.Update)
			org.DELETE("/:id", orgHandler.Delete)
		}

		// 技能组管理
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

		// 分机管理
		dn := api.Group("/dns")
		{
			// 当前登录座席的绑定分机（含 SIP 注册密码，电话条签入用）
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

		// 队列管理
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

		// 动态表单
		form := api.Group("/forms")
		{
			form.GET("/categories", formHandler.CategoryList)
			form.POST("/categories", formHandler.CategorySave)

			// 客户管理、工单管理（登录即可，所有角色可访问）
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
				// export 必须注册在 /:id 之前，避免被路径参数吞掉
				records.GET("/export", formHandler.RecordExport)
				records.GET("/:id", formHandler.RecordGet)
				records.POST("/save", formHandler.RecordSave)
				records.POST("/update", formHandler.RecordUpdate)
				records.DELETE("/:id", formHandler.RecordDelete)
			}
		}

		// 呼入接待管理
		route := api.Group("/inbound-routes")
		{
			route.GET("/page", inboundRouteHandler.Page)
			route.GET("/list", inboundRouteHandler.List)
			route.GET("/:id", inboundRouteHandler.Get)
			route.POST("/save", inboundRouteHandler.Save)
			route.POST("/update", inboundRouteHandler.Update)
			route.DELETE("/:id", inboundRouteHandler.Delete)
			// 节假日
			route.GET("/holiday/page", inboundRouteHandler.HolidayPage)
			route.GET("/holiday/list", inboundRouteHandler.HolidayList)
			route.GET("/holiday/:id", inboundRouteHandler.HolidayGet)
			route.POST("/holiday/save", inboundRouteHandler.HolidaySave)
			route.POST("/holiday/update", inboundRouteHandler.HolidayUpdate)
			route.DELETE("/holiday/:id", inboundRouteHandler.HolidayDelete)
		}

		// 通话记录
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

		// 线路资源管理
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

		// 统计报表
		stats := api.Group("/stats")
		{
			stats.GET("/agent", statsHandler.AgentStats)
			stats.GET("/queue", statsHandler.QueueStats)
			stats.GET("/line", statsHandler.LineStats)
			stats.GET("/trend", statsHandler.TrendStats)
			stats.GET("/realtime", statsHandler.RealtimeStats)
		}

		// 系统管理
		sys := api.Group("/system")
		{
			// 参数配置
			sys.GET("/param/list", systemHandler.ConfigList)
			sys.POST("/param/save", systemHandler.ConfigSave)
			sys.DELETE("/param/:id", systemHandler.ConfigDelete)

			// 操作日志
			sys.GET("/operation_log/page", systemHandler.OperationLogPage)

			// FS 实例
			sys.GET("/fs/page", systemHandler.FSInstancePage)
			sys.GET("/fs/list", systemHandler.FSInstanceList)
			sys.GET("/fs/:id", systemHandler.FSInstanceGet)
			sys.POST("/fs/save", systemHandler.FSInstanceSave)
			sys.POST("/fs/update", systemHandler.FSInstanceUpdate)
			sys.DELETE("/fs/:id", systemHandler.FSInstanceDelete)

			// 租户管理
			sys.GET("/tenant/page", systemHandler.TenantPage)
			sys.GET("/tenant/list", systemHandler.TenantList)
			sys.GET("/tenant/:id", systemHandler.TenantGet)
			sys.POST("/tenant/save", systemHandler.TenantSave)
			sys.POST("/tenant/update", systemHandler.TenantUpdate)
			sys.DELETE("/tenant/:id", systemHandler.TenantDelete)
		}

		// 授权许可
		licenseGroup := api.Group("/license")
		{
			licenseGroup.GET("/info", licenseHandler.Info)
			licenseGroup.POST("/import", licenseHandler.Import)
			licenseGroup.POST("/reload", licenseHandler.Reload)
		}
	}

	// 内部服务间 API（X-Internal-Token 鉴权，不挂 JWT；与 agentcenter internal 模式一致）
	internalGroup := h.Group("/api/v1/internal", middleware.InternalAuth(cfg.App.SecretKey))
	{
		// 呼入路由：callctrl 定时拉取 DID→队列/技能组映射
		internalGroup.GET("/inbound-routes", inboundRouteHandler.InboundRoutes)
	}

	// FS xml_curl 动态目录回调（不挂 JWT：FS 服务器访问，token 经 query 参数校验）
	h.POST("/api/v1/fs/directory", fsDirHandler.Directory)
	// FS xml_curl 动态拨号计划回调（不挂 JWT：FS 服务器访问，token 经 query 参数校验）
	h.POST("/api/v1/fs/dialplan", fsDialplanHandler.Dialplan)

	// 启动 HTTP 服务（后台）
	go func() {
		logger.Info("http server starting",
			zap.String("addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		)
		h.Spin()
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("admin service shutting down...")

	// 优雅关闭 HTTP 服务
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer shutdownCancel()

	if err := h.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", zap.Error(err))
	}

	logger.Info("admin service stopped")
}

// connectNATS 初始化 NATS 连接（可选，失败返回 nil 不阻塞服务启动）。
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
