package repository

import (
	"context"
	"fmt"
	"strconv"

	"vocara/services/agentcenter/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SkillRepo 技能组数据访问
type SkillRepo struct {
	db *gorm.DB
}

// NewSkillRepo 创建技能组仓库
func NewSkillRepo(db *gorm.DB) *SkillRepo {
	return &SkillRepo{db: db}
}

// GetBySkillID 根据技能组ID获取（skillID 为技能组主键 ID 的字符串形式）
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

// ListByTenant 获取租户技能组列表
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

// GetAgentsBySkill 获取技能组下的座席列表
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

// GetAgentSkillCount 获取座席技能组数量（agentID 为座席工号）
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

// AddAgentSkill 座席添加技能组
func (r *SkillRepo) AddAgentSkill(ctx context.Context, relation *model.AgentSkill) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(relation).Error
}

// RemoveAgentSkill 座席移除技能组（agentID 为座席工号）
func (r *SkillRepo) RemoveAgentSkill(ctx context.Context, tenantID, agentID, skillID string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND agent_number = ? AND skill_group_id = ?", parseUintOrZero(tenantID), agentID, parseUintOrZero(skillID)).
		Delete(&model.AgentSkill{}).Error
}

// parseUintOrZero 字符串转 uint64（解析失败为 0）
func parseUintOrZero(s string) uint64 {
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

// Create 创建技能组
func (r *SkillRepo) Create(ctx context.Context, skill *model.SkillGroup) error {
	return r.db.WithContext(ctx).Create(skill).Error
}

// Update 更新技能组
func (r *SkillRepo) Update(ctx context.Context, skill *model.SkillGroup) error {
	return r.db.WithContext(ctx).Save(skill).Error
}
