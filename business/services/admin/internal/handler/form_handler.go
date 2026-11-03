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

// FormHandler handles dynamic form requests.
type FormHandler struct {
	formSvc        *service.FormService
	categorySvc    *service.FormCategoryService
	licenseManager *license.Manager
}

// NewFormHandler creates a dynamic form handler.
func NewFormHandler(formSvc *service.FormService, categorySvc *service.FormCategoryService, licenseManager *license.Manager) *FormHandler {
	return &FormHandler{formSvc: formSvc, categorySvc: categorySvc, licenseManager: licenseManager}
}

// CategoryList returns the form category list.
func (h *FormHandler) CategoryList(ctx context.Context, c *app.RequestContext) {
	list, err := h.categorySvc.GetCategories(ctx)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单分类失败"))
		return
	}
	response.OK(c, list)
}

// CategorySave saves the complete form category list.
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

// TemplateUpdateStatus updates only whether a template is enabled, without creating a version.
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

// TemplateVersionList returns a template's version list.
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

// TemplateVersionGet returns the contents of one template version.
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

// TemplateSchemaByKey returns the current schema for a template key.
func (h *FormHandler) TemplateSchemaByKey(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	ver, err := h.formSvc.GetSchemaByKey(ctx, GetTenantID(ctx, c), key)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询模板失败"))
		return
	}
	response.OK(c, ver)
}

// TemplatePage returns a paginated list of templates.
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

// TemplateList returns all templates.
func (h *FormHandler) TemplateList(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.formSvc.TemplateList(ctx, tenantID, c.Query("category"))
	if err != nil {
		response.Fail(c, WrapErr(err, "查询表单模板失败"))
		return
	}
	response.OK(c, list)
}

// TemplateGet returns template details.
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

// TemplateSave creates a template.
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

// TemplateUpdate updates a template.
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

// TemplateDelete deletes a template.
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

// parseConditions parses dynamic query conditions from the JSON array string in the conditions query parameter.
func parseConditions(c *app.RequestContext) []repository.RecordCondition {
	raw := c.Query("conditions")
	if raw == "" {
		return nil
	}
	var conds []repository.RecordCondition
	if err := json.Unmarshal([]byte(raw), &conds); err != nil {
		return nil // Treat an invalid condition string as no conditions so the query can continue.
	}
	return conds
}

// RecordPage returns a dynamically filtered, paginated list of form instances.
func (h *FormHandler) RecordPage(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	// Query across templates when template_key is empty.
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

// RecordGet returns form instance details.
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

// RecordSave creates a form instance.
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

// RecordUpdate updates a form instance.
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

// RecordDelete deletes a form instance.
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

// RecordExport exports CSV with a UTF-8 BOM, matching the call record export format.
func (h *FormHandler) RecordExport(ctx context.Context, c *app.RequestContext) {
	if err := CheckReportEnabled(h.licenseManager); err != nil {
		response.Fail(c, WrapErr(err, "报表导出失败"))
		return
	}
	tenantID := GetTenantID(ctx, c)
	// Export across templates when template_key is empty.
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
