package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// LineRepository provides line resource data access.
type LineRepository struct {
	BaseRepository
}

// NewLineRepository creates a line resource repository.
func NewLineRepository() *LineRepository {
	return &LineRepository{}
}

// LinePageQuery contains line pagination filters.
type LinePageQuery struct {
	Page           int    `json:"page" query:"page"`
	PageSize       int    `json:"page_size" query:"page_size"`
	LineName       string `json:"line_name" query:"line_name"`
	LineCode       string `json:"line_code" query:"line_code"`
	LineType       *int   `json:"line_type" query:"line_type"`
	Provider       string `json:"provider" query:"provider"`
	Status         *int   `json:"status" query:"status"`
	RegisterStatus *int   `json:"register_status" query:"register_status"`
	FSInstanceID   uint64 `json:"fs_instance_id" query:"fs_instance_id"`
	Keyword        string `json:"keyword" query:"keyword"`
}

// PageQuery executes a paginated line query.
func (r *LineRepository) PageQuery(ctx context.Context, tenantID uint64, q *LinePageQuery) ([]*model.Line, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Line{}), tenantID)

	if q.LineName != "" {
		tx = tx.Where("line_name LIKE ?", "%"+q.LineName+"%")
	}
	if q.LineCode != "" {
		tx = tx.Where("line_code LIKE ?", "%"+q.LineCode+"%")
	}
	if q.LineType != nil {
		tx = tx.Where("line_type = ?", *q.LineType)
	}
	if q.Provider != "" {
		tx = tx.Where("provider LIKE ?", "%"+q.Provider+"%")
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.RegisterStatus != nil {
		tx = tx.Where("register_status = ?", *q.RegisterStatus)
	}
	if q.FSInstanceID > 0 {
		tx = tx.Where("fs_instance_id = ?", q.FSInstanceID)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(line_name LIKE ? OR line_code LIKE ? OR caller_id LIKE ?)", keyword, keyword, keyword)
	}

	result, err := r.Paginate(ctx, tx.Order("id DESC"), q.Page, q.PageSize, &[]*model.Line{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.Line)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// ListAll returns all lines.
func (r *LineRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.Line, error) {
	var list []*model.Line
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "id DESC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID returns a line by ID.
func (r *LineRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.Line, error) {
	var line model.Line
	if err := r.FirstByID(ctx, tenantID, &line, id); err != nil {
		return nil, err
	}
	return &line, nil
}

// Create creates a line.
func (r *LineRepository) Create(ctx context.Context, line *model.Line) error {
	return r.BaseRepository.Create(ctx, line)
}

// Update updates a line.
func (r *LineRepository) Update(ctx context.Context, line *model.Line) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Line{}), line.TenantID)
	result := tx.Where("id = ?", line.ID).Updates(map[string]interface{}{
		"line_name":      line.LineName,
		"line_type":      line.LineType,
		"provider":       line.Provider,
		"gateway":        line.Gateway,
		"port":           line.Port,
		"username":       line.Username,
		"password":       line.Password,
		"caller_id":      line.CallerID,
		"max_channels":   line.MaxChannels,
		"status":         line.Status,
		"skill_ids":      line.SkillIDs,
		"fs_instance_id": line.FSInstanceID,
		"remark":         line.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update line failed", zap.Error(result.Error), zap.Uint64("id", line.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete deletes a line.
func (r *LineRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.Line{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete line failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// ExistsByCode checks whether a line code exists.
func (r *LineRepository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.Line{}, "line_code", code, excludeID)
}

// Count returns the total number of lines for a tenant.
func (r *LineRepository) Count(ctx context.Context, tenantID uint64) (int64, error) {
	db := r.GetDB()
	var count int64
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Line{}), tenantID)
	if err := tx.Count(&count).Error; err != nil {
		logger.ErrorCtx(ctx, "count line failed", zap.Error(err))
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}

// UpdateStatus updates a line's status.
func (r *LineRepository) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return r.BaseRepository.UpdateStatus(ctx, tenantID, &model.Line{}, id, status)
}

// UpdateRegisterStatus updates a line's registration status.
func (r *LineRepository) UpdateRegisterStatus(ctx context.Context, tenantID, id uint64, registerStatus int) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Line{}), tenantID)
	result := tx.Where("id = ?", id).Update("register_status", registerStatus)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update line register status failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateUsedChannels updates a line's used channel count.
func (r *LineRepository) UpdateUsedChannels(ctx context.Context, tenantID, id uint64, usedChannels int) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Line{}), tenantID)
	result := tx.Where("id = ?", id).Update("used_channels", usedChannels)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update line used channels failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// SumChannels returns total and used channel counts.
func (r *LineRepository) SumChannels(ctx context.Context, tenantID uint64) (total, used int64, err error) {
	db := r.GetDB()
	type result struct {
		Total int64
		Used  int64
	}
	var res result
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Line{}), tenantID)
	if err = tx.Select("COALESCE(SUM(max_channels), 0) as total, COALESCE(SUM(used_channels), 0) as used").
		Where("status = 1").
		Scan(&res).Error; err != nil {
		logger.ErrorCtx(ctx, "sum line channels failed", zap.Error(err))
		return 0, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return res.Total, res.Used, nil
}
