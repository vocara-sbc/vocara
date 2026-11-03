package handler

import (
	"context"
	"strconv"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// TicketHandler handles ticket management.
type TicketHandler struct {
	formSvc *service.FormService
}

// NewTicketHandler creates a ticket management handler.
func NewTicketHandler(formSvc *service.FormService) *TicketHandler {
	return &TicketHandler{formSvc: formSvc}
}

// List returns a paginated list of tickets, filtering by status and matching keyword against titles.
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

// UpdateStatus closes or reopens a ticket using body status=open/closed.
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

// Get returns ticket record details; the frontend renders the version identified by the record's template_version_id.
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
