// Package handler provides HTTP request handling.
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

// AcdHandler handles ACD HTTP requests.
type AcdHandler struct {
	acdService service.AcdService
}

// NewAcdHandler creates an ACD handler.
func NewAcdHandler(acdService service.AcdService) *AcdHandler {
	return &AcdHandler{
		acdService: acdService,
	}
}

// --- Request and response structures ---

// EnqueueReq contains an enqueue request.
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

// EnqueueResp contains an enqueue response.
type EnqueueResp struct {
	CallID   string         `json:"callId"`
	QueueID  int64          `json:"queueId"`
	Priority model.Priority `json:"priority"`
	Status   string         `json:"status"`
	Position int64          `json:"position,omitempty"` // Queue position.
}

// DequeueReq contains a dequeue request.
type DequeueReq struct {
	CallID string `json:"callId"`
	Reason string `json:"reason"` // answered | abandoned | transferred | timeout | error
}

// PauseReq contains a pause request.
type PauseReq struct {
	CallID string `json:"callId"`
	Reason string `json:"reason,omitempty"`
}

// ResumeReq contains a resume request.
type ResumeReq struct {
	CallID string `json:"callId"`
}

// --- Handler functions ---

// Enqueue adds a call to a queue.
// @Summary Enqueue a call
// @Description Add a call to the specified queue to await allocation
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body EnqueueReq true "Enqueue request"
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

// Dequeue removes a call from a queue.
// @Summary Dequeue a call
// @Description Remove a call from the queue after hang up, transfer, successful allocation, or another outcome
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body DequeueReq true "Dequeue request"
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

// Pause pauses call allocation.
// @Summary Pause call allocation
// @Description Pause allocation for a call, such as during IVR interaction
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body PauseReq true "Pause request"
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

// Resume resumes call allocation.
// @Summary Resume call allocation
// @Description Resume allocation for a paused call
// @Tags ACD
// @Accept json
// @Produce json
// @Param request body ResumeReq true "Resume request"
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

// GetCallInfo returns call information.
// @Summary Get call information
// @Description Get queueing information for a call by call ID
// @Tags ACD
// @Produce json
// @Param callId path string true "Call ID"
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
