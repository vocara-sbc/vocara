package handler

import (
	"context"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/errors"
	"vocara/pkg/license"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// AgentHandler 座席处理器
type AgentHandler struct {
	agentSvc       *service.AgentService
	licenseManager *license.Manager
}

// NewAgentHandler 创建座席处理器
func NewAgentHandler(agentSvc *service.AgentService, licenseManager *license.Manager) *AgentHandler {
	return &AgentHandler{
		agentSvc:       agentSvc,
		licenseManager: licenseManager,
	}
}

// Page 分页查询座席
func (h *AgentHandler) Page(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)

	q := &repository.AgentPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}

	list, total, err := h.agentSvc.Page(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询座席失败"))
		return
	}

	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// List 查询座席列表
func (h *AgentHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	keyword := c.Query("keyword")

	list, err := h.agentSvc.List(ctx, tenantID, keyword)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询座席列表失败"))
		return
	}
	response.OK(c, list)
}

// Get 获取座席详情
func (h *AgentHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}

	agent, err := h.agentSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询座席失败"))
		return
	}
	response.OK(c, agent)
}

// Save 保存座席
func (h *AgentHandler) Save(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)

	dto := &service.AgentSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}

	agent, err := h.agentSvc.Create(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建座席失败"))
		return
	}
	response.OK(c, agent)
}

// Update 更新座席
func (h *AgentHandler) Update(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)

	dto := &service.AgentUpdateDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	if dto.ID == 0 {
		dto.ID = ParseUintParam(c, "id")
	}
	if dto.ID == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}

	if err := h.agentSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新座席失败"))
		return
	}
	response.OKNoContent(c)
}

// Delete 删除座席
func (h *AgentHandler) Delete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}

	if err := h.agentSvc.Delete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除座席失败"))
		return
	}
	response.OKNoContent(c)
}

// UpdateSkill 更新座席技能
type UpdateSkillRequest struct {
	SkillItems []service.SkillAssignItem `json:"skill_items"`
}

func (h *AgentHandler) UpdateSkill(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	agentID := ParseUintParam(c, "id")
	if agentID == 0 {
		response.FailParamError(c, "座席ID不能为空")
		return
	}

	var req UpdateSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		BindError(c, err)
		return
	}

	dto := &service.AgentSkillAssignDTO{
		AgentID:    agentID,
		SkillItems: req.SkillItems,
	}

	if err := h.agentSvc.AssignSkills(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新座席技能失败"))
		return
	}
	response.OKNoContent(c)
}

// BindDn 绑定分机
type BindDnRequest struct {
	Dn string `json:"dn"`
}

func (h *AgentHandler) BindDn(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	agentID := ParseUintParam(c, "id")
	if agentID == 0 {
		response.FailParamError(c, "座席ID不能为空")
		return
	}

	var req BindDnRequest
	if err := c.BindAndValidate(&req); err != nil {
		BindError(c, err)
		return
	}

	// 这里直接使用 agentService 暂没有 BindDn 方法，使用更新方式
	dto := &service.AgentUpdateDTO{
		ID: agentID,
		DN: req.Dn,
	}
	// 通过 update 来更新
	agent, err := h.agentSvc.GetByID(ctx, tenantID, agentID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询座席失败"))
		return
	}
	dto.AgentName = agent.AgentName
	dto.Role = agent.Role
	dto.Status = agent.Status
	dto.OrgID = agent.OrgID

	if err := h.agentSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "绑定分机失败"))
		return
	}
	response.OKNoContent(c)
}

// ResetPassword 重置密码
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func (h *AgentHandler) ResetPassword(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}

	var req ResetPasswordRequest
	if err := c.BindAndValidate(&req); err != nil {
		BindError(c, err)
		return
	}

	if err := h.agentSvc.ResetPassword(ctx, tenantID, id, req.NewPassword); err != nil {
		response.Fail(c, WrapErr(err, "重置密码失败"))
		return
	}
	response.OKNoContent(c)
}

// Enable 启用座席
func (h *AgentHandler) Enable(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}

	if err := h.agentSvc.UpdateStatus(ctx, tenantID, id, 1); err != nil {
		response.Fail(c, WrapErr(err, "启用座席失败"))
		return
	}
	response.OKNoContent(c)
}

// Disable 禁用座席
func (h *AgentHandler) Disable(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}

	if err := h.agentSvc.UpdateStatus(ctx, tenantID, id, 0); err != nil {
		response.Fail(c, WrapErr(err, "禁用座席失败"))
		return
	}
	response.OKNoContent(c)
}

// Import 批量导入座席（简化版，接收 JSON 数组）
func (h *AgentHandler) Import(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)

	// 简化：接收 JSON 数组批量创建
	var dtos []service.AgentSaveDTO
	if err := c.BindAndValidate(&dtos); err != nil {
		BindError(c, err)
		return
	}

	var success int
	var failed int
	for _, dto := range dtos {
		_, err := h.agentSvc.Create(ctx, tenantID, &dto)
		if err != nil {
			failed++
			continue
		}
		success++
	}

	response.OK(c, map[string]interface{}{
		"success": success,
		"failed":  failed,
	})
}

// Export 导出座席（简化版，返回数据由前端处理）
func (h *AgentHandler) Export(ctx context.Context, c *app.RequestContext) {
	if err := CheckReportEnabled(h.licenseManager); err != nil {
		response.Fail(c, WrapErr(err, "报表导出失败"))
		return
	}
	tenantID := GetTenantID(ctx, c)

	q := &repository.AgentPageQuery{
		Page:     1,
		PageSize: 10000,
	}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}

	list, total, err := h.agentSvc.Page(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "导出座席失败"))
		return
	}

	response.OK(c, map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// GetStatusCount 获取座席状态统计
func (h *AgentHandler) GetStatusCount(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)

	counts, err := h.agentSvc.GetStatusCount(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取状态统计失败"))
		return
	}
	response.OK(c, counts)
}

// 确保 errors 被使用
var _ = errors.CodeSuccess
