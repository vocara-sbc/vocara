package handler

import (
	"context"
	"strconv"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// SkillHandler 技能组处理器
type SkillHandler struct {
	skillSvc *service.SkillService
}

// NewSkillHandler 创建技能组处理器
func NewSkillHandler(skillSvc *service.SkillService) *SkillHandler {
	return &SkillHandler{
		skillSvc: skillSvc,
	}
}

// List 技能组列表
// @Summary 技能组列表
// @Description 分页获取技能组列表
// @Tags 技能组
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/skill/list [get]
func (h *SkillHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	list, total, err := h.skillSvc.ListSkill(ctx, tenantID, page, pageSize)
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

// Get 技能组详情
// @Summary 技能组详情
// @Description 获取技能组详细信息
// @Tags 技能组
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "技能组ID"
// @Success 200 {object} response.Response{data=model.SkillInfoResponse}
// @Router /api/v1/skill/{id} [get]
func (h *SkillHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	skillID := c.Param("id")
	if skillID == "" {
		response.Fail(c, errors.ErrParamError)
		return
	}

	info, err := h.skillSvc.GetSkill(ctx, tenantID, skillID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, info)
}

// GetAgents 技能组下的座席
// @Summary 技能组座席列表
// @Description 获取技能组下的所有座席及其实时状态
// @Tags 技能组
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "技能组ID"
// @Success 200 {object} response.Response{data=[]model.SkillAgentInfo}
// @Router /api/v1/skill/{id}/agents [get]
func (h *SkillHandler) GetAgents(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	skillID := c.Param("id")
	if skillID == "" {
		response.Fail(c, errors.ErrParamError)
		return
	}

	agents, err := h.skillSvc.GetSkillAgents(ctx, tenantID, skillID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, agents)
}
