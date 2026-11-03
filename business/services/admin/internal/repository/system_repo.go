package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// SystemRepository provides system management data access.
type SystemRepository struct {
	BaseRepository
}

// NewSystemRepository creates a system management repository.
func NewSystemRepository() *SystemRepository {
	return &SystemRepository{}
}

// ===== Tenant Management =====

// TenantPageQuery contains tenant pagination filters.
type TenantPageQuery struct {
	Page       int    `json:"page" query:"page"`
	PageSize   int    `json:"page_size" query:"page_size"`
	TenantName string `json:"tenant_name" query:"tenant_name"`
	TenantCode string `json:"tenant_code" query:"tenant_code"`
	Status     *int   `json:"status" query:"status"`
	Keyword    string `json:"keyword" query:"keyword"`
}

// PageQueryTenant executes a paginated tenant query.
func (r *SystemRepository) PageQueryTenant(ctx context.Context, q *TenantPageQuery) ([]*model.Tenant, int64, error) {
	db := r.GetDB()
	tx := db.WithContext(ctx).Model(&model.Tenant{})

	if q.TenantName != "" {
		tx = tx.Where("tenant_name LIKE ?", "%"+q.TenantName+"%")
	}
	if q.TenantCode != "" {
		tx = tx.Where("tenant_code LIKE ?", "%"+q.TenantCode+"%")
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(tenant_name LIKE ? OR tenant_code LIKE ? OR contact_name LIKE ?)", keyword, keyword, keyword)
	}

	result, err := r.Paginate(ctx, tx.Order("id DESC"), q.Page, q.PageSize, &[]*model.Tenant{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.Tenant)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// GetTenantByID returns a tenant by ID.
func (r *SystemRepository) GetTenantByID(ctx context.Context, id uint64) (*model.Tenant, error) {
	db := r.GetDB()
	var tenant model.Tenant
	if err := db.WithContext(ctx).First(&tenant, id).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrNotFound
		}
		logger.ErrorCtx(ctx, "get tenant by id failed", zap.Error(err), zap.Uint64("id", id))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &tenant, nil
}

// GetTenantByCode returns a tenant by code.
func (r *SystemRepository) GetTenantByCode(ctx context.Context, code string) (*model.Tenant, error) {
	db := r.GetDB()
	var tenant model.Tenant
	if err := db.WithContext(ctx).Where("tenant_code = ?", code).First(&tenant).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrNotFound
		}
		logger.ErrorCtx(ctx, "get tenant by code failed", zap.Error(err), zap.String("code", code))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &tenant, nil
}

// CountTenant returns the total number of tenants.
func (r *SystemRepository) CountTenant(ctx context.Context) (int64, error) {
	db := r.GetDB()
	var count int64
	if err := db.WithContext(ctx).Model(&model.Tenant{}).Count(&count).Error; err != nil {
		logger.ErrorCtx(ctx, "count tenant failed", zap.Error(err))
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}

// CreateTenant creates a tenant.
func (r *SystemRepository) CreateTenant(ctx context.Context, tenant *model.Tenant) error {
	db := r.GetDB()
	if err := db.Create(ctx, tenant); err != nil {
		logger.ErrorCtx(ctx, "create tenant failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateTenant updates a tenant.
func (r *SystemRepository) UpdateTenant(ctx context.Context, tenant *model.Tenant) error {
	db := r.GetDB()
	result := db.WithContext(ctx).Model(&model.Tenant{}).Where("id = ?", tenant.ID).Updates(map[string]interface{}{
		"tenant_name":   tenant.TenantName,
		"contact_name":  tenant.ContactName,
		"contact_phone": tenant.ContactPhone,
		"contact_email": tenant.ContactEmail,
		"address":       tenant.Address,
		"status":        tenant.Status,
		"expire_time":   tenant.ExpireTime,
		"max_agents":    tenant.MaxAgents,
		"max_lines":     tenant.MaxLines,
		"remark":        tenant.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update tenant failed", zap.Error(result.Error), zap.Uint64("id", tenant.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// DeleteTenant deletes a tenant.
func (r *SystemRepository) DeleteTenant(ctx context.Context, id uint64) error {
	db := r.GetDB()
	result := db.WithContext(ctx).Where("id = ?", id).Delete(&model.Tenant{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete tenant failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// ExistsTenantByCode checks whether a tenant code exists.
func (r *SystemRepository) ExistsTenantByCode(ctx context.Context, code string, excludeID uint64) (bool, error) {
	db := r.GetDB()
	tx := db.WithContext(ctx).Model(&model.Tenant{}).Where("tenant_code = ?", code)
	if excludeID > 0 {
		tx = tx.Where("id != ?", excludeID)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		logger.ErrorCtx(ctx, "check tenant exists failed", zap.Error(err))
		return false, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count > 0, nil
}

// UpdateTenantStatus updates a tenant's status.
func (r *SystemRepository) UpdateTenantStatus(ctx context.Context, id uint64, status int) error {
	db := r.GetDB()
	result := db.WithContext(ctx).Model(&model.Tenant{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update tenant status failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// ===== FS Instance Management =====

// FSPageQuery contains FS instance pagination filters.
type FSPageQuery struct {
	Page         int    `json:"page" query:"page"`
	PageSize     int    `json:"page_size" query:"page_size"`
	InstanceName string `json:"instance_name" query:"instance_name"`
	InstanceCode string `json:"instance_code" query:"instance_code"`
	Host         string `json:"host" query:"host"`
	Status       *int   `json:"status" query:"status"`
}

// PageQueryFS executes a paginated FS instance query.
func (r *SystemRepository) PageQueryFS(ctx context.Context, q *FSPageQuery) ([]*model.FSInstance, int64, error) {
	db := r.GetDB()
	tx := db.WithContext(ctx).Model(&model.FSInstance{})

	if q.InstanceName != "" {
		tx = tx.Where("instance_code LIKE ?", "%"+q.InstanceName+"%")
	}
	if q.InstanceCode != "" {
		tx = tx.Where("instance_code LIKE ?", "%"+q.InstanceCode+"%")
	}
	if q.Host != "" {
		tx = tx.Where("ip_address LIKE ?", "%"+q.Host+"%")
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	result, err := r.Paginate(ctx, tx.Order("id DESC"), q.Page, q.PageSize, &[]*model.FSInstance{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.FSInstance)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// ListAllFS returns all FS instances.
func (r *SystemRepository) ListAllFS(ctx context.Context) ([]*model.FSInstance, error) {
	db := r.GetDB()
	var list []*model.FSInstance
	if err := db.WithContext(ctx).Order("id DESC").Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "list all fs instances failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// GetFSByID returns an FS instance by ID.
func (r *SystemRepository) GetFSByID(ctx context.Context, id uint64) (*model.FSInstance, error) {
	db := r.GetDB()
	var fs model.FSInstance
	if err := db.WithContext(ctx).First(&fs, id).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrNotFound
		}
		logger.ErrorCtx(ctx, "get fs by id failed", zap.Error(err), zap.Uint64("id", id))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &fs, nil
}

// CreateFS creates an FS instance.
func (r *SystemRepository) CreateFS(ctx context.Context, fs *model.FSInstance) error {
	db := r.GetDB()
	if err := db.Create(ctx, fs); err != nil {
		logger.ErrorCtx(ctx, "create fs instance failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateFS updates an FS instance.
func (r *SystemRepository) UpdateFS(ctx context.Context, fs *model.FSInstance) error {
	db := r.GetDB()
	result := db.WithContext(ctx).Model(&model.FSInstance{}).Where("id = ?", fs.ID).Updates(map[string]interface{}{
		"instance_code": fs.InstanceCode,
		"ip_address":    fs.Host,
		"esl_port":      fs.ESLPort,
		"esl_password":  fs.ESLPassword,
		"sip_port":      fs.SIPPort,
		"status":        fs.Status,
		"max_sessions":  fs.MaxSessions,
		"remark":        fs.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update fs instance failed", zap.Error(result.Error), zap.Uint64("id", fs.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// DeleteFS deletes an FS instance.
func (r *SystemRepository) DeleteFS(ctx context.Context, id uint64) error {
	db := r.GetDB()
	result := db.WithContext(ctx).Where("id = ?", id).Delete(&model.FSInstance{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete fs instance failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// ExistsFSByCode checks whether an FS instance code exists.
func (r *SystemRepository) ExistsFSByCode(ctx context.Context, code string, excludeID uint64) (bool, error) {
	db := r.GetDB()
	tx := db.WithContext(ctx).Model(&model.FSInstance{}).Where("instance_code = ?", code)
	if excludeID > 0 {
		tx = tx.Where("id != ?", excludeID)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		logger.ErrorCtx(ctx, "check fs exists failed", zap.Error(err))
		return false, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count > 0, nil
}

// UpdateFSStatus updates an FS instance's status.
func (r *SystemRepository) UpdateFSStatus(ctx context.Context, id uint64, status int) error {
	db := r.GetDB()
	result := db.WithContext(ctx).Model(&model.FSInstance{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update fs status failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// ===== System Configuration =====

// GetSystemConfig returns a system configuration.
func (r *SystemRepository) GetSystemConfig(ctx context.Context, tenantID uint64, key string) (*model.SystemConfig, error) {
	db := r.GetDB()
	var config model.SystemConfig
	tx := db.WithContext(ctx).Where("config_key = ?", key)
	if tenantID > 0 {
		tx = tx.Where("tenant_id IN ?", []uint64{0, tenantID})
	} else {
		tx = tx.Where("tenant_id = 0")
	}
	if err := tx.Order("tenant_id DESC").First(&config).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrNotFound
		}
		logger.ErrorCtx(ctx, "get system config failed", zap.Error(err), zap.String("key", key))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &config, nil
}

// ListSystemConfig returns system configurations.
func (r *SystemRepository) ListSystemConfig(ctx context.Context, tenantID uint64) ([]*model.SystemConfig, error) {
	db := r.GetDB()
	var list []*model.SystemConfig
	tx := db.WithContext(ctx)
	if tenantID > 0 {
		tx = tx.Where("tenant_id IN ?", []uint64{0, tenantID})
	} else {
		tx = tx.Where("tenant_id = 0")
	}
	if err := tx.Order("id ASC").Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "list system config failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// SaveSystemConfig saves a system configuration.
func (r *SystemRepository) SaveSystemConfig(ctx context.Context, config *model.SystemConfig) error {
	db := r.GetDB()
	if config.ID > 0 {
		result := db.WithContext(ctx).Model(&model.SystemConfig{}).Where("id = ?", config.ID).Updates(map[string]interface{}{
			"config_value": config.ConfigValue,
			"config_desc":  config.ConfigDesc,
		})
		if result.Error != nil {
			logger.ErrorCtx(ctx, "update system config failed", zap.Error(result.Error))
			return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
		}
		return nil
	}
	if err := db.Create(ctx, config); err != nil {
		logger.ErrorCtx(ctx, "create system config failed", zap.Error(err))
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// DeleteSystemConfig deletes a system configuration.
func (r *SystemRepository) DeleteSystemConfig(ctx context.Context, id uint64) error {
	db := r.GetDB()
	result := db.WithContext(ctx).Where("id = ? AND is_system = 0", id).Delete(&model.SystemConfig{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete system config failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// ===== Operation Logs =====

// OperationLogPageQuery contains operation log pagination filters.
type OperationLogPageQuery struct {
	Page          int    `json:"page" query:"page"`
	PageSize      int    `json:"page_size" query:"page_size"`
	Module        string `json:"module" query:"module"`
	OperationType string `json:"operation_type" query:"operation_type"`
	Username      string `json:"username" query:"username"`
	Status        *int   `json:"status" query:"status"`
	StartTime     string `json:"start_time" query:"start_time"`
	EndTime       string `json:"end_time" query:"end_time"`
}

// PageQueryOperationLog executes a paginated operation log query.
func (r *SystemRepository) PageQueryOperationLog(ctx context.Context, tenantID uint64, q *OperationLogPageQuery) ([]*model.OperationLog, int64, error) {
	db := r.GetDB()
	tx := db.WithContext(ctx).Model(&model.OperationLog{})
	if tenantID > 0 {
		tx = tx.Where("tenant_id = ?", tenantID)
	}

	if q.Module != "" {
		tx = tx.Where("module = ?", q.Module)
	}
	if q.OperationType != "" {
		tx = tx.Where("operation_type = ?", q.OperationType)
	}
	if q.Username != "" {
		tx = tx.Where("username LIKE ?", "%"+q.Username+"%")
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.StartTime != "" {
		tx = tx.Where("created_at >= ?", q.StartTime)
	}
	if q.EndTime != "" {
		tx = tx.Where("created_at <= ?", q.EndTime)
	}

	result, err := r.Paginate(ctx, tx.Order("id DESC"), q.Page, q.PageSize, &[]*model.OperationLog{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.OperationLog)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// CreateOperationLog creates an operation log.
func (r *SystemRepository) CreateOperationLog(ctx context.Context, log *model.OperationLog) error {
	db := r.GetDB()
	if err := db.Create(ctx, log); err != nil {
		// Log the error without interrupting the main flow.
		logger.ErrorCtx(ctx, "create operation log failed", zap.Error(err))
	}
	return nil
}
