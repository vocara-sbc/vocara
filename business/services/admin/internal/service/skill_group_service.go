package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// SkillGroupService provides skill group management services.
type SkillGroupService struct {
	skillRepo *repository.SkillGroupRepository
	agentRepo *repository.AgentRepository
}

// NewSkillGroupService creates a skill group service.
func NewSkillGroupService() *SkillGroupService {
	return &SkillGroupService{
		skillRepo: repository.NewSkillGroupRepository(),
		agentRepo: repository.NewAgentRepository(),
	}
}

// SkillGroupSaveDTO contains skill group data for creation.
type SkillGroupSaveDTO struct {
	SkillName string `json:"skill_name"`
	SkillCode string `json:"skill_code"`
	SkillType int    `json:"skill_type"`
	Priority  int    `json:"priority"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}

// SkillGroupUpdateDTO contains skill group data for updates.
type SkillGroupUpdateDTO struct {
	ID        uint64 `json:"id"`
	SkillName string `json:"skill_name"`
	SkillType int    `json:"skill_type"`
	Priority  int    `json:"priority"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}

// Page returns a paginated list of skill groups.
func (s *SkillGroupService) Page(ctx context.Context, tenantID uint64, q *repository.SkillGroupPageQuery) ([]*model.SkillGroup, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.skillRepo.PageQuery(ctx, tenantID, q)
}

// List returns all skill groups.
func (s *SkillGroupService) List(ctx context.Context, tenantID uint64) ([]*model.SkillGroup, error) {
	return s.skillRepo.ListAll(ctx, tenantID)
}

// GetByID returns skill group details.
func (s *SkillGroupService) GetByID(ctx context.Context, tenantID, id uint64) (*model.SkillGroup, error) {
	return s.skillRepo.GetByID(ctx, tenantID, id)
}

// Create creates a skill group.
func (s *SkillGroupService) Create(ctx context.Context, tenantID uint64, dto *SkillGroupSaveDTO) (*model.SkillGroup, error) {
	if dto.SkillName == "" || dto.SkillCode == "" {
		return nil, errors.ErrParamMissing
	}

	exists, err := s.skillRepo.ExistsByCode(ctx, tenantID, dto.SkillCode, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("技能组编码已存在")
	}

	skill := &model.SkillGroup{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		SkillName: dto.SkillName,
		SkillCode: dto.SkillCode,
		SkillType: dto.SkillType,
		Priority:  dto.Priority,
		Status:    dto.Status,
		Remark:    dto.Remark,
	}
	if skill.SkillType == 0 {
		skill.SkillType = 1
	}
	if skill.Status == 0 {
		skill.Status = 1
	}

	err = s.skillRepo.Create(ctx, skill)
	if err != nil {
		return nil, err
	}

	return skill, nil
}

// Update updates a skill group.
func (s *SkillGroupService) Update(ctx context.Context, tenantID uint64, dto *SkillGroupUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	skill, err := s.skillRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	skill.SkillName = dto.SkillName
	skill.SkillType = dto.SkillType
	skill.Priority = dto.Priority
	skill.Status = dto.Status
	skill.Remark = dto.Remark

	return s.skillRepo.Update(ctx, skill)
}

// Delete deletes a skill group.
func (s *SkillGroupService) Delete(ctx context.Context, tenantID, id uint64) error {
	// Check whether the skill group contains agents.
	count, err := s.skillRepo.CountAgentsBySkillID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.ErrForbidden.WithDetail("技能组下存在座席，无法删除")
	}

	return s.skillRepo.Delete(ctx, tenantID, id)
}

// UpdateStatus updates a skill group's status.
func (s *SkillGroupService) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return s.skillRepo.UpdateStatus(ctx, tenantID, id, status)
}

// ListAgents returns the agents in a skill group.
func (s *SkillGroupService) ListAgents(ctx context.Context, tenantID, skillID uint64) ([]*model.AgentSkill, error) {
	return s.skillRepo.ListAgentsBySkillID(ctx, tenantID, skillID)
}

// AddAgents adds multiple agents to a skill group.
func (s *SkillGroupService) AddAgents(ctx context.Context, tenantID, skillID uint64, agentLevels map[uint64]int) error {
	if skillID == 0 || len(agentLevels) == 0 {
		return errors.ErrParamMissing
	}

	// Check that the skill group exists.
	skill, err := s.skillRepo.GetByID(ctx, tenantID, skillID)
	if err != nil {
		return err
	}

	// Collect the agent IDs.
	var agentIDs []uint64
	for id := range agentLevels {
		agentIDs = append(agentIDs, id)
	}

	// Retrieve agent information.
	agents, err := s.agentRepo.ListByIDs(ctx, tenantID, agentIDs)
	if err != nil {
		return err
	}

	var skills []*model.AgentSkill
	for _, agent := range agents {
		level := agentLevels[agent.ID]
		if level <= 0 {
			level = 5
		}
		skills = append(skills, &model.AgentSkill{
			TenantID:   tenantID,
			SkillGroupID:    skill.ID,
			AgentID:    agent.ID,
			AgentNumber:   agent.AgentNumber,
			AgentName:  agent.AgentName,
			SkillLevel: level,
		})
	}

	if len(skills) > 0 {
		return s.skillRepo.AddAgentSkills(ctx, skills)
	}

	return nil
}

// RemoveAgents removes agents from a skill group.
func (s *SkillGroupService) RemoveAgents(ctx context.Context, tenantID, skillID uint64, agentIDs []uint64) error {
	return s.skillRepo.RemoveAgentSkills(ctx, tenantID, skillID, agentIDs)
}

// CountAgents counts the agents in a skill group.
func (s *SkillGroupService) CountAgents(ctx context.Context, tenantID, skillID uint64) (int64, error) {
	return s.skillRepo.CountAgentsBySkillID(ctx, tenantID, skillID)
}
