package repository

import (
	"context"
	"regexp"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"

	"gorm.io/gorm"
)

// FormTemplatePageQuery 模板分页查询条件
type FormTemplatePageQuery struct {
	Page     int    `json:"page" query:"page"`
	PageSize int    `json:"page_size" query:"page_size"`
	Keyword  string `json:"keyword" query:"keyword"`
	Category string `json:"category" query:"category"`
	Status   *int   `json:"status" query:"status"`
}

// RecordCondition 实例动态查询条件（Field 必须先经白名单与格式校验）
type RecordCondition struct {
	Field string `json:"field"`
	Op    string `json:"op"` // eq-精确 like-模糊
	Value string `json:"value"`
}

// conditionFieldPattern 动态查询字段名格式防线：只允许标识符，
// 与模板 searchable 白名单配合，保证 JSON_EXTRACT 路径不受外部输入污染
var conditionFieldPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// ValidateConditionField 校验动态查询字段名格式
func ValidateConditionField(name string) bool {
	return conditionFieldPattern.MatchString(name)
}

// FormTemplateRepository 表单模板仓储
type FormTemplateRepository struct {
	BaseRepository
}

// NewFormTemplateRepository 创建表单模板仓储
func NewFormTemplateRepository() *FormTemplateRepository {
	return &FormTemplateRepository{}
}

// PageQuery 分页查询模板
func (r *FormTemplateRepository) PageQuery(ctx context.Context, tenantID uint64, q *FormTemplatePageQuery) ([]*model.FormTemplate, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormTemplate{}), tenantID)
	if q.Keyword != "" {
		tx = tx.Where("template_name LIKE ? OR template_key LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	if q.Category != "" {
		tx = tx.Where("category = ?", q.Category)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	var list []*model.FormTemplate
	page, pageSize := q.Page, q.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	if err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, total, nil
}

// ListAll 查询全部模板（实例页选模板用，可按分类过滤）
func (r *FormTemplateRepository) ListAll(ctx context.Context, tenantID uint64, category string) ([]*model.FormTemplate, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormTemplate{}), tenantID)
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	var list []*model.FormTemplate
	if err := tx.Order("id ASC").Find(&list).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// GetByID 按 ID 查询模板
func (r *FormTemplateRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.FormTemplate, error) {
	db := r.GetDB()
	var tpl model.FormTemplate
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("id = ?", id).First(&tpl).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &tpl, nil
}

// GetByKey 按标识查询模板
func (r *FormTemplateRepository) GetByKey(ctx context.Context, tenantID uint64, key string) (*model.FormTemplate, error) {
	db := r.GetDB()
	var tpl model.FormTemplate
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("template_key = ?", key).First(&tpl).Error; err != nil {
		// 未找到返回 nil,nil（语义：无此模板）；其他错误才包装
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &tpl, nil
}

// Create 创建模板
func (r *FormTemplateRepository) Create(ctx context.Context, tpl *model.FormTemplate) error {
	return r.BaseRepository.Create(ctx, tpl)
}

// Update 更新模板元数据与当前版本指针（按字段更新，不触碰 created_at）
func (r *FormTemplateRepository) Update(ctx context.Context, tpl *model.FormTemplate) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormTemplate{}), tpl.TenantID)
	if err := tx.Where("id = ?", tpl.ID).Updates(map[string]interface{}{
		"template_key":       tpl.TemplateKey,
		"template_name":      tpl.TemplateName,
		"category":           tpl.Category,
		"current_version_id": tpl.CurrentVersionID,
		"status":             tpl.Status,
		"remark":             tpl.Remark,
	}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// SetStatus 仅更新模板启停状态（不产生新版本）
func (r *FormTemplateRepository) SetStatus(ctx context.Context, templateID uint64, status int) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormTemplate{}).
		Where("id = ?", templateID).Update("status", status).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// SetBizType 更新模板业务类型
func (r *FormTemplateRepository) SetBizType(ctx context.Context, templateID uint64, bizType string) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormTemplate{}).
		Where("id = ?", templateID).Update("biz_type", bizType).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateCurrentVersion 仅更新当前版本指针
func (r *FormTemplateRepository) UpdateCurrentVersion(ctx context.Context, templateID, versionID uint64) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormTemplate{}).
		Where("id = ?", templateID).Update("current_version_id", versionID).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete 删除模板并级联清理其版本
func (r *FormTemplateRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	// 先删版本，再删主表
	if err := tx.Where("template_id = ?", id).Delete(&model.FormTemplateVersion{}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if err := tx.Where("id = ?", id).Delete(&model.FormTemplate{}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// CountByCategory 统计分类下的模板数量（删除分类前引用检查）
func (r *FormTemplateRepository) CountByCategory(ctx context.Context, category string) (int64, error) {
	var count int64
	db := r.GetDB()
	if err := db.WithContext(ctx).Model(&model.FormTemplate{}).
		Where("category = ?", category).Count(&count).Error; err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}

// CountRecords 统计模板下的实例数（删除前检查）
func (r *FormTemplateRepository) CountRecords(ctx context.Context, tenantID, templateID uint64) (int64, error) {
	db := r.GetDB()
	var count int64
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormRecord{}), tenantID)
	if err := tx.Where("template_id = ?", templateID).Count(&count).Error; err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}

// FormTemplateVersionRepository 模板版本仓储
type FormTemplateVersionRepository struct {
	BaseRepository
}

// NewFormTemplateVersionRepository 创建模板版本仓储
func NewFormTemplateVersionRepository() *FormTemplateVersionRepository {
	return &FormTemplateVersionRepository{}
}

// Create 写入版本
func (r *FormTemplateVersionRepository) Create(ctx context.Context, v *model.FormTemplateVersion) error {
	return r.BaseRepository.Create(ctx, v)
}

// GetByID 按主键取版本
func (r *FormTemplateVersionRepository) GetByID(ctx context.Context, id uint64) (*model.FormTemplateVersion, error) {
	var v model.FormTemplateVersion
	if err := r.GetDB().WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &v, nil
}

// ListByTemplate 列出模板全部版本（版本号倒序）
func (r *FormTemplateVersionRepository) ListByTemplate(ctx context.Context, templateID uint64) ([]model.FormTemplateVersion, error) {
	var list []model.FormTemplateVersion
	err := r.GetDB().WithContext(ctx).
		Where("template_id = ?", templateID).Order("version DESC").Find(&list).Error
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// MaxVersion 取模板当前最大版本号（无版本返回 0）
func (r *FormTemplateVersionRepository) MaxVersion(ctx context.Context, templateID uint64) (int, error) {
	var maxVersion int
	err := r.GetDB().WithContext(ctx).Model(&model.FormTemplateVersion{}).
		Where("template_id = ?", templateID).
		Select("COALESCE(MAX(version),0)").Scan(&maxVersion).Error
	if err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return maxVersion, nil
}

// FormRecordRepository 表单实例仓储
type FormRecordRepository struct {
	BaseRepository
}

// NewFormRecordRepository 创建表单实例仓储
func NewFormRecordRepository() *FormRecordRepository {
	return &FormRecordRepository{}
}

// PageQuery 实例动态分页查询（templateKey 为空时跨模板查询；动态条件内联拼装：字段名过格式防线，JSON 路径与值全部参数绑定）
func (r *FormRecordRepository) PageQuery(ctx context.Context, tenantID uint64, templateKey string, conds []RecordCondition, page, pageSize int) ([]*model.FormRecord, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormRecord{}), tenantID)
	if templateKey != "" {
		tx = tx.Where("template_key = ?", templateKey)
	}
	for _, cond := range conds {
		if !ValidateConditionField(cond.Field) || cond.Value == "" {
			continue
		}
		// JSON 路径与值均参数绑定；op 只认 eq/like，其他一律按 eq
		if cond.Op == "like" {
			tx = tx.Where("JSON_UNQUOTE(JSON_EXTRACT(`data`, ?)) LIKE ?", "$."+cond.Field, "%"+cond.Value+"%")
		} else {
			tx = tx.Where("JSON_UNQUOTE(JSON_EXTRACT(`data`, ?)) = ?", "$."+cond.Field, cond.Value)
		}
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
	var list []*model.FormRecord
	if err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, total, nil
}

// GetByID 按 ID 查询实例
func (r *FormRecordRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.FormRecord, error) {
	db := r.GetDB()
	var record model.FormRecord
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("id = ?", id).First(&record).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &record, nil
}

// Create 创建实例
func (r *FormRecordRepository) Create(ctx context.Context, record *model.FormRecord) error {
	return r.BaseRepository.Create(ctx, record)
}

// Update 更新实例（按业务字段更新，不触碰 created_at）
func (r *FormRecordRepository) Update(ctx context.Context, record *model.FormRecord) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormRecord{}), record.TenantID)
	if err := tx.Where("id = ?", record.ID).Updates(map[string]interface{}{
		"template_id":         record.TemplateID,
		"template_version_id": record.TemplateVersionID,
		"template_key":        record.TemplateKey,
		"record_key":          record.RecordKey,
		"data":                record.Data,
		"field_snapshot":      record.FieldSnapshot,
	}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete 删除实例
func (r *FormRecordRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("id = ?", id).Delete(&model.FormRecord{}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ListExport 导出用全量查询（与分页同条件，templateKey 为空时跨模板查询，上限 10000 条）
func (r *FormRecordRepository) ListExport(ctx context.Context, tenantID uint64, templateKey string, conds []RecordCondition) ([]*model.FormRecord, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormRecord{}), tenantID)
	if templateKey != "" {
		tx = tx.Where("template_key = ?", templateKey)
	}
	for _, cond := range conds {
		if !ValidateConditionField(cond.Field) || cond.Value == "" {
			continue
		}
		if cond.Op == "like" {
			tx = tx.Where("JSON_UNQUOTE(JSON_EXTRACT(`data`, ?)) LIKE ?", "$."+cond.Field, "%"+cond.Value+"%")
		} else {
			tx = tx.Where("JSON_UNQUOTE(JSON_EXTRACT(`data`, ?)) = ?", "$."+cond.Field, cond.Value)
		}
	}
	var list []*model.FormRecord
	if err := tx.Order("id DESC").Limit(10000).Find(&list).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// ExistsByRecordKey 检查业务主键是否已存在（excludeID 用于更新时排除自身）
func (r *FormRecordRepository) ExistsByRecordKey(ctx context.Context, tenantID uint64, templateKey, recordKey string, excludeID uint64) (bool, error) {
	if recordKey == "" {
		return false, nil
	}
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormRecord{}), tenantID).
		Where("template_key = ? AND record_key = ?", templateKey, recordKey)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return false, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count > 0, nil
}

// PageTickets 工单分页（biz_type=ticket）：status 过滤 open/closed/全部；keyword 匹配标题 JSON
func (r *FormRecordRepository) PageTickets(ctx context.Context, tenantID uint64, status, keyword string, page, pageSize int) ([]*model.FormRecord, int64, error) {
	tx := r.GetDB().WithContext(ctx).Model(&model.FormRecord{}).
		Where("tenant_id = ? AND biz_type = ?", tenantID, model.BizTypeTicket)
	if status == model.TicketStatusOpen || status == model.TicketStatusClosed {
		tx = tx.Where("biz_status = ?", status)
	}
	if keyword != "" {
		tx = tx.Where("JSON_UNQUOTE(JSON_EXTRACT(`data`, '$.ticket_title')) LIKE ?", "%"+keyword+"%")
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
	var list []*model.FormRecord
	if err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, total, nil
}

// SetBizStatus 更新工单业务状态
func (r *FormRecordRepository) SetBizStatus(ctx context.Context, id uint64, status string) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormRecord{}).
		Where("id = ?", id).Update("biz_status", status).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ListByCustomer 列出某客户关联的全部记录（按类型/时间）
func (r *FormRecordRepository) ListByCustomer(ctx context.Context, customerID uint64) ([]*model.FormRecord, error) {
	var list []*model.FormRecord
	err := r.GetDB().WithContext(ctx).
		Where("customer_id = ?", customerID).Order("id DESC").Find(&list).Error
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}
