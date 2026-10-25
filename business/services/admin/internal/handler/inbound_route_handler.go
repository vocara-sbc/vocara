package handler

import (
	"context"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// InboundRouteHandler 呼入接待处理器
type InboundRouteHandler struct {
	routeSvc *service.InboundRouteService
}

// NewInboundRouteHandler 创建呼入接待处理器
func NewInboundRouteHandler(routeSvc *service.InboundRouteService) *InboundRouteHandler {
	return &InboundRouteHandler{
		routeSvc: routeSvc,
	}
}

// InboundRoutes 内部接口：返回全租户启用呼入接待的 DID→队列路由列表。
// 供 callctrl 定时拉取构建动态路由缓存（X-Internal-Token 鉴权，不挂 JWT）。
func (h *InboundRouteHandler) InboundRoutes(ctx context.Context, c *app.RequestContext) {
	items, err := h.routeSvc.ListInboundRoutes(ctx)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询呼入路由失败"))
		return
	}
	response.OK(c, items)
}

// Page 分页查询
func (h *InboundRouteHandler) Page(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.InboundRoutePageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.routeSvc.Page(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询呼入接待失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// List 查询列表
func (h *InboundRouteHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.routeSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询呼入接待列表失败"))
		return
	}
	response.OK(c, list)
}

// Get 获取详情
func (h *InboundRouteHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	config, err := h.routeSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询呼入接待失败"))
		return
	}
	response.OK(c, config)
}

// Save 创建
func (h *InboundRouteHandler) Save(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.InboundRouteSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	config, err := h.routeSvc.Create(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建呼入接待失败"))
		return
	}
	response.OK(c, config)
}

// Update 更新
func (h *InboundRouteHandler) Update(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.InboundRouteUpdateDTO{}
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
	if err := h.routeSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新呼入接待失败"))
		return
	}
	response.OKNoContent(c)
}

// Delete 删除
func (h *InboundRouteHandler) Delete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.routeSvc.Delete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除呼入接待失败"))
		return
	}
	response.OKNoContent(c)
}

// ===== 节假日管理 =====

// HolidayPage 节假日分页查询
func (h *InboundRouteHandler) HolidayPage(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.HolidayPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.routeSvc.HolidayPage(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询节假日失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// HolidayList 节假日列表
func (h *InboundRouteHandler) HolidayList(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.routeSvc.HolidayList(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询节假日列表失败"))
		return
	}
	response.OK(c, list)
}

// HolidayGet 节假日详情
func (h *InboundRouteHandler) HolidayGet(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	holiday, err := h.routeSvc.HolidayGet(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询节假日失败"))
		return
	}
	response.OK(c, holiday)
}

// HolidaySave 创建节假日
func (h *InboundRouteHandler) HolidaySave(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.HolidaySaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	holiday, err := h.routeSvc.HolidayCreate(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建节假日失败"))
		return
	}
	response.OK(c, holiday)
}

// HolidayUpdate 更新节假日
func (h *InboundRouteHandler) HolidayUpdate(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.HolidayUpdateDTO{}
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
	if err := h.routeSvc.HolidayUpdate(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新节假日失败"))
		return
	}
	response.OKNoContent(c)
}

// HolidayDelete 删除节假日
func (h *InboundRouteHandler) HolidayDelete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.routeSvc.HolidayDelete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除节假日失败"))
		return
	}
	response.OKNoContent(c)
}
