package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// SkillGroupRepository 技能组数据访问层
type SkillGroupRepository struct {
	BaseRepository
}

// NewSkillGroupRepository 创建技能组仓储
func NewSkillGroupRepository() *SkillGroupRepository {
	return &SkillGroupRepository{}
}

// SkillGroupPageQuery 技能组分页查询条件
type SkillGroupPageQuery struct {
	Page      int    `json:"page" query:"page"`
	PageSize  int    `json:"page_size" query:"page_size"`
	SkillName string `json:"skill_name" query:"skill_name"`
	SkillCode string `json:"skill_code" query:"skill_code"`
	SkillType *int   `json:"skill_type" query:"skill_type"`
	Status    *int   `json:"status" query:"status"`
	Keyword   string `json:"keyword" query:"keyword"`
}

// PageQuery 分页查询
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

// ListAll 查询所有技能组
func (r *SkillGroupRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.SkillGroup, error) {
	var list []*model.SkillGroup
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "id DESC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 根据ID查询
func (r *SkillGroupRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.SkillGroup, error) {
	var skill model.SkillGroup
	if err := r.FirstByID(ctx, tenantID, &skill, id); err != nil {
		return nil, err
	}
	return &skill, nil
}

// Create 创建
func (r *SkillGroupRepository) Create(ctx context.Context, skill *model.SkillGroup) error {
	return r.BaseRepository.Create(ctx, skill)
}

// Update 更新
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

// Delete 删除
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

// ExistsByCode 检查编码是否存在
func (r *SkillGroupRepository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.SkillGroup{}, "skill_code", code, excludeID)
}

// UpdateStatus 更新状态
func (r *SkillGroupRepository) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return r.BaseRepository.UpdateStatus(ctx, tenantID, &model.SkillGroup{}, id, status)
}

// UpdateAgentCount 更新座席数量
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

// ===== 座席-技能组关系 =====

// ListAgentsBySkillID 查询技能组的座席列表
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

// AddAgentSkills 批量添加座席技能
func (r *SkillGroupRepository) AddAgentSkills(ctx context.Context, skills []*model.AgentSkill) error {
	return r.BatchCreate(ctx, skills)
}

// RemoveAgentSkills 移除技能组的座席
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

// CountAgentsBySkillID 统计技能组座席数
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
