package handler

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// SystemHandler 系统管理处理器
type SystemHandler struct {
	systemSvc *service.SystemService
	tenantSvc *service.TenantService
}

// NewSystemHandler 创建系统管理处理器
func NewSystemHandler(systemSvc *service.SystemService, tenantSvc *service.TenantService) *SystemHandler {
	return &SystemHandler{
		systemSvc: systemSvc,
		tenantSvc: tenantSvc,
	}
}

// ===== 系统配置 =====

// ConfigList 获取系统配置列表
func (h *SystemHandler) ConfigList(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	configs, err := h.systemSvc.GetConfigList(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取系统配置失败"))
		return
	}
	response.OK(c, configs)
}

// ConfigSave 保存系统配置
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

// ConfigDelete 删除系统配置
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

// ===== 操作日志 =====

// OperationLogPage 操作日志分页
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

// ===== FS 实例管理 =====

// FSInstancePage FS实例分页
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

// FSInstanceList FS实例列表
func (h *SystemHandler) FSInstanceList(ctx context.Context, c *app.RequestContext) {
	list, err := h.systemSvc.FSList(ctx)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询FS实例列表失败"))
		return
	}
	response.OK(c, list)
}

// FSInstanceGet FS实例详情
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

// FSInstanceSave 创建FS实例
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

// FSInstanceUpdate 更新FS实例
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

// FSInstanceDelete 删除FS实例
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

// ===== 租户管理 =====

// TenantPage 租户分页
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

// TenantList 租户列表
func (h *SystemHandler) TenantList(ctx context.Context, c *app.RequestContext) {
	list, err := h.tenantSvc.List(ctx)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询租户列表失败"))
		return
	}
	response.OK(c, list)
}

// TenantGet 租户详情
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

// TenantSave 创建租户
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

// TenantUpdate 更新租户
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

// TenantDelete 删除租户
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
