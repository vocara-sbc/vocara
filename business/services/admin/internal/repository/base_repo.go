// Package repository implements the data access layer.
// It encapsulates database operations and provides GORM-based CRUD methods.
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

// BaseRepository provides common CRUD methods.
type BaseRepository struct{}

// GetDB returns the database connection.
func (r *BaseRepository) GetDB() *database.DB {
	return database.Get()
}

// GetTenantID returns the tenant ID from the context.
func (r *BaseRepository) GetTenantID(ctx context.Context) uint64 {
	if tenantID, ok := ctx.Value("tenant_id").(uint64); ok {
		return tenantID
	}
	return 0
}

// Paginate executes a paginated query.
func (r *BaseRepository) Paginate(ctx context.Context, db *gorm.DB, page, pageSize int, dest interface{}) (*database.PageResult, error) {
	query := database.PageQuery{
		Page:     page,
		PageSize: pageSize,
	}
	return database.Paginate(ctx, db, query, dest)
}

// ===== Tenant Context Utilities =====

// WithTenantScope returns a tenant-scoped query.
func WithTenantScope(db *gorm.DB, tenantID uint64) *gorm.DB {
	if tenantID > 0 {
		return db.Where("tenant_id = ?", tenantID)
	}
	return db
}

// ===== Common Operations =====

// Create creates a record.
func (r *BaseRepository) Create(ctx context.Context, value interface{}) error {
	db := r.GetDB()
	if err := db.Create(ctx, value); err != nil {
		logger.ErrorCtx(ctx, "create record failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Save saves a record.
func (r *BaseRepository) Save(ctx context.Context, value interface{}) error {
	db := r.GetDB()
	if err := db.Save(ctx, value); err != nil {
		logger.ErrorCtx(ctx, "save record failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete deletes a record.
func (r *BaseRepository) Delete(ctx context.Context, value interface{}, id uint64) error {
	db := r.GetDB()
	if err := db.Delete(ctx, value, id); err != nil {
		logger.ErrorCtx(ctx, "delete record failed", zap.Error(err), zap.Uint64("id", id))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// FirstByID finds a record by ID within a tenant scope.
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

// ExistsByField checks whether a field value exists within a tenant scope, excluding the specified ID.
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

// ListAll returns all records within a tenant scope.
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

// BatchCreate creates records in a batch.
func (r *BaseRepository) BatchCreate(ctx context.Context, values interface{}) error {
	db := r.GetDB()
	if err := db.WithContext(ctx).Create(values).Error; err != nil {
		logger.ErrorCtx(ctx, "batch create records failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateStatus updates a record's status.
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

// Keep the model package referenced to avoid an unused import.
var _ = model.Agent{}
