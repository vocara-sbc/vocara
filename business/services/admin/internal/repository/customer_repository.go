package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"

	"gorm.io/gorm"
)

// CustomerRepository 客户主档仓储
type CustomerRepository struct {
	BaseRepository
}

// NewCustomerRepository 创建客户仓储
func NewCustomerRepository() *CustomerRepository {
	return &CustomerRepository{}
}

// Create 创建客户
func (r *CustomerRepository) Create(ctx context.Context, c *model.Customer) error {
	return r.BaseRepository.Create(ctx, c)
}

// GetByPhone 按手机号取客户；不存在返回 (nil, nil)
func (r *CustomerRepository) GetByPhone(ctx context.Context, tenantID uint64, phone string) (*model.Customer, error) {
	var c model.Customer
	err := r.GetDB().WithContext(ctx).
		Where("tenant_id = ? AND phone = ?", tenantID, phone).First(&c).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &c, nil
}

// GetByID 按主键取客户
func (r *CustomerRepository) GetByID(ctx context.Context, id uint64) (*model.Customer, error) {
	var c model.Customer
	if err := r.GetDB().WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &c, nil
}

// Save 更新客户整行（按字段，不触碰 created_at）
func (r *CustomerRepository) Save(ctx context.Context, c *model.Customer) error {
	err := r.GetDB().WithContext(ctx).Model(&model.Customer{}).Where("id = ?", c.ID).
		Updates(map[string]interface{}{
			"name":          c.Name,
			"data":          c.Data,
			"call_count":    c.CallCount,
			"last_seen_at":  c.LastSeenAt,
		}).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Page 客户分页：keyword 模糊匹配姓名/手机号
func (r *CustomerRepository) Page(ctx context.Context, tenantID uint64, keyword string, page, pageSize int) ([]*model.Customer, int64, error) {
	tx := r.GetDB().WithContext(ctx).Model(&model.Customer{}).Where("tenant_id = ?", tenantID)
	if keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where("name LIKE ? OR phone LIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	var list []*model.Customer
	if err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, total, nil
}
