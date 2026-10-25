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

// DnHandler 分机处理器
type DnHandler struct {
	dnSvc *service.DnService
}

// NewDnHandler 创建分机处理器
func NewDnHandler(dnSvc *service.DnService) *DnHandler {
	return &DnHandler{
		dnSvc: dnSvc,
	}
}

// List 分机列表
// @Summary 分机列表
// @Description 分页获取分机列表
// @Tags 分机
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
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

// Get 分机详情
// @Summary 分机详情
// @Description 根据分机ID获取分机详情
// @Tags 分机
// @Accept json
// @Produce json
// @Security Bearer
// @Param dnId path string true "分机ID"
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

// CheckOnline 检测分机在线
// @Summary 检测分机在线
// @Description 批量检测分机是否在线
// @Tags 分机
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body model.DnCheckOnlineRequest true "分机号列表"
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
