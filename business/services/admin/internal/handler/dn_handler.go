package handler

import (
	"context"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// DNHandler 分机处理器
type DNHandler struct {
	dnSvc *service.DNService
}

// NewDNHandler 创建分机处理器
func NewDNHandler(dnSvc *service.DNService) *DNHandler {
	return &DNHandler{
		dnSvc: dnSvc,
	}
}

// Page 分页查询
func (h *DNHandler) Page(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.DNPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.dnSvc.Page(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询分机失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// List 查询列表
func (h *DNHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.dnSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询分机列表失败"))
		return
	}
	response.OK(c, list)
}

// Get 获取详情
func (h *DNHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	dn, err := h.dnSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询分机失败"))
		return
	}
	response.OK(c, dn)
}

// Save 创建
func (h *DNHandler) Save(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.DNSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	dn, err := h.dnSvc.Create(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建分机失败"))
		return
	}
	response.OK(c, dn)
}

// Update 更新
func (h *DNHandler) Update(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.DNUpdateDTO{}
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
	if err := h.dnSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新分机失败"))
		return
	}
	response.OKNoContent(c)
}

// Delete 删除
func (h *DNHandler) Delete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.dnSvc.Delete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除分机失败"))
		return
	}
	response.OKNoContent(c)
}

// StatusCheck 状态检查
func (h *DNHandler) StatusCheck(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.dnSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询分机状态失败"))
		return
	}
	response.OK(c, list)
}

// BatchImportRequest 批量导入分机请求
type BatchImportRequest struct {
	StartDN string `json:"start_dn"`
	Count   int    `json:"count"`
}

// BatchImport 批量导入分机
func (h *DNHandler) BatchImport(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.DNBatchCreateDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	count, err := h.dnSvc.BatchCreate(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "批量导入分机失败"))
		return
	}
	response.OK(c, map[string]interface{}{
		"success": count,
	})
}

// BindAgentRequest 绑定座席请求
type BindAgentRequest struct {
	AgentID uint64 `json:"agent_id"`
}

// BindDnAgent 绑定座席到分机
func (h *DNHandler) BindDnAgent(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dnID := ParseUintParam(c, "id")
	if dnID == 0 {
		response.FailParamError(c, "分机ID不能为空")
		return
	}
	var req BindAgentRequest
	if err := c.BindAndValidate(&req); err != nil {
		BindError(c, err)
		return
	}
	if req.AgentID == 0 {
		response.FailParamError(c, "座席ID不能为空")
		return
	}
	if err := h.dnSvc.BindAgent(ctx, tenantID, dnID, req.AgentID); err != nil {
		response.Fail(c, WrapErr(err, "绑定座席失败"))
		return
	}
	response.OKNoContent(c)
}

// UnbindAgent 解绑座席
func (h *DNHandler) UnbindAgent(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dnID := ParseUintParam(c, "id")
	if dnID == 0 {
		response.FailParamError(c, "分机ID不能为空")
		return
	}
	if err := h.dnSvc.UnbindAgent(ctx, tenantID, dnID); err != nil {
		response.Fail(c, WrapErr(err, "解绑座席失败"))
		return
	}
	response.OKNoContent(c)
}

// Available 查询可用分机
func (h *DNHandler) Available(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.dnSvc.ListAvailable(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询可用分机失败"))
		return
	}
	response.OK(c, list)
}
