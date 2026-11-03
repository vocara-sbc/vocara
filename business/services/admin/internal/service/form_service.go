package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// FormService provides dynamic form operations.
type FormService struct {
	templateRepo *repository.FormTemplateRepository
	recordRepo   *repository.FormRecordRepository
	versionRepo  *repository.FormTemplateVersionRepository
	customerRepo *repository.CustomerRepository
	categorySvc  *FormCategoryService
}

// NewFormService creates a dynamic form service.
func NewFormService() *FormService {
	return &FormService{
		templateRepo: repository.NewFormTemplateRepository(),
		recordRepo:   repository.NewFormRecordRepository(),
		versionRepo:  repository.NewFormTemplateVersionRepository(),
		customerRepo: repository.NewCustomerRepository(),
		categorySvc:  NewFormCategoryService(),
	}
}

// currentVersionSchema returns a template's current version schema.
func (s *FormService) currentVersionSchema(ctx context.Context, tpl *model.FormTemplate) (*FormSchema, error) {
	ver, err := s.versionRepo.GetByID(ctx, tpl.CurrentVersionID)
	if err != nil {
		return nil, err
	}
	return ParseFormSchema(ver.Schema)
}

// recordVersion returns a record's bound version or the template's current version when no version is bound.
func (s *FormService) recordVersion(ctx context.Context, tpl *model.FormTemplate, versionID uint64) (*model.FormTemplateVersion, error) {
	if versionID > 0 {
		return s.versionRepo.GetByID(ctx, versionID)
	}
	return s.versionRepo.GetByID(ctx, tpl.CurrentVersionID)
}

// normalizeCategory defaults an empty category to general and requires its code to exist, including disabled categories.
func (s *FormService) normalizeCategory(ctx context.Context, category string) (string, error) {
	if category == "" {
		category = "general"
	}
	exists, err := s.categorySvc.ExistsCode(ctx, category)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.NewBizError(errors.CodeParamError, "表单分类不存在："+category)
	}
	return category, nil
}

// ===== Templates =====

// FormTemplateSaveDTO contains form template creation parameters.
type FormTemplateSaveDTO struct {
	TemplateKey  string `json:"template_key" binding:"required"`
	TemplateName string `json:"template_name" binding:"required"`
	Category     string `json:"category"`
	Schema       string `json:"schema" binding:"required"`
	Remark       string `json:"remark"`
}

// FormTemplateUpdateDTO contains form template update parameters.
type FormTemplateUpdateDTO struct {
	ID           uint64 `json:"id" binding:"required"`
	TemplateKey  string `json:"template_key" binding:"required"`
	TemplateName string `json:"template_name" binding:"required"`
	Category     string `json:"category"`
	Schema       string `json:"schema" binding:"required"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

// TemplatePage executes a paginated form template query.
func (s *FormService) TemplatePage(ctx context.Context, tenantID uint64, q *repository.FormTemplatePageQuery) ([]*model.FormTemplate, int64, error) {
	// Normalize and write back pagination parameters because fields with only json tags do not bind GET queries.
	// The handler uses the normalized page and page_size values to build the pagination response.
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.templateRepo.PageQuery(ctx, tenantID, q)
}

// TemplateList returns all form templates.
func (s *FormService) TemplateList(ctx context.Context, tenantID uint64, category string) ([]*model.FormTemplate, error) {
	return s.templateRepo.ListAll(ctx, tenantID, category)
}

// TemplateGet returns form template details.
func (s *FormService) TemplateGet(ctx context.Context, tenantID, id uint64) (*model.FormTemplate, error) {
	return s.templateRepo.GetByID(ctx, tenantID, id)
}

// TemplateCreate creates a template and its v1 schema without a draft.
func (s *FormService) TemplateCreate(ctx context.Context, tenantID uint64, dto *FormTemplateSaveDTO) (*model.FormTemplate, error) {
	if _, err := ParseFormSchema(dto.Schema); err != nil {
		return nil, err
	}
	category, err := s.normalizeCategory(ctx, dto.Category)
	if err != nil {
		return nil, err
	}
	dto.Category = category
	if existing, _ := s.templateRepo.GetByKey(ctx, tenantID, dto.TemplateKey); existing != nil && existing.ID > 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "模板标识已存在")
	}

	tpl := &model.FormTemplate{
		TemplateKey:  dto.TemplateKey,
		TemplateName: dto.TemplateName,
		Category:     dto.Category,
		Status:       1,
		Remark:       dto.Remark,
	}
	tpl.TenantID = tenantID
	if err := s.templateRepo.Create(ctx, tpl); err != nil {
		return nil, err
	}

	// Save v1 and set the template's current version pointer.
	ver := &model.FormTemplateVersion{
		TenantID:   tenantID,
		TemplateID: tpl.ID,
		Version:    1,
		Schema:     dto.Schema,
	}
	if err := s.versionRepo.Create(ctx, ver); err != nil {
		return nil, err
	}
	if err := s.templateRepo.UpdateCurrentVersion(ctx, tpl.ID, ver.ID); err != nil {
		return nil, err
	}
	tpl.CurrentVersionID = ver.ID
	return tpl, nil
}

// TemplateUpdate creates an immutable version and updates template metadata without a draft.
func (s *FormService) TemplateUpdate(ctx context.Context, tenantID uint64, dto *FormTemplateUpdateDTO) error {
	if _, err := ParseFormSchema(dto.Schema); err != nil {
		return err
	}
	tpl, err := s.templateRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil || tpl == nil || tpl.ID == 0 {
		return errors.NewBizError(errors.CodeParamError, "模板不存在")
	}
	if tpl.TemplateKey != dto.TemplateKey {
		if existing, _ := s.templateRepo.GetByKey(ctx, tenantID, dto.TemplateKey); existing != nil && existing.ID > 0 {
			return errors.NewBizError(errors.CodeParamError, "模板标识已存在")
		}
	}
	category, err := s.normalizeCategory(ctx, dto.Category)
	if err != nil {
		return err
	}

	// The server assigns the next version number as the current maximum plus one.
	nextVersion, err := s.versionRepo.MaxVersion(ctx, tpl.ID)
	if err != nil {
		return err
	}
	ver := &model.FormTemplateVersion{
		TenantID:   tenantID,
		TemplateID: tpl.ID,
		Version:    nextVersion + 1,
		Schema:     dto.Schema,
	}
	if err := s.versionRepo.Create(ctx, ver); err != nil {
		return err
	}

	// Update template metadata and the current version pointer.
	tpl.TemplateKey = dto.TemplateKey
	tpl.TemplateName = dto.TemplateName
	tpl.Category = category
	tpl.Status = dto.Status
	tpl.Remark = dto.Remark
	tpl.CurrentVersionID = ver.ID
	return s.templateRepo.Update(ctx, tpl)
}

// UpdateTemplateStatus updates only the enabled status without creating a version.
func (s *FormService) UpdateTemplateStatus(ctx context.Context, templateID uint64, status int) error {
	return s.templateRepo.SetStatus(ctx, templateID, status)
}

// TemplateVersionList returns template versions in descending version order.
func (s *FormService) TemplateVersionList(ctx context.Context, templateID uint64) ([]model.FormTemplateVersion, error) {
	return s.versionRepo.ListByTemplate(ctx, templateID)
}

// TemplateVersionGet returns a complete template version.
func (s *FormService) TemplateVersionGet(ctx context.Context, versionID uint64) (*model.FormTemplateVersion, error) {
	return s.versionRepo.GetByID(ctx, versionID)
}

// GetSchemaByKey returns a template's current version schema by key.
func (s *FormService) GetSchemaByKey(ctx context.Context, tenantID uint64, key string) (*model.FormTemplateVersion, error) {
	tpl, err := s.templateRepo.GetByKey(ctx, tenantID, key)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, errors.NewBizError(errors.CodeParamError, "模板不存在："+key)
	}
	return s.versionRepo.GetByID(ctx, tpl.CurrentVersionID)
}

// EnsureTemplateVersions populates missing template version data idempotently:
// it creates v1 from the template schema, sets current_version_id to the latest version when zero,
// and binds records with a zero template_version_id to their template's v1.
func (s *FormService) EnsureTemplateVersions(ctx context.Context) error {
	db := s.templateRepo.GetDB()

	var tpls []model.FormTemplate
	if err := db.WithContext(ctx).Find(&tpls).Error; err != nil {
		return err
	}
	for i := range tpls {
		t := &tpls[i]
		versions, err := s.versionRepo.ListByTemplate(ctx, t.ID)
		if err != nil {
			return err
		}
		// When no version exists, read the template schema column, which AutoMigrate retains, and create v1.
		if len(versions) == 0 {
			var oldSchema string
			if err := db.WithContext(ctx).
				Raw("SELECT `schema` FROM tb_form_template WHERE id = ?", t.ID).
				Scan(&oldSchema).Error; err != nil {
				return err
			}
			if oldSchema == "" {
				return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("模板 %d 缺少 schema，无法补建版本", t.ID))
			}
			v1 := &model.FormTemplateVersion{
				TenantID:   t.TenantID,
				TemplateID: t.ID,
				Version:    1,
				Schema:     oldSchema,
			}
			if err := s.versionRepo.Create(ctx, v1); err != nil {
				return err
			}
			versions = []model.FormTemplateVersion{*v1}
		}
		// Set a zero current version pointer to the latest version; ListByTemplate returns descending version order.
		if t.CurrentVersionID == 0 {
			if err := s.templateRepo.UpdateCurrentVersion(ctx, t.ID, versions[0].ID); err != nil {
				return err
			}
		}
	}

	// Populate each record's template_version_id from its template's v1.
	return db.WithContext(ctx).Exec(`
		UPDATE tb_form_record r
		JOIN tb_form_template_version v
		  ON v.template_id = r.template_id AND v.version = 1
		SET r.template_version_id = v.id
		WHERE r.template_version_id = 0`).Error
}

// TemplateDelete deletes a template that has no records.
func (s *FormService) TemplateDelete(ctx context.Context, tenantID, id uint64) error {
	count, err := s.templateRepo.CountRecords(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("该模板下已有%d条实例数据，请先清理", count))
	}
	return s.templateRepo.Delete(ctx, tenantID, id)
}

// ===== Records =====

// normalizeRecordKey maps an empty business primary key to nil for database NULL and returns a pointer otherwise.
func normalizeRecordKey(s string) *string {
	if s == "" {
		return nil
	}
	v := s
	return &v
}

// FormRecordSaveDTO contains form record creation parameters.
type FormRecordSaveDTO struct {
	TemplateKey string                 `json:"template_key" binding:"required"`
	RecordKey   string                 `json:"record_key"`
	Data        map[string]interface{} `json:"data" binding:"required"`
	// Call context supplied by workstation C; empty for manual entry.
	CallID    string `json:"call_id"`
	Direction string `json:"direction"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
}

// FormRecordUpdateDTO contains form record update parameters.
type FormRecordUpdateDTO struct {
	ID        uint64                 `json:"id" binding:"required"`
	RecordKey string                 `json:"record_key"`
	Data      map[string]interface{} `json:"data" binding:"required"`
}

// RecordPage executes a dynamic paginated record query across templates when templateKey is empty, with whitelist-filtered conditions.
func (s *FormService) RecordPage(ctx context.Context, tenantID uint64, templateKey string, conds []repository.RecordCondition, page, pageSize int) ([]*model.FormRecord, int64, error) {
	allowed, err := s.searchableWhitelist(ctx, tenantID, templateKey)
	if err != nil {
		return nil, 0, err
	}
	filtered, err := filterRecordConditions(conds, allowed)
	if err != nil {
		return nil, 0, err
	}
	return s.recordRepo.PageQuery(ctx, tenantID, templateKey, filtered, page, pageSize)
}

// filterRecordConditions filters dynamic conditions by whitelist and treats every operator other than like as eq.
func filterRecordConditions(conds []repository.RecordCondition, allowed map[string]bool) ([]repository.RecordCondition, error) {
	var filtered []repository.RecordCondition
	for _, cond := range conds {
		// Reject query fields outside the whitelist so configuration errors reach the frontend.
		if !allowed[cond.Field] {
			return nil, errors.NewBizError(errors.CodeParamError, fmt.Sprintf("字段 %s 不可搜索", cond.Field))
		}
		if cond.Op != "like" {
			cond.Op = "eq"
		}
		filtered = append(filtered, cond)
	}
	return filtered, nil
}

// searchableWhitelist returns searchable fields, taking the union across all templates when templateKey is empty.
func (s *FormService) searchableWhitelist(ctx context.Context, tenantID uint64, templateKey string) (map[string]bool, error) {
	templates, err := s.templateRepo.ListAll(ctx, tenantID, "")
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, tpl := range templates {
		if templateKey != "" && tpl.TemplateKey != templateKey {
			continue
		}
		schema, err := s.currentVersionSchema(ctx, tpl)
		if err != nil {
			return nil, err
		}
		for field := range SearchableFieldSet(schema) {
			allowed[field] = true
		}
	}
	// Return a parameter error when a specified template is not found and the whitelist is empty.
	if templateKey != "" && len(allowed) == 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "模板不存在")
	}
	return allowed, nil
}

// tableFieldsUnion returns the union of current-version table fields, deduplicated by name with labels from the first template.
func (s *FormService) tableFieldsUnion(ctx context.Context, templates []*model.FormTemplate) ([]FormField, error) {
	seen := map[string]bool{}
	var fields []FormField
	for _, tpl := range templates {
		schema, err := s.currentVersionSchema(ctx, tpl)
		if err != nil {
			return nil, err
		}
		walkFields(schema.Fields, func(f FormField) bool {
			if f.ShowInTable && !layoutFieldTypes[f.Type] && !seen[f.Name] {
				seen[f.Name] = true
				fields = append(fields, f)
			}
			return true
		})
	}
	return fields, nil
}

// RecordGet returns form record details.
func (s *FormService) RecordGet(ctx context.Context, tenantID, id uint64) (*model.FormRecord, error) {
	return s.recordRepo.GetByID(ctx, tenantID, id)
}

// RecordCreate validates data, generates a snapshot, checks record_key uniqueness, and creates a record.
func (s *FormService) RecordCreate(ctx context.Context, tenantID uint64, dto *FormRecordSaveDTO) (*model.FormRecord, error) {
	tpl, err := s.templateRepo.GetByKey(ctx, tenantID, dto.TemplateKey)
	if err != nil || tpl == nil || tpl.ID == 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "模板不存在")
	}
	if tpl.Status != 1 {
		return nil, errors.NewBizError(errors.CodeParamError, "模板已停用，不能填报新数据")
	}
	// Validate against the template's current version and bind the record to it.
	schema, err := s.currentVersionSchema(ctx, tpl)
	if err != nil {
		return nil, err
	}
	if err := ValidateRecordData(schema, dto.Data); err != nil {
		return nil, err
	}
	if dto.RecordKey != "" {
		exists, err := s.recordRepo.ExistsByRecordKey(ctx, tenantID, dto.TemplateKey, dto.RecordKey, 0)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.NewBizError(errors.CodeParamError, "业务主键已存在："+dto.RecordKey)
		}
	}
	dataJSON, _ := json.Marshal(dto.Data)
	snapshotJSON, _ := json.Marshal(FlattenSnapshot(schema))
	now := time.Now()
	record := &model.FormRecord{
		TemplateID:        tpl.ID,
		TemplateVersionID: tpl.CurrentVersionID,
		TemplateKey:       tpl.TemplateKey,
		BizType:           tpl.BizType,
		CallID:            dto.CallID,
		Direction:         dto.Direction,
		AgentID:           dto.AgentID,
		AgentName:         dto.AgentName,
		RecordKey:         normalizeRecordKey(dto.RecordKey),
		Data:              string(dataJSON),
		FieldSnapshot:     string(snapshotJSON),
	}
	record.TenantID = tenantID

	// Associate the record with a customer profile or ticket.
	switch tpl.BizType {
	case model.BizTypeCustomer:
		customerID, err := s.upsertCustomerFromForm(ctx, tenantID, dto.Data, now)
		if err != nil {
			return nil, err
		}
		record.CustomerID = customerID
	case model.BizTypeTicket:
		customerID, err := s.ensureTicketCustomer(ctx, tenantID, dto.Data, now)
		if err != nil {
			return nil, err
		}
		record.CustomerID = customerID
		record.BizStatus = model.TicketStatusOpen
	}

	if err := s.recordRepo.Create(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

// RecordUpdate revalidates data against the record's bound version and regenerates its snapshot without changing the binding.
func (s *FormService) RecordUpdate(ctx context.Context, tenantID uint64, dto *FormRecordUpdateDTO) error {
	record, err := s.recordRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil || record == nil || record.ID == 0 {
		return errors.NewBizError(errors.CodeParamError, "实例不存在")
	}
	tpl, err := s.templateRepo.GetByKey(ctx, tenantID, record.TemplateKey)
	if err != nil || tpl == nil || tpl.ID == 0 {
		return errors.NewBizError(errors.CodeParamError, "模板不存在")
	}
	// Use the record's bound version or the current version when no version is bound.
	ver, err := s.recordVersion(ctx, tpl, record.TemplateVersionID)
	if err != nil {
		return err
	}
	schema, err := ParseFormSchema(ver.Schema)
	if err != nil {
		return err
	}
	if err := ValidateRecordData(schema, dto.Data); err != nil {
		return err
	}
	if dto.RecordKey != "" {
		exists, err := s.recordRepo.ExistsByRecordKey(ctx, tenantID, record.TemplateKey, dto.RecordKey, record.ID)
		if err != nil {
			return err
		}
		if exists {
			return errors.NewBizError(errors.CodeParamError, "业务主键已存在："+dto.RecordKey)
		}
	}
	dataJSON, _ := json.Marshal(dto.Data)
	snapshotJSON, _ := json.Marshal(FlattenSnapshot(schema))
	record.RecordKey = normalizeRecordKey(dto.RecordKey)
	record.Data = string(dataJSON)
	record.FieldSnapshot = string(snapshotJSON)
	return s.recordRepo.Update(ctx, record)
}

// RecordDelete deletes a form record.
func (s *FormService) RecordDelete(ctx context.Context, tenantID, id uint64) error {
	return s.recordRepo.Delete(ctx, tenantID, id)
}

// RecordExport exports CSV using each record's own snapshot for headings and translated values.
// An empty templateKey exports across templates with the template in the first column and the union of all table fields as dynamic columns.
func (s *FormService) RecordExport(ctx context.Context, tenantID uint64, templateKey string, conds []repository.RecordCondition, writer io.Writer) error {
	allowed, err := s.searchableWhitelist(ctx, tenantID, templateKey)
	if err != nil {
		return err
	}
	filtered, err := filterRecordConditions(conds, allowed)
	if err != nil {
		return err
	}
	list, err := s.recordRepo.ListExport(ctx, tenantID, templateKey, filtered)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	templates, err := s.templateRepo.ListAll(ctx, tenantID, "")
	if err != nil {
		return err
	}
	var scopedTemplates []*model.FormTemplate
	templateNames := map[string]string{}
	for _, tpl := range templates {
		templateNames[tpl.TemplateKey] = tpl.TemplateName
		if templateKey == "" || tpl.TemplateKey == templateKey {
			scopedTemplates = append(scopedTemplates, tpl)
		}
	}
	if templateKey != "" && len(scopedTemplates) == 0 {
		return errors.NewBizError(errors.CodeParamError, "模板不存在")
	}

	// Cross-template exports include the template in the first column; single-template exports omit it.
	exportFields, err := s.tableFieldsUnion(ctx, scopedTemplates)
	if err != nil {
		return err
	}
	var headers []string
	if templateKey == "" {
		headers = append(headers, "所属模板")
	}
	for _, f := range exportFields {
		headers = append(headers, f.Label)
	}
	headers = append(headers, "业务主键", "创建时间", "更新时间")
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, record := range list {
		var data map[string]interface{}
		_ = json.Unmarshal([]byte(record.Data), &data)
		var snapshot []FieldSnapshotItem
		_ = json.Unmarshal([]byte(record.FieldSnapshot), &snapshot)
		snapshotMap := map[string]FieldSnapshotItem{}
		for _, item := range snapshot {
			snapshotMap[item.Name] = item
		}
		var row []string
		if templateKey == "" {
			row = append(row, templateNames[record.TemplateKey])
		}
		for _, f := range exportFields {
			row = append(row, translateExportValue(data[f.Name], snapshotMap[f.Name]))
		}
		recordKey := ""
		if record.RecordKey != nil {
			recordKey = *record.RecordKey
		}
		row = append(row, recordKey, record.CreatedAt.Format("2006-01-02 15:04:05"), record.UpdatedAt.Format("2006-01-02 15:04:05"))
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// translateExportValue converts option values to labels and joins multiple selections with Chinese enumeration commas.
func translateExportValue(value interface{}, snap FieldSnapshotItem) string {
	if value == nil {
		return ""
	}
	translate := func(v interface{}) string {
		for _, opt := range snap.Options {
			if fmt.Sprintf("%v", opt.Value) == fmt.Sprintf("%v", v) {
				return opt.Label
			}
		}
		return fmt.Sprintf("%v", v)
	}
	if arr, ok := value.([]interface{}); ok {
		var parts []string
		for _, v := range arr {
			parts = append(parts, translate(v))
		}
		return strings.Join(parts, "、")
	}
	return translate(value)
}
