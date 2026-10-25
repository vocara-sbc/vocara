package repository

import (
	"context"
	"fmt"

	"vocara/services/agentcenter/internal/model"

	"gorm.io/gorm"
)

// DnRepo 分机数据访问
type DnRepo struct {
	db *gorm.DB
}

// NewDnRepo 创建分机仓库
func NewDnRepo(db *gorm.DB) *DnRepo {
	return &DnRepo{db: db}
}

// GetByDnID 根据分机ID获取
func (r *DnRepo) GetByDnID(ctx context.Context, tenantID, dnID string) (*model.Dn, error) {
	var dn model.Dn
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND dn_id = ?", tenantID, dnID).
		First(&dn).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get dn failed: %w", err)
	}
	return &dn, nil
}

// GetByDnNumber 根据分机号获取
func (r *DnRepo) GetByDnNumber(ctx context.Context, tenantID, dnNumber string) (*model.Dn, error) {
	var dn model.Dn
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND dn_number = ?", tenantID, dnNumber).
		First(&dn).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get dn by number failed: %w", err)
	}
	return &dn, nil
}

// ListByTenant 分页获取租户分机列表
func (r *DnRepo) ListByTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.Dn, int64, error) {
	var dns []*model.Dn
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Dn{}).Where("tenant_id = ?", tenantID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count dns failed: %w", err)
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&dns).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list dns failed: %w", err)
	}
	return dns, total, nil
}

// ListByNumbers 批量获取分机信息
func (r *DnRepo) ListByNumbers(ctx context.Context, tenantID string, dnNumbers []string) ([]*model.Dn, error) {
	var dns []*model.Dn
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND dn_number IN ?", tenantID, dnNumbers).
		Find(&dns).Error
	if err != nil {
		return nil, fmt.Errorf("list dns by numbers failed: %w", err)
	}
	return dns, nil
}

// Create 创建分机
func (r *DnRepo) Create(ctx context.Context, dn *model.Dn) error {
	return r.db.WithContext(ctx).Create(dn).Error
}

// Update 更新分机
func (r *DnRepo) Update(ctx context.Context, dn *model.Dn) error {
	return r.db.WithContext(ctx).Save(dn).Error
}

// BindAgent 绑定座席到分机
func (r *DnRepo) BindAgent(ctx context.Context, tenantID, dnNumber, agentID string) error {
	return r.db.WithContext(ctx).
		Model(&model.Dn{}).
		Where("tenant_id = ? AND dn_number = ?", tenantID, dnNumber).
		Updates(map[string]interface{}{
			"agent_id":     agentID,
			"agent_number": agentID,
		}).Error
}

// UnbindAgent 解绑分机座席
func (r *DnRepo) UnbindAgent(ctx context.Context, tenantID, dnNumber string) error {
	return r.db.WithContext(ctx).
		Model(&model.Dn{}).
		Where("tenant_id = ? AND dn_number = ?", tenantID, dnNumber).
		Updates(map[string]interface{}{
			"agent_id":     nil,
			"agent_number": nil,
		}).Error
}
