package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/license"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// FormHandler 动态表单处理器
type FormHandler struct {
	formSvc        *service.FormService
	categorySvc    *service.FormCategoryService
	licenseManager *license.Manager
}

// NewFormHandler 创建动态表单处理器
func NewFormHandler(formSvc *service.FormService, categorySvc *service.FormCategoryService, licenseManager *license.Manager) *FormHandler {
	return &FormHandler{formSvc: formSvc, categorySvc: categorySvc, licenseManager: licenseManager}
}

// CategoryList 表单分类列表
func (h *FormHandler) CategoryList(ctx context.Context, c *app.RequestContext) {
	list, err := h.categorySvc.GetCategories(ctx)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单分类失败"))
		return
	}
	response.OK(c, list)
}

// CategorySave 保存表单分类（全量）
func (h *FormHandler) CategorySave(ctx context.Context, c *app.RequestContext) {
	var list []service.FormCategory
	if err := c.BindAndValidate(&list); err != nil {
		BindError(c, err)
		return
	}
	if err := h.categorySvc.SaveCategories(ctx, list); err != nil {
		response.Fail(c, WrapErr(err, "保存表单分类失败"))
		return
	}
	response.OK(c, nil)
}

// TemplateUpdateStatus 仅更新模板启停状态（不产生版本）
func (h *FormHandler) TemplateUpdateStatus(ctx context.Context, c *app.RequestContext) {
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, WrapErr(err, "参数错误"))
		return
	}
	status, err := strconv.Atoi(c.Query("status"))
	if err != nil || (status != 0 && status != 1) {
		response.Fail(c, WrapErr(err, "status 参数非法"))
		return
	}
	if err := h.formSvc.UpdateTemplateStatus(ctx, templateID, status); err != nil {
		response.Fail(c, WrapErr(err, "更新状态失败"))
		return
	}
	response.OK(c, nil)
}

// TemplateVersionList 模板历史版本列表
func (h *FormHandler) TemplateVersionList(ctx context.Context, c *app.RequestContext) {
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, WrapErr(err, "参数错误"))
		return
	}
	list, err := h.formSvc.TemplateVersionList(ctx, templateID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询版本失败"))
		return
	}
	response.OK(c, list)
}

// TemplateVersionGet 单个历史版本内容
func (h *FormHandler) TemplateVersionGet(ctx context.Context, c *app.RequestContext) {
	versionID, err := strconv.ParseUint(c.Param("versionId"), 10, 64)
	if err != nil {
		response.Fail(c, WrapErr(err, "参数错误"))
		return
	}
	ver, err := h.formSvc.TemplateVersionGet(ctx, versionID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询版本失败"))
		return
	}
	response.OK(c, ver)
}

// TemplateSchemaByKey 按模板标识取当前版本 schema
func (h *FormHandler) TemplateSchemaByKey(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	ver, err := h.formSvc.GetSchemaByKey(ctx, GetTenantID(ctx, c), key)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询模板失败"))
		return
	}
	response.OK(c, ver)
}

// TemplatePage 模板分页
func (h *FormHandler) TemplatePage(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.FormTemplatePageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.formSvc.TemplatePage(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单模板失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// TemplateList 模板全量
func (h *FormHandler) TemplateList(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.formSvc.TemplateList(ctx, tenantID, c.Query("category"))
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单模板失败"))
		return
	}
	response.OK(c, list)
}

// TemplateGet 模板详情
func (h *FormHandler) TemplateGet(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	tpl, err := h.formSvc.TemplateGet(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单模板失败"))
		return
	}
	response.OK(c, tpl)
}

// TemplateSave 创建模板
func (h *FormHandler) TemplateSave(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.FormTemplateSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	tpl, err := h.formSvc.TemplateCreate(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建表单模板失败"))
		return
	}
	response.OK(c, tpl)
}

// TemplateUpdate 更新模板
func (h *FormHandler) TemplateUpdate(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.FormTemplateUpdateDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	if err := h.formSvc.TemplateUpdate(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新表单模板失败"))
		return
	}
	response.OK(c, nil)
}

// TemplateDelete 删除模板
func (h *FormHandler) TemplateDelete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.formSvc.TemplateDelete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除表单模板失败"))
		return
	}
	response.OK(c, nil)
}

// parseConditions 解析动态查询条件（query 参数 conditions 为 JSON 数组字符串）
func parseConditions(c *app.RequestContext) []repository.RecordCondition {
	raw := c.Query("conditions")
	if raw == "" {
		return nil
	}
	var conds []repository.RecordCondition
	if err := json.Unmarshal([]byte(raw), &conds); err != nil {
		return nil // 非法条件串按无条件处理，查询语义不中断
	}
	return conds
}

// RecordPage 实例动态分页
func (h *FormHandler) RecordPage(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	// template_key 为空时跨模板分页查询
	templateKey := c.Query("template_key")
	page, pageSize := 1, 10
	if v := ParseUintParam(c, "page"); v > 0 {
		page = int(v)
	}
	if v := ParseUintParam(c, "page_size"); v > 0 {
		pageSize = int(v)
	}
	list, total, err := h.formSvc.RecordPage(ctx, tenantID, templateKey, parseConditions(c), page, pageSize)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单数据失败"))
		return
	}
	PageSuccess(c, list, total, page, pageSize)
}

// RecordGet 实例详情
func (h *FormHandler) RecordGet(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	record, err := h.formSvc.RecordGet(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单数据失败"))
		return
	}
	response.OK(c, record)
}

// RecordSave 创建实例
func (h *FormHandler) RecordSave(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.FormRecordSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	record, err := h.formSvc.RecordCreate(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "保存表单数据失败"))
		return
	}
	response.OK(c, record)
}

// RecordUpdate 更新实例
func (h *FormHandler) RecordUpdate(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.FormRecordUpdateDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	if err := h.formSvc.RecordUpdate(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新表单数据失败"))
		return
	}
	response.OK(c, nil)
}

// RecordDelete 删除实例
func (h *FormHandler) RecordDelete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.formSvc.RecordDelete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除表单数据失败"))
		return
	}
	response.OK(c, nil)
}

// RecordExport 导出 CSV（带 UTF-8 BOM，与通话记录导出同一形态）
func (h *FormHandler) RecordExport(ctx context.Context, c *app.RequestContext) {
	if err := CheckReportEnabled(h.licenseManager); err != nil {
		response.Fail(c, WrapErr(err, "报表导出失败"))
		return
	}
	tenantID := GetTenantID(ctx, c)
	// template_key 为空时跨模板导出
	templateKey := c.Query("template_key")
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	if err := h.formSvc.RecordExport(ctx, tenantID, templateKey, parseConditions(c), &buf); err != nil {
		response.Fail(c, WrapErr(err, "导出表单数据失败"))
		return
	}
	exportName := "全部模板"
	if templateKey != "" {
		exportName = templateKey
	}
	fileName := url.QueryEscape(fmt.Sprintf("表单数据_%s_%s.csv", exportName, time.Now().Format("20060102150405")))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", fileName))
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
}
