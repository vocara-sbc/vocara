package handler

import (
	"context"
	"strconv"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// CustomerHandler handles customer management.
type CustomerHandler struct {
	formSvc *service.FormService
}

// NewCustomerHandler creates a customer management handler.
func NewCustomerHandler(formSvc *service.FormService) *CustomerHandler {
	return &CustomerHandler{formSvc: formSvc}
}

// queryPage parses pagination parameters and applies default values.
func queryPage(c *app.RequestContext) (int, int) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page <= 0 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("page_size"))
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}
	return page, pageSize
}

// List returns a paginated list of customers, matching keyword against names and phone numbers.
func (h *CustomerHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	page, pageSize := queryPage(c)
	list, total, err := h.formSvc.ListCustomers(ctx, tenantID, c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询客户失败"))
		return
	}
	PageSuccess(c, list, total, page, pageSize)
}

// Get returns customer details, including associated records.
func (h *CustomerHandler) Get(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, WrapErr(err, "参数错误"))
		return
	}
	customer, records, err := h.formSvc.GetCustomerDetail(ctx, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询客户详情失败"))
		return
	}
	response.OK(c, map[string]interface{}{
		"customer": customer,
		"records":  records,
	})
}
