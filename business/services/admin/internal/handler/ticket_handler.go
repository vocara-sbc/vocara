package handler

import (
	"context"
	"strconv"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// TicketHandler 工单管理
type TicketHandler struct {
	formSvc *service.FormService
}

// NewTicketHandler 创建工单管理 handler
func NewTicketHandler(formSvc *service.FormService) *TicketHandler {
	return &TicketHandler{formSvc: formSvc}
}

// List 工单分页：status 过滤、keyword 匹配标题
func (h *TicketHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	page, pageSize := queryPage(c)
	list, total, err := h.formSvc.ListTickets(
		ctx, tenantID, c.Query("status"), c.Query("keyword"), page, pageSize,
	)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询工单失败"))
		return
	}
	PageSuccess(c, list, total, page, pageSize)
}

// UpdateStatus 工单处理/重开：body status=open/closed
func (h *TicketHandler) UpdateStatus(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, WrapErr(err, "参数错误"))
		return
	}
	if err := h.formSvc.UpdateTicketStatus(ctx, id, c.Query("status")); err != nil {
		response.Fail(c, WrapErr(err, "更新工单状态失败"))
		return
	}
	response.OK(c, nil)
}

// Get 工单记录详情（前端按记录 template_version_id 取版本渲染）
func (h *TicketHandler) Get(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, WrapErr(err, "参数错误"))
		return
	}
	record, err := h.formSvc.GetTicketRecord(ctx, GetTenantID(ctx, c), id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询工单失败"))
		return
	}
	response.OK(c, record)
}
