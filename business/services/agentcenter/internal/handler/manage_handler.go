package handler

import (
	"context"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// ManageHandler handles internal management requests from ACD.
type ManageHandler struct {
	agentSvc *service.AgentService
}

// NewManageHandler creates an internal management handler.
func NewManageHandler(agentSvc *service.AgentService) *ManageHandler {
	return &ManageHandler{
		agentSvc: agentSvc,
	}
}

// ---------- Request structures ----------

// TryAcquireRequest contains a request to allocate an agent.
type TryAcquireRequest struct {
	TenantID     string `json:"tenant_id"     binding:"required"`
	SkillGroupID string `json:"skill_group_id" binding:"required"`
}

// TryAcquireResponse contains the result of an agent allocation attempt.
type TryAcquireResponse struct {
	AgentID  string `json:"agent_id"`
	DnNumber string `json:"dn_number"`
	Success  bool   `json:"success"`
}

// RingRequest contains a ringing notification.
type RingRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	CallID   string `json:"call_id"   binding:"required"`
}

// ResumeReadyRequest contains a request to reset an agent to ready.
type ResumeReadyRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
}

// UpdateStatusRequest contains an agent status update.
type UpdateStatusRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	Status   int    `json:"status"    binding:"required"`
	Force    bool   `json:"force"`
	Reason   string `json:"reason"` // busy reason passed through when the target state is BUSY
}

// ---------- Endpoint implementations ----------

// TryAcquireAgent attempts to allocate an idle agent.
// @Summary Allocate an idle agent
// @Description Allocates the longest-idle agent from a skill group and uses a distributed lock to prevent oversell.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body TryAcquireRequest true "Allocation request"
// @Success 200 {object} response.Response{data=TryAcquireResponse}
// @Router /api/v1/internal/agent/try-acquire [post]
func (h *ManageHandler) TryAcquireAgent(ctx context.Context, c *app.RequestContext) {
	var req TryAcquireRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	agentID, err := h.agentSvc.TryAcquireAgent(ctx, req.TenantID, req.SkillGroupID)
	if err != nil {
		// No idle agent is a normal result, so return success with success=false.
		if bizErr, ok := errors.IsBizError(err); ok && bizErr.Code == errors.CodeAgentNoIdle {
			response.Success(c, TryAcquireResponse{
				Success: false,
			})
			return
		}
		response.Error(c, err)
		return
	}

	// Get the agent's extension number.
	session, _ := h.agentSvc.GetSessionService().GetSession(ctx, req.TenantID, agentID)
	dnNumber := ""
	if session != nil {
		dnNumber = session.DnNumber
	}

	response.Success(c, TryAcquireResponse{
		AgentID:  agentID,
		DnNumber: dnNumber,
		Success:  true,
	})
}

// Ring notifies an agent of an incoming call.
// @Summary Notify an agent to ring
// @Description ACD transitions the agent to the ringing state.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body RingRequest true "Ringing request"
// @Success 200 {object} response.Response
// @Router /api/v1/internal/agent/ring [post]
func (h *ManageHandler) Ring(ctx context.Context, c *app.RequestContext) {
	var req RingRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	if err := h.agentSvc.Ring(ctx, req.TenantID, req.AgentID, req.CallID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ResumeReady resets an agent to ready after an allocation failure.
// @Summary Reset an agent to ready
// @Description Resets the agent to the ready state after an allocation failure.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body ResumeReadyRequest true "Reset request"
// @Success 200 {object} response.Response
// @Router /api/v1/internal/agent/resume-ready [post]
func (h *ManageHandler) ResumeReady(ctx context.Context, c *app.RequestContext) {
	var req ResumeReadyRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	if err := h.agentSvc.ResumeReady(ctx, req.TenantID, req.AgentID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// CallLifecycleRequest contains a call answer or end notification.
type CallLifecycleRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	CallID   string `json:"call_id"  binding:"required"`
}

// CallAnswer notifies the agent center that an agent answered a call.
// @Summary Notify call answer
// @Description Transitions the agent from ringing to in-call after the agent answers.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body CallLifecycleRequest true "Call answer notification"
// @Success 200 {object} response.Response
// @Router /api/v1/internal/agent/call-answer [post]
func (h *ManageHandler) CallAnswer(ctx context.Context, c *app.RequestContext) {
	var req CallLifecycleRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	if err := h.agentSvc.InternalCallAnswer(ctx, req.TenantID, req.AgentID, req.CallID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// CallEnd notifies the agent center that a call ended.
// @Summary Notify call end
// @Description Releases the agent to the ready state and returns the agent to the idle pool after a call ends.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body CallLifecycleRequest true "Call end notification"
// @Success 200 {object} response.Response
// @Router /api/v1/internal/agent/call-end [post]
func (h *ManageHandler) CallEnd(ctx context.Context, c *app.RequestContext) {
	var req CallLifecycleRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	if err := h.agentSvc.InternalCallEnd(ctx, req.TenantID, req.AgentID, req.CallID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateStatus updates an agent's status.
// @Summary Update agent status
// @Description Updates agent status internally, with an optional forced mode that skips state machine validation.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body UpdateStatusRequest true "Status update request"
// @Success 200 {object} response.Response
// @Router /api/v1/internal/agent/update-status [post]
func (h *ManageHandler) UpdateStatus(ctx context.Context, c *app.RequestContext) {
	var req UpdateStatusRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	status := model.AgentStatus(req.Status)
	if !status.IsValid() {
		response.Fail(c, errors.ErrAgentStatusInvalid)
		return
	}

	if err := h.agentSvc.UpdateStatus(ctx, req.TenantID, req.AgentID, status, req.Force, req.Reason); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ---------- Internal agent session API for the push WebSocket gateway ----------

// SessionRequest contains a request to establish, renew, or delete an agent session.
type SessionRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	DnNumber string `json:"dn_number"` // required only when establishing a session
}

// SessionResponse contains the established session.
type SessionResponse struct {
	Status     int      `json:"status"`
	StatusName string   `json:"status_name"`
	DnNumber   string   `json:"dn_number"`
	Skills     []string `json:"skills"`
}

// EstablishSession checks in an agent when a WebSocket connects, with idempotent semantics.
// @Summary Establish or restore an agent session
// @Description A push WebSocket connection checks in the agent by renewing an existing session or validating the agent and extension before creating one.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body SessionRequest true "Session request"
// @Success 200 {object} response.Response{data=SessionResponse}
// @Router /api/v1/internal/agent/session [post]
func (h *ManageHandler) EstablishSession(ctx context.Context, c *app.RequestContext) {
	var req SessionRequest
	if err := c.BindAndValidate(&req); err != nil || req.DnNumber == "" {
		response.Fail(c, errors.ErrParamError)
		return
	}
	session, err := h.agentSvc.InternalEstablishSession(ctx, req.TenantID, req.AgentID, req.DnNumber)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, SessionResponse{
		Status:     session.Status,
		StatusName: session.GetStatus().String(),
		DnNumber:   session.DnNumber,
		Skills:     session.SkillGroups,
	})
}

// RefreshSession renews an agent session.
// @Summary Renew an agent session
// @Description Provides the session renewal endpoint called by the push keepalive goroutine.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body SessionRequest true "Session request"
// @Success 200 {object} response.Response
// @Router /api/v1/internal/agent/session/refresh [post]
func (h *ManageHandler) RefreshSession(ctx context.Context, c *app.RequestContext) {
	var req SessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}
	if err := h.agentSvc.InternalRefreshSession(ctx, req.TenantID, req.AgentID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// DeleteSession checks out an agent and clears the session idempotently.
// @Summary Delete an agent session
// @Description Checks out and clears the session when a WebSocket disconnects; deletion succeeds even if no session exists.
// @Tags InternalManagement
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "Internal service token"
// @Param request body SessionRequest true "Session request"
// @Success 200 {object} response.Response
// @Router /api/v1/internal/agent/session/delete [post]
func (h *ManageHandler) DeleteSession(ctx context.Context, c *app.RequestContext) {
	var req SessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}
	if err := h.agentSvc.InternalDeleteSession(ctx, req.TenantID, req.AgentID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}
