package repository

import (
	"context"
	"regexp"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"

	"gorm.io/gorm"
)

// FormTemplatePageQuery contains form template pagination filters.
type FormTemplatePageQuery struct {
	Page     int    `json:"page" query:"page"`
	PageSize int    `json:"page_size" query:"page_size"`
	Keyword  string `json:"keyword" query:"keyword"`
	Category string `json:"category" query:"category"`
	Status   *int   `json:"status" query:"status"`
}

// RecordCondition contains a dynamic record query condition; Field requires whitelist and format validation.
type RecordCondition struct {
	Field string `json:"field"`
	Op    string `json:"op"` // eq-exact like-fuzzy
	Value string `json:"value"`
}

// conditionFieldPattern restricts dynamic query field names to identifiers and,
// with the template searchable whitelist, keeps external input out of JSON_EXTRACT paths.
var conditionFieldPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// ValidateConditionField validates a dynamic query field name's format.
func ValidateConditionField(name string) bool {
	return conditionFieldPattern.MatchString(name)
}

// FormTemplateRepository provides form template data access.
type FormTemplateRepository struct {
	BaseRepository
}

// NewFormTemplateRepository creates a form template repository.
func NewFormTemplateRepository() *FormTemplateRepository {
	return &FormTemplateRepository{}
}

// PageQuery executes a paginated form template query.
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

// ListAll returns all templates for record-page selection, optionally filtered by category.
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

// GetByID returns a form template by ID.
func (r *FormTemplateRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.FormTemplate, error) {
	db := r.GetDB()
	var tpl model.FormTemplate
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("id = ?", id).First(&tpl).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &tpl, nil
}

// GetByKey returns a form template by key.
func (r *FormTemplateRepository) GetByKey(ctx context.Context, tenantID uint64, key string) (*model.FormTemplate, error) {
	db := r.GetDB()
	var tpl model.FormTemplate
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("template_key = ?", key).First(&tpl).Error; err != nil {
		// Return nil, nil when no template exists; wrap other errors.
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &tpl, nil
}

// Create creates a form template.
func (r *FormTemplateRepository) Create(ctx context.Context, tpl *model.FormTemplate) error {
	return r.BaseRepository.Create(ctx, tpl)
}

// Update updates template metadata and the current version pointer without changing created_at.
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

// SetStatus updates only a template's enabled status without creating a version.
func (r *FormTemplateRepository) SetStatus(ctx context.Context, templateID uint64, status int) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormTemplate{}).
		Where("id = ?", templateID).Update("status", status).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// SetBizType updates a template's business type.
func (r *FormTemplateRepository) SetBizType(ctx context.Context, templateID uint64, bizType string) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormTemplate{}).
		Where("id = ?", templateID).Update("biz_type", bizType).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateCurrentVersion updates only the current version pointer.
func (r *FormTemplateRepository) UpdateCurrentVersion(ctx context.Context, templateID, versionID uint64) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormTemplate{}).
		Where("id = ?", templateID).Update("current_version_id", versionID).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete deletes a form template and its versions.
func (r *FormTemplateRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	// Delete versions before the template.
	if err := tx.Where("template_id = ?", id).Delete(&model.FormTemplateVersion{}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if err := tx.Where("id = ?", id).Delete(&model.FormTemplate{}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// CountByCategory counts templates in a category for reference checks before category deletion.
func (r *FormTemplateRepository) CountByCategory(ctx context.Context, category string) (int64, error) {
	var count int64
	db := r.GetDB()
	if err := db.WithContext(ctx).Model(&model.FormTemplate{}).
		Where("category = ?", category).Count(&count).Error; err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}

// CountRecords counts a template's records before deletion.
func (r *FormTemplateRepository) CountRecords(ctx context.Context, tenantID, templateID uint64) (int64, error) {
	db := r.GetDB()
	var count int64
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.FormRecord{}), tenantID)
	if err := tx.Where("template_id = ?", templateID).Count(&count).Error; err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}

// FormTemplateVersionRepository provides form template version data access.
type FormTemplateVersionRepository struct {
	BaseRepository
}

// NewFormTemplateVersionRepository creates a form template version repository.
func NewFormTemplateVersionRepository() *FormTemplateVersionRepository {
	return &FormTemplateVersionRepository{}
}

// Create creates a form template version.
func (r *FormTemplateVersionRepository) Create(ctx context.Context, v *model.FormTemplateVersion) error {
	return r.BaseRepository.Create(ctx, v)
}

// GetByID returns a form template version by primary key.
func (r *FormTemplateVersionRepository) GetByID(ctx context.Context, id uint64) (*model.FormTemplateVersion, error) {
	var v model.FormTemplateVersion
	if err := r.GetDB().WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &v, nil
}

// ListByTemplate returns all template versions in descending version order.
func (r *FormTemplateVersionRepository) ListByTemplate(ctx context.Context, templateID uint64) ([]model.FormTemplateVersion, error) {
	var list []model.FormTemplateVersion
	err := r.GetDB().WithContext(ctx).
		Where("template_id = ?", templateID).Order("version DESC").Find(&list).Error
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// MaxVersion returns a template's maximum version number, or 0 if none exists.
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

// FormRecordRepository provides form record data access.
type FormRecordRepository struct {
	BaseRepository
}

// NewFormRecordRepository creates a form record repository.
func NewFormRecordRepository() *FormRecordRepository {
	return &FormRecordRepository{}
}

// PageQuery executes a dynamic paginated record query across templates when templateKey is empty; field names are validated and JSON paths and values are parameter-bound.
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
		// Bind JSON paths and values as parameters; op accepts only eq or like, and all other values use eq.
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

// GetByID returns a form record by ID.
func (r *FormRecordRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.FormRecord, error) {
	db := r.GetDB()
	var record model.FormRecord
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("id = ?", id).First(&record).Error; err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &record, nil
}

// Create creates a form record.
func (r *FormRecordRepository) Create(ctx context.Context, record *model.FormRecord) error {
	return r.BaseRepository.Create(ctx, record)
}

// Update updates a form record's business fields without changing created_at.
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

// Delete deletes a form record.
func (r *FormRecordRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("id = ?", id).Delete(&model.FormRecord{}).Error; err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ListExport returns up to 10000 records for export using the pagination filters and querying across templates when templateKey is empty.
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

// ExistsByRecordKey checks whether a business primary key exists, excluding excludeID during updates.
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

// PageTickets executes a paginated ticket query (biz_type=ticket); status filters open or closed tickets, and keyword matches the title JSON.
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

// SetBizStatus updates a ticket's business status.
func (r *FormRecordRepository) SetBizStatus(ctx context.Context, id uint64, status string) error {
	err := r.GetDB().WithContext(ctx).Model(&model.FormRecord{}).
		Where("id = ?", id).Update("biz_status", status).Error
	if err != nil {
		return errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ListByCustomer returns all records associated with a customer.
func (r *FormRecordRepository) ListByCustomer(ctx context.Context, customerID uint64) ([]*model.FormRecord, error) {
	var list []*model.FormRecord
	err := r.GetDB().WithContext(ctx).
		Where("customer_id = ?", customerID).Order("id DESC").Find(&list).Error
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}
