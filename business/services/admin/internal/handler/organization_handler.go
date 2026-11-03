package handler

import (
	"context"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// OrganizationHandler handles organization requests.
type OrganizationHandler struct {
	orgSvc *service.OrganizationService
}

// NewOrganizationHandler creates an organization handler.
func NewOrganizationHandler(orgSvc *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{orgSvc: orgSvc}
}

// List returns all organizations so the frontend can build an organization tree.
func (h *OrganizationHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.orgSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询组织机构失败"))
		return
	}
	response.OK(c, list)
}

// Get returns organization details.
func (h *OrganizationHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	org, err := h.orgSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询组织机构失败"))
		return
	}
	response.OK(c, org)
}

// Save creates an organization.
func (h *OrganizationHandler) Save(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.OrganizationSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	org, err := h.orgSvc.Create(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建组织机构失败"))
		return
	}
	response.OK(c, org)
}

// Update updates an organization.
func (h *OrganizationHandler) Update(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.OrganizationUpdateDTO{}
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
	if err := h.orgSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新组织机构失败"))
		return
	}
	response.OKNoContent(c)
}

// Delete deletes an organization.
func (h *OrganizationHandler) Delete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.orgSvc.Delete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除组织机构失败"))
		return
	}
	response.OKNoContent(c)
}
