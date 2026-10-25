// Package repository 数据访问层
// 封装数据库操作，提供基于 GORM 的 CRUD 方法
package repository

import (
	"context"

	"gorm.io/gorm"

	"vocara/pkg/database"
	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/admin/internal/model"

	"go.uber.org/zap"
)

// BaseRepository 基础仓储，提供通用的 CRUD 方法
type BaseRepository struct{}

// GetDB 获取数据库连接
func (r *BaseRepository) GetDB() *database.DB {
	return database.Get()
}

// GetTenantID 从上下文获取租户ID
func (r *BaseRepository) GetTenantID(ctx context.Context) uint64 {
	if tenantID, ok := ctx.Value("tenant_id").(uint64); ok {
		return tenantID
	}
	return 0
}

// Paginate 分页查询
func (r *BaseRepository) Paginate(ctx context.Context, db *gorm.DB, page, pageSize int, dest interface{}) (*database.PageResult, error) {
	query := database.PageQuery{
		Page:     page,
		PageSize: pageSize,
	}
	return database.Paginate(ctx, db, query, dest)
}

// ===== 租户上下文工具 =====

// WithTenantScope 返回带租户隔离的查询
func WithTenantScope(db *gorm.DB, tenantID uint64) *gorm.DB {
	if tenantID > 0 {
		return db.Where("tenant_id = ?", tenantID)
	}
	return db
}

// ===== 通用操作 =====

// Create 创建记录
func (r *BaseRepository) Create(ctx context.Context, value interface{}) error {
	db := r.GetDB()
	if err := db.Create(ctx, value); err != nil {
		logger.ErrorCtx(ctx, "create record failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Save 保存记录
func (r *BaseRepository) Save(ctx context.Context, value interface{}) error {
	db := r.GetDB()
	if err := db.Save(ctx, value); err != nil {
		logger.ErrorCtx(ctx, "save record failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete 删除记录
func (r *BaseRepository) Delete(ctx context.Context, value interface{}, id uint64) error {
	db := r.GetDB()
	if err := db.Delete(ctx, value, id); err != nil {
		logger.ErrorCtx(ctx, "delete record failed", zap.Error(err), zap.Uint64("id", id))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// FirstByID 按ID查询（带租户隔离）
func (r *BaseRepository) FirstByID(ctx context.Context, tenantID uint64, dest interface{}, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.First(dest, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrNotFound
		}
		logger.ErrorCtx(ctx, "find record by id failed", zap.Error(err), zap.Uint64("id", id))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ExistsByField 检查字段值是否存在（带租户隔离，排除指定ID）
func (r *BaseRepository) ExistsByField(ctx context.Context, tenantID uint64, model interface{}, field string, value interface{}, excludeID uint64) (bool, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(model), tenantID)
	tx = tx.Where(field+" = ?", value)
	if excludeID > 0 {
		tx = tx.Where("id != ?", excludeID)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		logger.ErrorCtx(ctx, "check record exists failed", zap.Error(err), zap.String("field", field))
		return false, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count > 0, nil
}

// ListAll 查询所有记录（带租户隔离）
func (r *BaseRepository) ListAll(ctx context.Context, tenantID uint64, dest interface{}, order string) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if order != "" {
		tx = tx.Order(order)
	}
	if err := tx.Find(dest).Error; err != nil {
		logger.ErrorCtx(ctx, "list all records failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// BatchCreate 批量创建
func (r *BaseRepository) BatchCreate(ctx context.Context, values interface{}) error {
	db := r.GetDB()
	if err := db.WithContext(ctx).Create(values).Error; err != nil {
		logger.ErrorCtx(ctx, "batch create records failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateStatus 更新状态
func (r *BaseRepository) UpdateStatus(ctx context.Context, tenantID uint64, model interface{}, id uint64, status int) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(model), tenantID)
	result := tx.Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update status failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// 确保 model 包被引用（避免 unused import）
var _ = model.Agent{}
