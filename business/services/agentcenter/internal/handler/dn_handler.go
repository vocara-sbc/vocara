package handler

import (
	"context"
	"strconv"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// DnHandler handles extension HTTP requests.
type DnHandler struct {
	dnSvc *service.DnService
}

// NewDnHandler creates an extension handler.
func NewDnHandler(dnSvc *service.DnService) *DnHandler {
	return &DnHandler{
		dnSvc: dnSvc,
	}
}

// List returns a paginated list of extensions.
// @Summary List extensions
// @Description Returns a paginated list of extensions.
// @Tags Extension
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/dn/list [get]
func (h *DnHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	list, total, err := h.dnSvc.ListDn(ctx, tenantID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	pages := 0
	if pageSize > 0 {
		pages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	response.PageSuccess(c, list, total, page, pageSize, pages)
}

// Get returns extension details.
// @Summary Get extension details
// @Description Returns extension details by extension ID.
// @Tags Extension
// @Accept json
// @Produce json
// @Security Bearer
// @Param dnId path string true "Extension ID"
// @Success 200 {object} response.Response{data=model.DnInfoResponse}
// @Router /api/v1/dn/{dnId} [get]
func (h *DnHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	dnID := c.Param("dnId")
	if dnID == "" {
		response.Fail(c, errors.ErrParamError)
		return
	}

	info, err := h.dnSvc.GetDn(ctx, tenantID, dnID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, info)
}

// CheckOnline checks whether extensions are online.
// @Summary Check extension online status
// @Description Checks the online status of multiple extensions.
// @Tags Extension
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.DnCheckOnlineRequest true "Extension numbers"
// @Success 200 {object} response.Response{data=model.DnCheckOnlineResponse}
// @Router /api/v1/dn/checkonline [post]
func (h *DnHandler) CheckOnline(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	var req model.DnCheckOnlineRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, errors.ErrParamError)
		return
	}

	if len(req.DnNumbers) == 0 {
		response.Success(c, model.DnCheckOnlineResponse{Results: map[string]bool{}})
		return
	}

	results, err := h.dnSvc.CheckOnline(ctx, tenantID, req.DnNumbers)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, model.DnCheckOnlineResponse{Results: results})
}
