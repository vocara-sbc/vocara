package handler

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"

	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/pkg/response"
	"vocara/services/push/internal/model"
	"vocara/services/push/internal/service"
)

// PushHandler 推送接口处理器
// 提供 HTTP 接口供其他服务调用推送
type PushHandler struct {
	pushService   *service.PushService
	internalToken string // 内部服务鉴权 token
}

// NewPushHandler 创建推送接口处理器
func NewPushHandler(pushService *service.PushService, internalToken string) *PushHandler {
	return &PushHandler{
		pushService:   pushService,
		internalToken: internalToken,
	}
}

// ===== 推送接口 =====

// PushToAgent 推送给指定座席
// POST /api/v1/push/to-agent
//
// 请求体：
//
//	{
//	  "tenant_id": 1,
//	  "agent_id": 1001,
//	  "event": "agent.status_change",
//	  "data": { ... }
//	}
func (h *PushHandler) PushToAgent(c context.Context, ctx *app.RequestContext) {
	var req model.PushToAgentRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		logger.Warn("push to agent: invalid request", zap.Error(err))
		response.FailParamError(ctx, err.Error())
		return
	}

	count, err := h.pushService.PushToAgent(req.TenantID, req.AgentID, req.Event, req.Data)
	if err != nil {
		logger.Error("push to agent failed",
			zap.Int64("agent_id", req.AgentID),
			zap.String("event", req.Event),
			zap.Error(err),
		)
		response.Fail(ctx, errors.ErrServiceError)
		return
	}

	response.OK(ctx, &model.PushResponse{
		Success: true,
		Count:   count,
		Message: "ok",
	})
}

// PushToTenant 推送给整个租户
// POST /api/v1/push/to-tenant
//
// 请求体：
//
//	{
//	  "tenant_id": 1,
//	  "event": "system.notice",
//	  "data": { ... },
//	  "user_types": ["agent", "admin"]
//	}
func (h *PushHandler) PushToTenant(c context.Context, ctx *app.RequestContext) {
	var req model.PushToTenantRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		logger.Warn("push to tenant: invalid request", zap.Error(err))
		response.FailParamError(ctx, err.Error())
		return
	}

	count, err := h.pushService.PushToTenant(req.TenantID, req.Event, req.Data, req.UserTypes)
	if err != nil {
		logger.Error("push to tenant failed",
			zap.Int64("tenant_id", req.TenantID),
			zap.String("event", req.Event),
			zap.Error(err),
		)
		response.Fail(ctx, errors.ErrServiceError)
		return
	}

	response.OK(ctx, &model.PushResponse{
		Success: true,
		Count:   count,
		Message: "ok",
	})
}

// PushToAll 广播给所有人
// POST /api/v1/push/to-all
//
// 请求体：
//
//	{
//	  "event": "system.notice",
//	  "data": { ... }
//	}
func (h *PushHandler) PushToAll(c context.Context, ctx *app.RequestContext) {
	var req model.PushToAllRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		logger.Warn("push to all: invalid request", zap.Error(err))
		response.FailParamError(ctx, err.Error())
		return
	}

	count, err := h.pushService.PushToAll(req.Event, req.Data)
	if err != nil {
		logger.Error("push to all failed",
			zap.String("event", req.Event),
			zap.Error(err),
		)
		response.Fail(ctx, errors.ErrServiceError)
		return
	}

	response.OK(ctx, &model.PushResponse{
		Success: true,
		Count:   count,
		Message: "ok",
	})
}

// PushToTopic 推送到指定主题
// POST /api/v1/push/topic
//
// 请求体：
//
//	{
//	  "topic": "queue:1001",
//	  "event": "queue.update",
//	  "data": { ... }
//	}
func (h *PushHandler) PushToTopic(c context.Context, ctx *app.RequestContext) {
	var req model.PushToTopicRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		logger.Warn("push to topic: invalid request", zap.Error(err))
		response.FailParamError(ctx, err.Error())
		return
	}

	count, err := h.pushService.PushToTopic(req.Topic, req.Event, req.Data)
	if err != nil {
		logger.Error("push to topic failed",
			zap.String("topic", req.Topic),
			zap.String("event", req.Event),
			zap.Error(err),
		)
		response.Fail(ctx, errors.ErrServiceError)
		return
	}

	response.OK(ctx, &model.PushResponse{
		Success: true,
		Count:   count,
		Message: "ok",
	})
}

// PushBatch 批量推送
// POST /api/v1/push/batch
//
// 请求体：
//
//	{
//	  "items": [
//	    {
//	      "target_type": "agent",
//	      "target_id": "1001",
//	      "tenant_id": 1,
//	      "event": "agent.status_change",
//	      "data": { ... }
//	    },
//	    ...
//	  ]
//	}
func (h *PushHandler) PushBatch(c context.Context, ctx *app.RequestContext) {
	var req model.PushBatchRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		logger.Warn("push batch: invalid request", zap.Error(err))
		response.FailParamError(ctx, err.Error())
		return
	}

	if len(req.Items) == 0 {
		response.FailParamError(ctx, "推送项不能为空")
		return
	}

	// 限制批量推送数量
	if len(req.Items) > 100 {
		response.FailParamError(ctx, "批量推送不能超过100条")
		return
	}

	totalCount := 0
	failedCount := 0

	for _, item := range req.Items {
		var count int
		var err error

		switch item.TargetType {
		case "agent":
			count, err = h.pushService.PushToAgent(item.TenantID, parseInt64(item.TargetID), item.Event, item.Data)
		case "tenant":
			count, err = h.pushService.PushToTenant(parseInt64(item.TargetID), item.Event, item.Data, nil)
		case "topic":
			count, err = h.pushService.PushToTopic(item.TargetID, item.Event, item.Data)
		case "all":
			count, err = h.pushService.PushToAll(item.Event, item.Data)
		default:
			failedCount++
			continue
		}

		if err != nil {
			failedCount++
			logger.Warn("batch push item failed",
				zap.String("target_type", item.TargetType),
				zap.String("target_id", item.TargetID),
				zap.Error(err),
			)
			continue
		}

		totalCount += count
	}

	response.OK(ctx, map[string]interface{}{
		"success":       true,
		"total_items":   len(req.Items),
		"total_count":   totalCount,
		"failed_count":  failedCount,
		"message":       "batch push completed",
	})
}

// ===== 内部方法 =====

// parseInt64 字符串转 int64
func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}

// 确保 json 包被使用
var _ = json.Marshal
