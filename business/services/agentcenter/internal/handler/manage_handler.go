package handler

import (
	"context"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// ManageHandler 内部管理处理器（供 ACD 调用）
type ManageHandler struct {
	agentSvc *service.AgentService
}

// NewManageHandler 创建内部管理处理器
func NewManageHandler(agentSvc *service.AgentService) *ManageHandler {
	return &ManageHandler{
		agentSvc: agentSvc,
	}
}

// ---------- 请求结构 ----------

// TryAcquireRequest 尝试分配座席请求
type TryAcquireRequest struct {
	TenantID     string `json:"tenant_id"     binding:"required"`
	SkillGroupID string `json:"skill_group_id" binding:"required"`
}

// TryAcquireResponse 尝试分配座席响应
type TryAcquireResponse struct {
	AgentID  string `json:"agent_id"`
	DnNumber string `json:"dn_number"`
	Success  bool   `json:"success"`
}

// RingRequest 振铃通知请求
type RingRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	CallID   string `json:"call_id"   binding:"required"`
}

// ResumeReadyRequest 重置就绪请求
type ResumeReadyRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	Status   int    `json:"status"    binding:"required"`
	Force    bool   `json:"force"`
	Reason   string `json:"reason"` // 示忙原因，目标状态为 BUSY 时透传
}

// ---------- 接口实现 ----------

// TryAcquireAgent 尝试分配一个空闲座席
// @Summary 分配空闲座席
// @Description 从指定技能组分配一个空闲座席（最长空闲优先），加分布式锁防止超卖
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body TryAcquireRequest true "分配请求"
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
		// 无空闲座席是正常情况，返回成功但 success=false
		if bizErr, ok := errors.IsBizError(err); ok && bizErr.Code == errors.CodeAgentNoIdle {
			response.Success(c, TryAcquireResponse{
				Success: false,
			})
			return
		}
		response.Error(c, err)
		return
	}

	// 获取座席分机号
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

// Ring 通知座席振铃
// @Summary 通知座席振铃
// @Description ACD 通知座席进入振铃状态
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body RingRequest true "振铃请求"
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

// ResumeReady 分配失败，重置为空闲
// @Summary 重置座席为空闲
// @Description 分配失败时将座席重置为就绪状态
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body ResumeReadyRequest true "重置请求"
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

// CallLifecycleRequest 通话接听/结束通知请求
type CallLifecycleRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	CallID   string `json:"call_id"  binding:"required"`
}

// CallAnswer 通知座席通话接听
// @Summary 通知通话接听
// @Description 座席摘机后将座席状态从振铃中转为通话中
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body CallLifecycleRequest true "接听通知请求"
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

// CallEnd 通知座席通话结束
// @Summary 通知通话结束
// @Description 通话结束后将座席释放为就绪状态并重新进入空闲池
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body CallLifecycleRequest true "结束通知请求"
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

// UpdateStatus 更新座席状态
// @Summary 更新座席状态
// @Description 内部更新座席状态，可选强制模式（跳过状态机校验）
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body UpdateStatusRequest true "更新状态请求"
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

// ---------- 座席会话内部接口（供 push WS 网关调用） ----------

// SessionRequest 建立/续期/删除座席会话请求
type SessionRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	AgentID  string `json:"agent_id"  binding:"required"`
	DnNumber string `json:"dn_number"` // 仅建立时必填
}

// SessionResponse 建立会话响应
type SessionResponse struct {
	Status     int      `json:"status"`
	StatusName string   `json:"status_name"`
	DnNumber   string   `json:"dn_number"`
	Skills     []string `json:"skills"`
}

// EstablishSession WS 连接即签入（幂等）
// @Summary 建立或恢复座席会话
// @Description push WS 连接即签入：会话存在则续期恢复，不存在则校验座席/分机后新建
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body SessionRequest true "会话请求"
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

// RefreshSession 会话续期
// @Summary 座席会话续期
// @Description push 保活协程调用的会话续期入口
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body SessionRequest true "会话请求"
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

// DeleteSession 签出清会话（幂等）
// @Summary 删除座席会话
// @Description WS 断开时签出清会话，幂等：无会话直接成功
// @Tags 内部管理
// @Accept json
// @Produce json
// @Param X-Internal-Token header string true "内部服务Token"
// @Param request body SessionRequest true "会话请求"
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
