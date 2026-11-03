package handler

import (
	"context"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// SkillGroupHandler handles skill group requests.
type SkillGroupHandler struct {
	skillSvc *service.SkillGroupService
}

// NewSkillGroupHandler creates a skill group handler.
func NewSkillGroupHandler(skillSvc *service.SkillGroupService) *SkillGroupHandler {
	return &SkillGroupHandler{
		skillSvc: skillSvc,
	}
}

// Page returns a paginated list of skill groups.
func (h *SkillGroupHandler) Page(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.SkillGroupPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.skillSvc.Page(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询技能组失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// List returns all skill groups.
func (h *SkillGroupHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.skillSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询技能组列表失败"))
		return
	}
	response.OK(c, list)
}

// Get returns skill group details.
func (h *SkillGroupHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	skill, err := h.skillSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询技能组失败"))
		return
	}
	response.OK(c, skill)
}

// Save creates a skill group.
func (h *SkillGroupHandler) Save(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.SkillGroupSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	skill, err := h.skillSvc.Create(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建技能组失败"))
		return
	}
	response.OK(c, skill)
}

// Update updates a skill group.
func (h *SkillGroupHandler) Update(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.SkillGroupUpdateDTO{}
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
	if err := h.skillSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新技能组失败"))
		return
	}
	response.OKNoContent(c)
}

// Delete deletes a skill group.
func (h *SkillGroupHandler) Delete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.skillSvc.Delete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除技能组失败"))
		return
	}
	response.OKNoContent(c)
}

// Agents returns the agents in a skill group.
func (h *SkillGroupHandler) Agents(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	skillID := ParseUintParam(c, "id")
	if skillID == 0 {
		response.FailParamError(c, "技能组ID不能为空")
		return
	}
	agents, err := h.skillSvc.ListAgents(ctx, tenantID, skillID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询技能组座席失败"))
		return
	}
	response.OK(c, agents)
}

// SetAgentsRequest is a request to set a skill group's agents.
type SetAgentsRequest struct {
	AgentLevels map[uint64]int `json:"agent_levels"`
}

// SetAgents sets a skill group's agents.
func (h *SkillGroupHandler) SetAgents(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	skillID := ParseUintParam(c, "id")
	if skillID == 0 {
		response.FailParamError(c, "技能组ID不能为空")
		return
	}
	var req SetAgentsRequest
	if err := c.BindAndValidate(&req); err != nil {
		BindError(c, err)
		return
	}

	// Remove all listed agents before adding them with their new levels.
	agentIDs := make([]uint64, 0, len(req.AgentLevels))
	for id := range req.AgentLevels {
		agentIDs = append(agentIDs, id)
	}
	_ = h.skillSvc.RemoveAgents(ctx, tenantID, skillID, agentIDs)

	if len(req.AgentLevels) > 0 {
		if err := h.skillSvc.AddAgents(ctx, tenantID, skillID, req.AgentLevels); err != nil {
			response.Fail(c, WrapErr(err, "设置技能组座席失败"))
			return
		}
	}
	response.OKNoContent(c)
}
