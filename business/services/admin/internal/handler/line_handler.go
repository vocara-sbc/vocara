package handler

import (
	"context"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// LineHandler 线路资源处理器
type LineHandler struct {
	lineSvc *service.LineService
}

// NewLineHandler 创建线路资源处理器
func NewLineHandler(lineSvc *service.LineService) *LineHandler {
	return &LineHandler{
		lineSvc: lineSvc,
	}
}

// Page 分页查询
func (h *LineHandler) Page(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.LinePageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.lineSvc.Page(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询线路失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// List 查询列表
func (h *LineHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.lineSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询线路列表失败"))
		return
	}
	response.OK(c, list)
}

// Get 获取详情
func (h *LineHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	line, err := h.lineSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询线路失败"))
		return
	}
	response.OK(c, line)
}

// Save 创建
func (h *LineHandler) Save(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.LineSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	line, err := h.lineSvc.Create(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建线路失败"))
		return
	}
	response.OK(c, line)
}

// Update 更新
func (h *LineHandler) Update(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.LineUpdateDTO{}
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
	if err := h.lineSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新线路失败"))
		return
	}
	response.OKNoContent(c)
}

// Delete 删除
func (h *LineHandler) Delete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.lineSvc.Delete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除线路失败"))
		return
	}
	response.OKNoContent(c)
}

// Status 获取线路状态统计
func (h *LineHandler) Status(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	status, err := h.lineSvc.GetStatus(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取线路状态失败"))
		return
	}
	response.OK(c, status)
}
