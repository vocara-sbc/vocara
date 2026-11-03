package handler

import (
	"context"
	"strconv"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// SkillHandler handles skill group HTTP requests.
type SkillHandler struct {
	skillSvc *service.SkillService
}

// NewSkillHandler creates a skill group handler.
func NewSkillHandler(skillSvc *service.SkillService) *SkillHandler {
	return &SkillHandler{
		skillSvc: skillSvc,
	}
}

// List returns a paginated list of skill groups.
// @Summary List skill groups
// @Description Returns a paginated list of skill groups.
// @Tags SkillGroup
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
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

// Get returns skill group details.
// @Summary Get skill group details
// @Description Returns detailed information about a skill group.
// @Tags SkillGroup
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Skill group ID"
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

// GetAgents returns the agents in a skill group.
// @Summary List skill group agents
// @Description Returns all agents in a skill group with their real-time status.
// @Tags SkillGroup
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Skill group ID"
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
