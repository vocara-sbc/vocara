package handler

import (
	"context"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// AgentHandler 座席处理器
type AgentHandler struct {
	agentSvc *service.AgentService
}

// NewAgentHandler 创建座席处理器
func NewAgentHandler(agentSvc *service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentSvc: agentSvc,
	}
}

// Login 座席登录
// @Summary 座席登录
// @Description 座席使用账号密码登录，返回 JWT Token
// @Tags 座席
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "登录信息"
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

// Logout 座席登出
// @Summary 座席登出
// @Description 座席退出登录，自动签出
// @Tags 座席
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

// Checkin 座席签入
// @Summary 座席签入
// @Description 绑定分机、加入技能组，进入工作状态
// @Tags 座席
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.CheckinRequest true "签入信息"
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

// Checkout 座席签出
// @Summary 座席签出
// @Description 退出工作状态，解绑分机
// @Tags 座席
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.CheckoutRequest true "签出信息"
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

// Heartbeat 心跳保活
// @Summary 座席心跳
// @Description 座席定时心跳，维持在线状态
// @Tags 座席
// @Accept json
// @Produce json
// @Param agentId path string true "座席ID"
// @Param X-Tenant-Id header string true "租户ID"
// @Success 200 {object} response.Response{data=model.HeartbeatResponse}
// @Router /api/v1/agent/heartbeat/{agentId} [post]
func (h *AgentHandler) Heartbeat(ctx context.Context, c *app.RequestContext) {
	agentID := c.Param("agentId")
	tenantID := string(c.GetHeader("X-Tenant-Id"))
	if tenantID == "" {
		// 兼容：从 query 获取
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

// Busy 示忙
// @Summary 座席示忙
// @Description 座席设置为忙碌状态
// @Tags 座席
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.StatusChangeRequest true "示忙原因"
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

// Ready 示闲
// @Summary 座席示闲
// @Description 座席设置为就绪状态
// @Tags 座席
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

// GetInfo 获取当前座席信息
// @Summary 获取座席信息
// @Description 获取当前登录座席的详细信息和实时状态
// @Tags 座席
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

// GetStatus 获取座席状态
// @Summary 获取座席状态
// @Description 获取当前座席的实时状态
// @Tags 座席
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
