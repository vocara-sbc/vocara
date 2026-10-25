package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// InboundRouteRepository 呼入接待配置数据访问层
type InboundRouteRepository struct {
	BaseRepository
}

// NewInboundRouteRepository 创建呼入接待仓储
func NewInboundRouteRepository() *InboundRouteRepository {
	return &InboundRouteRepository{}
}

// InboundRoutePageQuery 呼入接待分页查询条件
type InboundRoutePageQuery struct {
	Page        int    `json:"page" query:"page"`
	PageSize    int    `json:"page_size" query:"page_size"`
	RouteName   string `json:"route_name" query:"route_name"`
	RouteCode   string `json:"route_code" query:"route_code"`
	DID         string `json:"did" query:"did"`
	QueueID     uint64 `json:"queue_id" query:"queue_id"`
	Status      *int   `json:"status" query:"status"`
	Keyword     string `json:"keyword" query:"keyword"`
}

// PageQuery 分页查询
func (r *InboundRouteRepository) PageQuery(ctx context.Context, tenantID uint64, q *InboundRoutePageQuery) ([]*model.InboundRoute, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.InboundRoute{}), tenantID)

	if q.RouteName != "" {
		tx = tx.Where("route_name LIKE ?", "%"+q.RouteName+"%")
	}
	if q.RouteCode != "" {
		tx = tx.Where("route_code LIKE ?", "%"+q.RouteCode+"%")
	}
	if q.DID != "" {
		tx = tx.Where("did LIKE ?", "%"+q.DID+"%")
	}
	if q.QueueID > 0 {
		tx = tx.Where("queue_id = ?", q.QueueID)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(route_name LIKE ? OR route_code LIKE ? OR did LIKE ?)", keyword, keyword, keyword)
	}

	result, err := r.Paginate(ctx, tx.Order("sort ASC, id DESC"), q.Page, q.PageSize, &[]*model.InboundRoute{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.InboundRoute)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// ListAll 查询所有配置
func (r *InboundRouteRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.InboundRoute, error) {
	var list []*model.InboundRoute
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "sort ASC, id DESC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 根据ID查询
func (r *InboundRouteRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.InboundRoute, error) {
	var config model.InboundRoute
	if err := r.FirstByID(ctx, tenantID, &config, id); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByDID 根据业务号码查询
func (r *InboundRouteRepository) GetByDID(ctx context.Context, tenantID uint64, number string) (*model.InboundRoute, error) {
	db := r.GetDB()
	var config model.InboundRoute
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("did = ? AND status = 1", number).First(&config).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrNotFound
		}
		logger.ErrorCtx(ctx, "get admit by business number failed", zap.Error(err), zap.String("number", number))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &config, nil
}

// ListInboundEnabled 查询所有启用且配置了业务号码的呼入接待（跨租户，internal API 用，不走租户作用域）
func (r *InboundRouteRepository) ListInboundEnabled(ctx context.Context) ([]*model.InboundRoute, error) {
	db := r.GetDB()
	var list []*model.InboundRoute
	err := db.WithContext(ctx).
		Where("status = 1 AND did IS NOT NULL AND did <> ''").
		Order("sort ASC, id ASC").
		Find(&list).Error
	if err != nil {
		logger.ErrorCtx(ctx, "list inbound enabled admits failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// ExistsDID 查业务号码是否被同租户其他启用配置占用（DID 查重）
func (r *InboundRouteRepository) ExistsDID(ctx context.Context, tenantID uint64, number string, excludeID uint64) (bool, error) {
	db := r.GetDB()
	var count int64
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.InboundRoute{}), tenantID)
	err := tx.Where("did = ? AND status = 1 AND id <> ?", number, excludeID).
		Count(&count).Error
	if err != nil {
		logger.ErrorCtx(ctx, "exists business number failed", zap.Error(err), zap.String("number", number))
		return false, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count > 0, nil
}

// Create 创建
func (r *InboundRouteRepository) Create(ctx context.Context, config *model.InboundRoute) error {
	return r.BaseRepository.Create(ctx, config)
}

// Update 更新
func (r *InboundRouteRepository) Update(ctx context.Context, config *model.InboundRoute) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.InboundRoute{}), config.TenantID)
	result := tx.Where("id = ?", config.ID).Updates(map[string]interface{}{
		"route_name":      config.RouteName,
		"did":  config.DID,
		"welcome_sound":    config.WelcomeSound,
		"wait_sound":       config.WaitSound,
		"queue_id":         config.QueueID,
		"queue_name":       config.QueueName,
		"skill_group_id":   config.SkillGroupID,
		"work_time_type":   config.WorkTimeType,
		"work_time_config": config.WorkTimeConfig,
		"holiday_ids":      config.HolidayIDs,
		"off_work_type":    config.OffWorkType,
		"off_work_value":   config.OffWorkValue,
		"off_work_sound":   config.OffWorkSound,
		"status":           config.Status,
		"sort":             config.Sort,
		"remark":           config.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update admit config failed", zap.Error(result.Error), zap.Uint64("id", config.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete 删除
func (r *InboundRouteRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.InboundRoute{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete admit config failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// ExistsByCode 检查编码是否存在
func (r *InboundRouteRepository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.InboundRoute{}, "route_code", code, excludeID)
}

// UpdateStatus 更新状态
func (r *InboundRouteRepository) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return r.BaseRepository.UpdateStatus(ctx, tenantID, &model.InboundRoute{}, id, status)
}

// ===== 节假日操作 =====

// HolidayPageQuery 节假日分页查询
type HolidayPageQuery struct {
	Page        int    `json:"page" query:"page"`
	PageSize    int    `json:"page_size" query:"page_size"`
	HolidayName string `json:"holiday_name" query:"holiday_name"`
	HolidayType *int   `json:"holiday_type" query:"holiday_type"`
	Status      *int   `json:"status" query:"status"`
}

// PageQueryHoliday 分页查询节假日
func (r *InboundRouteRepository) PageQueryHoliday(ctx context.Context, tenantID uint64, q *HolidayPageQuery) ([]*model.HolidaySchedule, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.HolidaySchedule{}), tenantID)

	if q.HolidayName != "" {
		tx = tx.Where("holiday_name LIKE ?", "%"+q.HolidayName+"%")
	}
	if q.HolidayType != nil {
		tx = tx.Where("holiday_type = ?", *q.HolidayType)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	result, err := r.Paginate(ctx, tx.Order("start_date DESC"), q.Page, q.PageSize, &[]*model.HolidaySchedule{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.HolidaySchedule)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// ListAllHolidays 查询所有节假日
func (r *InboundRouteRepository) ListAllHolidays(ctx context.Context, tenantID uint64) ([]*model.HolidaySchedule, error) {
	var list []*model.HolidaySchedule
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "start_date DESC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetHolidayByID 查询节假日详情
func (r *InboundRouteRepository) GetHolidayByID(ctx context.Context, tenantID, id uint64) (*model.HolidaySchedule, error) {
	var holiday model.HolidaySchedule
	if err := r.FirstByID(ctx, tenantID, &holiday, id); err != nil {
		return nil, err
	}
	return &holiday, nil
}

// CreateHoliday 创建节假日
func (r *InboundRouteRepository) CreateHoliday(ctx context.Context, holiday *model.HolidaySchedule) error {
	return r.BaseRepository.Create(ctx, holiday)
}

// UpdateHoliday 更新节假日
func (r *InboundRouteRepository) UpdateHoliday(ctx context.Context, holiday *model.HolidaySchedule) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.HolidaySchedule{}), holiday.TenantID)
	result := tx.Where("id = ?", holiday.ID).Updates(map[string]interface{}{
		"holiday_name": holiday.HolidayName,
		"holiday_type": holiday.HolidayType,
		"start_date":   holiday.StartDate,
		"end_date":     holiday.EndDate,
		"time_config":  holiday.TimeConfig,
		"status":       holiday.Status,
		"remark":       holiday.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update holiday failed", zap.Error(result.Error), zap.Uint64("id", holiday.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// DeleteHoliday 删除节假日
func (r *InboundRouteRepository) DeleteHoliday(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.HolidaySchedule{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete holiday failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}
