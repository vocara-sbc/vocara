package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// OrganizationRepository 组织机构数据访问层
type OrganizationRepository struct {
	BaseRepository
}

// NewOrganizationRepository 创建组织机构仓储
func NewOrganizationRepository() *OrganizationRepository {
	return &OrganizationRepository{}
}

// ListAll 查询全部机构（树形用），按排序与ID
func (r *OrganizationRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.Organization, error) {
	var list []*model.Organization
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "sort ASC, id ASC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 按主键查询机构
func (r *OrganizationRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.Organization, error) {
	var org model.Organization
	if err := r.FirstByID(ctx, tenantID, &org, id); err != nil {
		return nil, err
	}
	return &org, nil
}

// ExistsByCode 检查机构编码是否已存在
func (r *OrganizationRepository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.Organization{}, "org_code", code, excludeID)
}

// Create 创建机构
func (r *OrganizationRepository) Create(ctx context.Context, org *model.Organization) error {
	return r.BaseRepository.Create(ctx, org)
}

// Update 更新机构（按字段）
func (r *OrganizationRepository) Update(ctx context.Context, org *model.Organization) error {
	tx := WithTenantScope(r.GetDB().WithContext(ctx).Model(&model.Organization{}), org.TenantID)
	result := tx.Where("id = ?", org.ID).Updates(map[string]interface{}{
		"org_name":    org.OrgName,
		"parent_id":   org.ParentID,
		"level":       org.Level,
		"path":        org.Path,
		"leader_id":   org.LeaderID,
		"leader_name": org.LeaderName,
		"sort":        org.Sort,
		"status":      org.Status,
		"remark":      org.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update organization failed", zap.Error(result.Error), zap.Uint64("id", org.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete 删除机构
func (r *OrganizationRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	tx := WithTenantScope(r.GetDB().WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.Organization{})
	if result.Error != nil {
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// CountChildren 统计子机构数量
func (r *OrganizationRepository) CountChildren(ctx context.Context, tenantID, parentID uint64) (int64, error) {
	var count int64
	tx := WithTenantScope(r.GetDB().WithContext(ctx).Model(&model.Organization{}), tenantID)
	if err := tx.Where("parent_id = ?", parentID).Count(&count).Error; err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}
