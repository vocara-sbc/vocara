// Package handler 提供 HTTP 请求处理
package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"

	"vocara/pkg/logger"
	"vocara/pkg/response"
	"vocara/services/acd/internal/model"
	"vocara/services/acd/internal/service"
)

// AcdHandler ACD HTTP 处理器
type AcdHandler struct {
	acdService service.AcdService
}

// NewAcdHandler 创建 ACD 处理器
func NewAcdHandler(acdService service.AcdService) *AcdHandler {
	return &AcdHandler{
		acdService: acdService,
	}
}

// --- 请求/响应结构体 ---

// EnqueueReq 入队请求
type EnqueueReq struct {
	CallID       string         `json:"callId"`
	Caller       string         `json:"caller"`
	Called       string         `json:"called"`
	QueueID      int64          `json:"queueId"`
	SkillGroupID int64          `json:"skillGroupId"`
	Priority     model.Priority `json:"priority"`
	TenantID     int64          `json:"tenantId"`
	CustomerInfo string         `json:"customerInfo,omitempty"`
	UUI          string         `json:"uui,omitempty"`
}

// EnqueueResp 入队响应
type EnqueueResp struct {
	CallID   string         `json:"callId"`
	QueueID  int64          `json:"queueId"`
	Priority model.Priority `json:"priority"`
	Status   string         `json:"status"`
	Position int64          `json:"position,omitempty"` // 排队位置
}

// DequeueReq 出队请求
type DequeueReq struct {
	CallID string `json:"callId"`
	Reason string `json:"reason"` // answered | abandoned | transferred | timeout | error
}

// PauseReq 暂停请求
type PauseReq struct {
	CallID string `json:"callId"`
	Reason string `json:"reason,omitempty"`
}

// ResumeReq 恢复请求
type ResumeReq struct {
	CallID string `json:"callId"`
}

// --- 处理函数 ---

// Enqueue 通话入队
// @Summary 通话入队
// @Description 将通话加入指定队列等待分配
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body EnqueueReq true "入队请求"
// @Success 200 {object} response.Response{data=EnqueueResp}
// @Router /api/v1/acd/enqueue [post]
func (h *AcdHandler) Enqueue(ctx context.Context, c *app.RequestContext) {
	var req EnqueueReq
	if err := c.BindAndValidate(&req); err != nil {
		logger.WarnCtx(ctx, "invalid enqueue request", zap.Error(err))
		response.FailParamError(c, "invalid request: "+err.Error())
		return
	}

	call, err := h.acdService.Enqueue(ctx, &service.EnqueueRequest{
		CallID:       req.CallID,
		Caller:       req.Caller,
		Called:       req.Called,
		QueueID:      req.QueueID,
		SkillGroupID: req.SkillGroupID,
		Priority:     req.Priority,
		TenantID:     req.TenantID,
		CustomerInfo: req.CustomerInfo,
		UUI:          req.UUI,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "enqueue failed",
			zap.String("callId", req.CallID),
			zap.Error(err),
		)
		response.Error(c, err)
		return
	}

	response.Success(c, EnqueueResp{
		CallID:   call.CallID,
		QueueID:  call.QueueID,
		Priority: call.Priority,
		Status:   string(call.Status),
	})
}

// Dequeue 通话出队
// @Summary 通话出队
// @Description 将通话从队列中移除（挂机、转接、分配成功等）
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body DequeueReq true "出队请求"
// @Success 200 {object} response.Response
// @Router /api/v1/acd/dequeue [post]
func (h *AcdHandler) Dequeue(ctx context.Context, c *app.RequestContext) {
	var req DequeueReq
	if err := c.BindAndValidate(&req); err != nil {
		logger.WarnCtx(ctx, "invalid dequeue request", zap.Error(err))
		response.FailParamError(c, "invalid request: "+err.Error())
		return
	}

	if req.Reason == "" {
		response.FailParamError(c, "reason is required")
		return
	}

	err := h.acdService.Dequeue(ctx, req.CallID, req.Reason)
	if err != nil {
		logger.ErrorCtx(ctx, "dequeue failed",
			zap.String("callId", req.CallID),
			zap.Error(err),
		)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Pause 暂停分配
// @Summary 暂停通话分配
// @Description 暂停通话的分配（如进入 IVR 交互）
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body PauseReq true "暂停请求"
// @Success 200 {object} response.Response
// @Router /api/v1/acd/pause [post]
func (h *AcdHandler) Pause(ctx context.Context, c *app.RequestContext) {
	var req PauseReq
	if err := c.BindAndValidate(&req); err != nil {
		logger.WarnCtx(ctx, "invalid pause request", zap.Error(err))
		response.FailParamError(c, "invalid request: "+err.Error())
		return
	}

	err := h.acdService.Pause(ctx, req.CallID, req.Reason)
	if err != nil {
		logger.ErrorCtx(ctx, "pause failed",
			zap.String("callId", req.CallID),
			zap.Error(err),
		)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Resume 恢复分配
// @Summary 恢复通话分配
// @Description 恢复暂停状态的通话分配
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body ResumeReq true "恢复请求"
// @Success 200 {object} response.Response
// @Router /api/v1/acd/resume [post]
func (h *AcdHandler) Resume(ctx context.Context, c *app.RequestContext) {
	var req ResumeReq
	if err := c.BindAndValidate(&req); err != nil {
		logger.WarnCtx(ctx, "invalid resume request", zap.Error(err))
		response.FailParamError(c, "invalid request: "+err.Error())
		return
	}

	err := h.acdService.Resume(ctx, req.CallID)
	if err != nil {
		logger.ErrorCtx(ctx, "resume failed",
			zap.String("callId", req.CallID),
			zap.Error(err),
		)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// GetCallInfo 查询通话信息
// @Summary 查询通话信息
// @Description 根据通话 ID 查询通话排队信息
// @Tags ACD
// @Produce json
// @Param callId path string true "通话 ID"
// @Success 200 {object} response.Response{data=model.CallInfo}
// @Router /api/v1/acd/call/{callId} [get]
func (h *AcdHandler) GetCallInfo(ctx context.Context, c *app.RequestContext) {
	callID := c.Param("callId")
	if callID == "" {
		response.FailParamError(c, "callId is required")
		return
	}

	call, err := h.acdService.GetCallInfo(ctx, callID)
	if err != nil {
		logger.ErrorCtx(ctx, "get call info failed",
			zap.String("callId", callID),
			zap.Error(err),
		)
		response.Error(c, err)
		return
	}

	response.Success(c, call)
}
