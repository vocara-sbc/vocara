package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// SkillGroupService 技能组服务
type SkillGroupService struct {
	skillRepo *repository.SkillGroupRepository
	agentRepo *repository.AgentRepository
}

// NewSkillGroupService 创建技能组服务
func NewSkillGroupService() *SkillGroupService {
	return &SkillGroupService{
		skillRepo: repository.NewSkillGroupRepository(),
		agentRepo: repository.NewAgentRepository(),
	}
}

// SkillGroupSaveDTO 技能组保存
type SkillGroupSaveDTO struct {
	SkillName string `json:"skill_name"`
	SkillCode string `json:"skill_code"`
	SkillType int    `json:"skill_type"`
	Priority  int    `json:"priority"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}

// SkillGroupUpdateDTO 技能组更新
type SkillGroupUpdateDTO struct {
	ID        uint64 `json:"id"`
	SkillName string `json:"skill_name"`
	SkillType int    `json:"skill_type"`
	Priority  int    `json:"priority"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}

// Page 分页查询
func (s *SkillGroupService) Page(ctx context.Context, tenantID uint64, q *repository.SkillGroupPageQuery) ([]*model.SkillGroup, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.skillRepo.PageQuery(ctx, tenantID, q)
}

// List 查询列表
func (s *SkillGroupService) List(ctx context.Context, tenantID uint64) ([]*model.SkillGroup, error) {
	return s.skillRepo.ListAll(ctx, tenantID)
}

// GetByID 获取详情
func (s *SkillGroupService) GetByID(ctx context.Context, tenantID, id uint64) (*model.SkillGroup, error) {
	return s.skillRepo.GetByID(ctx, tenantID, id)
}

// Create 创建
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

// Update 更新
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

// Delete 删除
func (s *SkillGroupService) Delete(ctx context.Context, tenantID, id uint64) error {
	// 检查技能组下是否有座席
	count, err := s.skillRepo.CountAgentsBySkillID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.ErrForbidden.WithDetail("技能组下存在座席，无法删除")
	}

	return s.skillRepo.Delete(ctx, tenantID, id)
}

// UpdateStatus 更新状态
func (s *SkillGroupService) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return s.skillRepo.UpdateStatus(ctx, tenantID, id, status)
}

// ListAgents 查询技能组座席列表
func (s *SkillGroupService) ListAgents(ctx context.Context, tenantID, skillID uint64) ([]*model.AgentSkill, error) {
	return s.skillRepo.ListAgentsBySkillID(ctx, tenantID, skillID)
}

// AddAgents 批量添加座席到技能组
func (s *SkillGroupService) AddAgents(ctx context.Context, tenantID, skillID uint64, agentLevels map[uint64]int) error {
	if skillID == 0 || len(agentLevels) == 0 {
		return errors.ErrParamMissing
	}

	// 检查技能组是否存在
	skill, err := s.skillRepo.GetByID(ctx, tenantID, skillID)
	if err != nil {
		return err
	}

	// 获取座席ID列表
	var agentIDs []uint64
	for id := range agentLevels {
		agentIDs = append(agentIDs, id)
	}

	// 查询座席信息
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

// RemoveAgents 从技能组移除座席
func (s *SkillGroupService) RemoveAgents(ctx context.Context, tenantID, skillID uint64, agentIDs []uint64) error {
	return s.skillRepo.RemoveAgentSkills(ctx, tenantID, skillID, agentIDs)
}

// CountAgents 统计技能组座席数
func (s *SkillGroupService) CountAgents(ctx context.Context, tenantID, skillID uint64) (int64, error) {
	return s.skillRepo.CountAgentsBySkillID(ctx, tenantID, skillID)
}
