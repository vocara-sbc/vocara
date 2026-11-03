package handler

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// SystemHandler handles system management requests.
type SystemHandler struct {
	systemSvc *service.SystemService
	tenantSvc *service.TenantService
}

// NewSystemHandler creates a system management handler.
func NewSystemHandler(systemSvc *service.SystemService, tenantSvc *service.TenantService) *SystemHandler {
	return &SystemHandler{
		systemSvc: systemSvc,
		tenantSvc: tenantSvc,
	}
}

// ===== System Configuration =====

// ConfigList returns system configurations.
func (h *SystemHandler) ConfigList(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	configs, err := h.systemSvc.GetConfigList(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取系统配置失败"))
		return
	}
	response.OK(c, configs)
}

// ConfigSaveRequest contains a system configuration to save.
type ConfigSaveRequest struct {
	ConfigKey   string `json:"config_key"`
	ConfigValue string `json:"config_value"`
	ConfigType  string `json:"config_type"`
	ConfigDesc  string `json:"config_desc"`
}

func (h *SystemHandler) ConfigSave(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	var req ConfigSaveRequest
	if err := c.BindAndValidate(&req); err != nil {
		BindError(c, err)
		return
	}
	if req.ConfigKey == "" {
		response.FailParamError(c, "配置键不能为空")
		return
	}
	config := &model.SystemConfig{
		TenantID:    tenantID,
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigType:  req.ConfigType,
		ConfigDesc:  req.ConfigDesc,
	}
	if err := h.systemSvc.SaveConfig(ctx, config); err != nil {
		response.Fail(c, WrapErr(err, "保存配置失败"))
		return
	}
	response.OKNoContent(c)
}

// ConfigDelete deletes a system configuration.
func (h *SystemHandler) ConfigDelete(ctx context.Context, c *app.RequestContext) {
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.systemSvc.DeleteConfig(ctx, id); err != nil {
		response.Fail(c, WrapErr(err, "删除配置失败"))
		return
	}
	response.OKNoContent(c)
}

// ===== Operation Logs =====

// OperationLogPage returns a paginated list of operation logs.
func (h *SystemHandler) OperationLogPage(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.OperationLogPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.systemSvc.OperationLogPage(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询操作日志失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// ===== FS Instance Management =====

// FSInstancePage returns a paginated list of FS instances.
func (h *SystemHandler) FSInstancePage(ctx context.Context, c *app.RequestContext) {
	q := &repository.FSPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.systemSvc.FSPage(ctx, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询FS实例失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// FSInstanceList returns all FS instances.
func (h *SystemHandler) FSInstanceList(ctx context.Context, c *app.RequestContext) {
	list, err := h.systemSvc.FSList(ctx)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询FS实例列表失败"))
		return
	}
	response.OK(c, list)
}

// FSInstanceGet returns FS instance details.
func (h *SystemHandler) FSInstanceGet(ctx context.Context, c *app.RequestContext) {
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	fs, err := h.systemSvc.FSGet(ctx, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询FS实例失败"))
		return
	}
	response.OK(c, fs)
}

// FSInstanceSave creates an FS instance.
func (h *SystemHandler) FSInstanceSave(ctx context.Context, c *app.RequestContext) {
	dto := &service.FSInstanceSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	fs, err := h.systemSvc.FSCreate(ctx, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建FS实例失败"))
		return
	}
	response.OK(c, fs)
}

// FSInstanceUpdate updates an FS instance.
func (h *SystemHandler) FSInstanceUpdate(ctx context.Context, c *app.RequestContext) {
	dto := &service.FSInstanceUpdateDTO{}
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
	if err := h.systemSvc.FSUpdate(ctx, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新FS实例失败"))
		return
	}
	response.OKNoContent(c)
}

// FSInstanceDelete deletes an FS instance.
func (h *SystemHandler) FSInstanceDelete(ctx context.Context, c *app.RequestContext) {
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.systemSvc.FSDelete(ctx, id); err != nil {
		response.Fail(c, WrapErr(err, "删除FS实例失败"))
		return
	}
	response.OKNoContent(c)
}

// ===== Tenant Management =====

// TenantPage returns a paginated list of tenants.
func (h *SystemHandler) TenantPage(ctx context.Context, c *app.RequestContext) {
	q := &repository.TenantPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.tenantSvc.Page(ctx, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询租户失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// TenantList returns all tenants.
func (h *SystemHandler) TenantList(ctx context.Context, c *app.RequestContext) {
	list, err := h.tenantSvc.List(ctx)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询租户列表失败"))
		return
	}
	response.OK(c, list)
}

// TenantGet returns tenant details.
func (h *SystemHandler) TenantGet(ctx context.Context, c *app.RequestContext) {
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	tenant, err := h.tenantSvc.GetByID(ctx, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询租户失败"))
		return
	}
	response.OK(c, tenant)
}

// TenantSave creates a tenant.
func (h *SystemHandler) TenantSave(ctx context.Context, c *app.RequestContext) {
	dto := &service.TenantSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	tenant, err := h.tenantSvc.Create(ctx, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建租户失败"))
		return
	}
	response.OK(c, tenant)
}

// TenantUpdate updates a tenant.
func (h *SystemHandler) TenantUpdate(ctx context.Context, c *app.RequestContext) {
	dto := &service.TenantUpdateDTO{}
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
	if err := h.tenantSvc.Update(ctx, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新租户失败"))
		return
	}
	response.OKNoContent(c)
}

// TenantDelete deletes a tenant.
func (h *SystemHandler) TenantDelete(ctx context.Context, c *app.RequestContext) {
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.tenantSvc.Delete(ctx, id); err != nil {
		response.Fail(c, WrapErr(err, "删除租户失败"))
		return
	}
	response.OKNoContent(c)
}
