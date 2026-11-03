package handler

import (
	"context"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// AgentHandler handles agent HTTP requests.
type AgentHandler struct {
	agentSvc *service.AgentService
}

// NewAgentHandler creates an agent handler.
func NewAgentHandler(agentSvc *service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentSvc: agentSvc,
	}
}

// Login authenticates an agent.
// @Summary Agent login
// @Description Authenticates an agent with an account and password and returns a JWT.
// @Tags Agent
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "Login credentials"
// @Success 200 {object} response.Response{data=model.LoginResponse}
// @Router /api/v1/agent/login [post]
func (h *AgentHandler) Login(ctx context.Context, c *app.RequestContext) {
	var req model.LoginRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	resp, err := h.agentSvc.Login(ctx, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Logout logs out an agent.
// @Summary Agent logout
// @Description Logs out the agent and automatically checks out.
// @Tags Agent
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/agent/logout [post]
func (h *AgentHandler) Logout(ctx context.Context, c *app.RequestContext) {
	agentID := GetAgentID(c)
	tenantID := GetTenantID(c)

	if agentID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	if err := h.agentSvc.Logout(ctx, tenantID, agentID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Checkin checks in an agent.
// @Summary Agent check-in
// @Description Binds an extension, joins skill groups, and enters the working state.
// @Tags Agent
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.CheckinRequest true "Check-in information"
// @Success 200 {object} response.Response
// @Router /api/v1/agent/checkin [post]
func (h *AgentHandler) Checkin(ctx context.Context, c *app.RequestContext) {
	agentID := GetAgentID(c)
	tenantID := GetTenantID(c)

	if agentID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	var req model.CheckinRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	if err := h.agentSvc.Checkin(ctx, tenantID, agentID, &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Checkout checks out an agent.
// @Summary Agent checkout
// @Description Leaves the working state and unbinds the extension.
// @Tags Agent
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.CheckoutRequest true "Checkout information"
// @Success 200 {object} response.Response
// @Router /api/v1/agent/checkout [post]
func (h *AgentHandler) Checkout(ctx context.Context, c *app.RequestContext) {
	agentID := GetAgentID(c)
	tenantID := GetTenantID(c)

	if agentID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	var req model.CheckoutRequest
	_ = c.Bind(&req)

	if err := h.agentSvc.Checkout(ctx, tenantID, agentID, &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Heartbeat keeps an agent session alive.
// @Summary Agent heartbeat
// @Description Processes periodic agent heartbeats to maintain online status.
// @Tags Agent
// @Accept json
// @Produce json
// @Param agentId path string true "Agent ID"
// @Param X-Tenant-Id header string true "Tenant ID"
// @Success 200 {object} response.Response{data=model.HeartbeatResponse}
// @Router /api/v1/agent/heartbeat/{agentId} [post]
func (h *AgentHandler) Heartbeat(ctx context.Context, c *app.RequestContext) {
	agentID := c.Param("agentId")
	tenantID := string(c.GetHeader("X-Tenant-Id"))
	if tenantID == "" {
		// Also accept the tenant ID from the query string.
		tenantID = string(c.Query("tenant_id"))
	}

	if agentID == "" || tenantID == "" {
		response.Fail(c, errors.ErrParamError)
		return
	}

	resp, err := h.agentSvc.Heartbeat(ctx, tenantID, agentID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Busy marks an agent as busy.
// @Summary Mark an agent busy
// @Description Sets the agent to the busy state.
// @Tags Agent
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.StatusChangeRequest true "Busy reason"
// @Success 200 {object} response.Response
// @Router /api/v1/agent/busy [post]
func (h *AgentHandler) Busy(ctx context.Context, c *app.RequestContext) {
	agentID := GetAgentID(c)
	tenantID := GetTenantID(c)

	if agentID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	var req model.StatusChangeRequest
	_ = c.Bind(&req)

	if err := h.agentSvc.Busy(ctx, tenantID, agentID, req.Reason); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Ready marks an agent as ready.
// @Summary Mark an agent ready
// @Description Sets the agent to the ready state.
// @Tags Agent
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/agent/ready [post]
func (h *AgentHandler) Ready(ctx context.Context, c *app.RequestContext) {
	agentID := GetAgentID(c)
	tenantID := GetTenantID(c)

	if agentID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	if err := h.agentSvc.Ready(ctx, tenantID, agentID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// GetInfo returns the current agent's information.
// @Summary Get agent information
// @Description Returns detailed information and real-time status for the current agent.
// @Tags Agent
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=model.AgentInfoResponse}
// @Router /api/v1/agent/info [get]
func (h *AgentHandler) GetInfo(ctx context.Context, c *app.RequestContext) {
	agentID := GetAgentID(c)
	tenantID := GetTenantID(c)

	if agentID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	info, err := h.agentSvc.GetAgentInfo(ctx, tenantID, agentID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, info)
}

// GetStatus returns the current agent's status.
// @Summary Get agent status
// @Description Returns the current agent's real-time status.
// @Tags Agent
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=model.AgentStatusResponse}
// @Router /api/v1/agent/status [get]
func (h *AgentHandler) GetStatus(ctx context.Context, c *app.RequestContext) {
	agentID := GetAgentID(c)
	tenantID := GetTenantID(c)

	if agentID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	status, err := h.agentSvc.GetAgentStatus(ctx, tenantID, agentID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, status)
}
