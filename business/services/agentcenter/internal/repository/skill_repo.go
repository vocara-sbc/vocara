package repository

import (
	"context"
	"fmt"
	"strconv"

	"vocara/services/agentcenter/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SkillRepo provides skill group data access.
type SkillRepo struct {
	db *gorm.DB
}

// NewSkillRepo creates a skill group repository.
func NewSkillRepo(db *gorm.DB) *SkillRepo {
	return &SkillRepo{db: db}
}

// GetBySkillID returns a skill group by ID; skillID is the string form of its primary key.
func (r *SkillRepo) GetBySkillID(ctx context.Context, tenantID, skillID string) (*model.SkillGroup, error) {
	var skill model.SkillGroup
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", parseUintOrZero(tenantID), parseUintOrZero(skillID)).
		First(&skill).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get skill failed: %w", err)
	}
	return &skill, nil
}

// ListByTenant returns the skill groups for a tenant.
func (r *SkillRepo) ListByTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.SkillGroup, int64, error) {
	var skills []*model.SkillGroup
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SkillGroup{}).
		Where("tenant_id = ? AND status = 1", tenantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count skills failed: %w", err)
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).
		Order("priority DESC, id ASC").
		Find(&skills).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list skills failed: %w", err)
	}
	return skills, total, nil
}

// GetAgentsBySkill returns the agents in a skill group.
func (r *SkillRepo) GetAgentsBySkill(ctx context.Context, tenantID, skillID string) ([]*model.AgentSkill, error) {
	var relations []*model.AgentSkill
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND skill_group_id = ?", parseUintOrZero(tenantID), parseUintOrZero(skillID)).
		Find(&relations).Error
	if err != nil {
		return nil, fmt.Errorf("get agents by skill failed: %w", err)
	}
	return relations, nil
}

// GetAgentSkillCount returns the number of skill groups for an agent; agentID is the agent number.
func (r *SkillRepo) GetAgentSkillCount(ctx context.Context, tenantID, agentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.AgentSkill{}).
		Where("tenant_id = ? AND agent_number = ?", parseUintOrZero(tenantID), agentID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count agent skills failed: %w", err)
	}
	return count, nil
}

// AddAgentSkill adds a skill group to an agent.
func (r *SkillRepo) AddAgentSkill(ctx context.Context, relation *model.AgentSkill) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(relation).Error
}

// RemoveAgentSkill removes a skill group from an agent; agentID is the agent number.
func (r *SkillRepo) RemoveAgentSkill(ctx context.Context, tenantID, agentID, skillID string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND agent_number = ? AND skill_group_id = ?", parseUintOrZero(tenantID), agentID, parseUintOrZero(skillID)).
		Delete(&model.AgentSkill{}).Error
}

// parseUintOrZero converts a string to uint64 and returns 0 on parse failure.
func parseUintOrZero(s string) uint64 {
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

// Create creates a skill group.
func (r *SkillRepo) Create(ctx context.Context, skill *model.SkillGroup) error {
	return r.db.WithContext(ctx).Create(skill).Error
}

// Update updates a skill group.
func (r *SkillRepo) Update(ctx context.Context, skill *model.SkillGroup) error {
	return r.db.WithContext(ctx).Save(skill).Error
}
