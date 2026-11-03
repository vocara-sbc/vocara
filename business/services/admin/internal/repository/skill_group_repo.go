package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// SkillGroupRepository provides skill group data access.
type SkillGroupRepository struct {
	BaseRepository
}

// NewSkillGroupRepository creates a skill group repository.
func NewSkillGroupRepository() *SkillGroupRepository {
	return &SkillGroupRepository{}
}

// SkillGroupPageQuery contains skill group pagination filters.
type SkillGroupPageQuery struct {
	Page      int    `json:"page" query:"page"`
	PageSize  int    `json:"page_size" query:"page_size"`
	SkillName string `json:"skill_name" query:"skill_name"`
	SkillCode string `json:"skill_code" query:"skill_code"`
	SkillType *int   `json:"skill_type" query:"skill_type"`
	Status    *int   `json:"status" query:"status"`
	Keyword   string `json:"keyword" query:"keyword"`
}

// PageQuery executes a paginated skill group query.
func (r *SkillGroupRepository) PageQuery(ctx context.Context, tenantID uint64, q *SkillGroupPageQuery) ([]*model.SkillGroup, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.SkillGroup{}), tenantID)

	if q.SkillName != "" {
		tx = tx.Where("skill_name LIKE ?", "%"+q.SkillName+"%")
	}
	if q.SkillCode != "" {
		tx = tx.Where("skill_code LIKE ?", "%"+q.SkillCode+"%")
	}
	if q.SkillType != nil {
		tx = tx.Where("skill_type = ?", *q.SkillType)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(skill_name LIKE ? OR skill_code LIKE ?)", keyword, keyword)
	}

	result, err := r.Paginate(ctx, tx.Order("id DESC"), q.Page, q.PageSize, &[]*model.SkillGroup{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.SkillGroup)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// ListAll returns all skill groups.
func (r *SkillGroupRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.SkillGroup, error) {
	var list []*model.SkillGroup
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "id DESC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID returns a skill group by ID.
func (r *SkillGroupRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.SkillGroup, error) {
	var skill model.SkillGroup
	if err := r.FirstByID(ctx, tenantID, &skill, id); err != nil {
		return nil, err
	}
	return &skill, nil
}

// Create creates a skill group.
func (r *SkillGroupRepository) Create(ctx context.Context, skill *model.SkillGroup) error {
	return r.BaseRepository.Create(ctx, skill)
}

// Update updates a skill group.
func (r *SkillGroupRepository) Update(ctx context.Context, skill *model.SkillGroup) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.SkillGroup{}), skill.TenantID)
	result := tx.Where("id = ?", skill.ID).Updates(map[string]interface{}{
		"skill_name": skill.SkillName,
		"skill_type": skill.SkillType,
		"status":     skill.Status,
		"priority":   skill.Priority,
		"remark":     skill.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update skill group failed", zap.Error(result.Error), zap.Uint64("id", skill.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete deletes a skill group.
func (r *SkillGroupRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.SkillGroup{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete skill group failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrSkillNotFound
	}
	return nil
}

// ExistsByCode checks whether a skill group code exists.
func (r *SkillGroupRepository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.SkillGroup{}, "skill_code", code, excludeID)
}

// UpdateStatus updates a skill group's status.
func (r *SkillGroupRepository) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return r.BaseRepository.UpdateStatus(ctx, tenantID, &model.SkillGroup{}, id, status)
}

// UpdateAgentCount updates a skill group's agent count.
func (r *SkillGroupRepository) UpdateAgentCount(ctx context.Context, tenantID, id uint64, count int) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.SkillGroup{}), tenantID)
	result := tx.Where("id = ?", id).Update("agent_count", count)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update skill agent count failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ===== Agent-Skill Group Associations =====

// ListAgentsBySkillID returns the agents in a skill group.
func (r *SkillGroupRepository) ListAgentsBySkillID(ctx context.Context, tenantID, skillID uint64) ([]*model.AgentSkill, error) {
	db := r.GetDB()
	var list []*model.AgentSkill
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("skill_group_id = ?", skillID).Order("skill_level DESC, id DESC").Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "list agents by skill failed", zap.Error(err), zap.Uint64("skill_id", skillID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// AddAgentSkills adds agent skills in a batch.
func (r *SkillGroupRepository) AddAgentSkills(ctx context.Context, skills []*model.AgentSkill) error {
	return r.BatchCreate(ctx, skills)
}

// RemoveAgentSkills removes agents from a skill group.
func (r *SkillGroupRepository) RemoveAgentSkills(ctx context.Context, tenantID, skillID uint64, agentIDs []uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("skill_group_id = ? AND agent_id IN ?", skillID, agentIDs).Delete(&model.AgentSkill{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "remove agent skills failed", zap.Error(result.Error), zap.Uint64("skill_id", skillID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// CountAgentsBySkillID counts the agents in a skill group.
func (r *SkillGroupRepository) CountAgentsBySkillID(ctx context.Context, tenantID, skillID uint64) (int64, error) {
	db := r.GetDB()
	var count int64
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.AgentSkill{}), tenantID)
	if err := tx.Where("skill_group_id = ?", skillID).Count(&count).Error; err != nil {
		logger.ErrorCtx(ctx, "count agents by skill failed", zap.Error(err))
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}
