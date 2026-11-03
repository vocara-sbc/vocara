package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// OrganizationRepository provides data access for organizations.
type OrganizationRepository struct {
	BaseRepository
}

// NewOrganizationRepository creates an organization repository.
func NewOrganizationRepository() *OrganizationRepository {
	return &OrganizationRepository{}
}

// ListAll returns all organizations for building a tree, ordered by sort value and ID.
func (r *OrganizationRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.Organization, error) {
	var list []*model.Organization
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "sort ASC, id ASC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID retrieves an organization by primary key.
func (r *OrganizationRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.Organization, error) {
	var org model.Organization
	if err := r.FirstByID(ctx, tenantID, &org, id); err != nil {
		return nil, err
	}
	return &org, nil
}

// ExistsByCode checks whether an organization code already exists.
func (r *OrganizationRepository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.Organization{}, "org_code", code, excludeID)
}

// Create creates an organization.
func (r *OrganizationRepository) Create(ctx context.Context, org *model.Organization) error {
	return r.BaseRepository.Create(ctx, org)
}

// Update updates individual organization fields.
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

// Delete deletes an organization.
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

// CountChildren counts child organizations.
func (r *OrganizationRepository) CountChildren(ctx context.Context, tenantID, parentID uint64) (int64, error) {
	var count int64
	tx := WithTenantScope(r.GetDB().WithContext(ctx).Model(&model.Organization{}), tenantID)
	if err := tx.Where("parent_id = ?", parentID).Count(&count).Error; err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}
