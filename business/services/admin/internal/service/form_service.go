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

// FormService 动态表单服务
type FormService struct {
	templateRepo *repository.FormTemplateRepository
	recordRepo   *repository.FormRecordRepository
	versionRepo  *repository.FormTemplateVersionRepository
	customerRepo *repository.CustomerRepository
	categorySvc  *FormCategoryService
}

// NewFormService 创建动态表单服务
func NewFormService() *FormService {
	return &FormService{
		templateRepo: repository.NewFormTemplateRepository(),
		recordRepo:   repository.NewFormRecordRepository(),
		versionRepo:  repository.NewFormTemplateVersionRepository(),
		customerRepo: repository.NewCustomerRepository(),
		categorySvc:  NewFormCategoryService(),
	}
}

// currentVersionSchema 取模板当前版本 schema
func (s *FormService) currentVersionSchema(ctx context.Context, tpl *model.FormTemplate) (*FormSchema, error) {
	ver, err := s.versionRepo.GetByID(ctx, tpl.CurrentVersionID)
	if err != nil {
		return nil, err
	}
	return ParseFormSchema(ver.Schema)
}

// recordVersion 取记录绑定版本；未回填（存量）时回退模板当前版本
func (s *FormService) recordVersion(ctx context.Context, tpl *model.FormTemplate, versionID uint64) (*model.FormTemplateVersion, error) {
	if versionID > 0 {
		return s.versionRepo.GetByID(ctx, versionID)
	}
	return s.versionRepo.GetByID(ctx, tpl.CurrentVersionID)
}

// normalizeCategory 归一分类：空值取 general，编码须存在（含停用项）
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

// ===== 模板 =====

// FormTemplateSaveDTO 模板保存参数
type FormTemplateSaveDTO struct {
	TemplateKey  string `json:"template_key" binding:"required"`
	TemplateName string `json:"template_name" binding:"required"`
	Category     string `json:"category"`
	Schema       string `json:"schema" binding:"required"`
	Remark       string `json:"remark"`
}

// FormTemplateUpdateDTO 模板更新参数
type FormTemplateUpdateDTO struct {
	ID           uint64 `json:"id" binding:"required"`
	TemplateKey  string `json:"template_key" binding:"required"`
	TemplateName string `json:"template_name" binding:"required"`
	Category     string `json:"category"`
	Schema       string `json:"schema" binding:"required"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

// TemplatePage 模板分页
func (s *FormService) TemplatePage(ctx context.Context, tenantID uint64, q *repository.FormTemplatePageQuery) ([]*model.FormTemplate, int64, error) {
	// 分页参数归一化并回写：GET 查询只有 json tag 的字段不参与 query 绑定，
	// handler 需要用回写后的 page/page_size 组装分页响应
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.templateRepo.PageQuery(ctx, tenantID, q)
}

// TemplateList 模板全量
func (s *FormService) TemplateList(ctx context.Context, tenantID uint64, category string) ([]*model.FormTemplate, error) {
	return s.templateRepo.ListAll(ctx, tenantID, category)
}

// TemplateGet 模板详情
func (s *FormService) TemplateGet(ctx context.Context, tenantID, id uint64) (*model.FormTemplate, error) {
	return s.templateRepo.GetByID(ctx, tenantID, id)
}

// TemplateCreate 创建模板：主表 + v1（无草稿）
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

	// v1 落库并回填主表当前版本指针
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

// TemplateUpdate 编辑保存：产生不可变新版本并更新主表元数据（无草稿）
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

	// 新版本号 = 当前最大 +1（服务端生成）
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

	// 主表元数据与当前版本指针更新
	tpl.TemplateKey = dto.TemplateKey
	tpl.TemplateName = dto.TemplateName
	tpl.Category = category
	tpl.Status = dto.Status
	tpl.Remark = dto.Remark
	tpl.CurrentVersionID = ver.ID
	return s.templateRepo.Update(ctx, tpl)
}

// UpdateTemplateStatus 仅更新启停状态（不产生新版本）
func (s *FormService) UpdateTemplateStatus(ctx context.Context, templateID uint64, status int) error {
	return s.templateRepo.SetStatus(ctx, templateID, status)
}

// TemplateVersionList 模板历史版本列表（版本号倒序）
func (s *FormService) TemplateVersionList(ctx context.Context, templateID uint64) ([]model.FormTemplateVersion, error) {
	return s.versionRepo.ListByTemplate(ctx, templateID)
}

// TemplateVersionGet 单个版本完整内容
func (s *FormService) TemplateVersionGet(ctx context.Context, versionID uint64) (*model.FormTemplateVersion, error) {
	return s.versionRepo.GetByID(ctx, versionID)
}

// GetSchemaByKey 按模板标识取其当前版本内容（schema）
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

// EnsureTemplateVersions 存量库一次性迁移（幂等）：
// 为无版本的旧模板按旧 schema 补 v1；主表 current_version_id 为 0 时回填最新版本；
// 记录 template_version_id 为 0 时按所属模板 v1 回填。
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
		// 无版本：旧库主表仍含 schema 列（AutoMigrate 不删列），读取后补 v1
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
		// 主表当前版本指针为 0：回填为最新版本（ListByTemplate 按版本号倒序）
		if t.CurrentVersionID == 0 {
			if err := s.templateRepo.UpdateCurrentVersion(ctx, t.ID, versions[0].ID); err != nil {
				return err
			}
		}
	}

	// 记录回填：按所属模板 v1 补 template_version_id
	return db.WithContext(ctx).Exec(`
		UPDATE tb_form_record r
		JOIN tb_form_template_version v
		  ON v.template_id = r.template_id AND v.version = 1
		SET r.template_version_id = v.id
		WHERE r.template_version_id = 0`).Error
}

// TemplateDelete 删除模板（有实例时禁止）
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

// ===== 实例 =====

// normalizeRecordKey 业务主键空值归一：空串返回 nil（落库为 NULL），非空返回指向新变量的指针
func normalizeRecordKey(s string) *string {
	if s == "" {
		return nil
	}
	v := s
	return &v
}

// FormRecordSaveDTO 实例保存参数
type FormRecordSaveDTO struct {
	TemplateKey string                 `json:"template_key" binding:"required"`
	RecordKey   string                 `json:"record_key"`
	Data        map[string]interface{} `json:"data" binding:"required"`
	// 通话上下文（工作台 C 传入；手工填报为空）
	CallID    string `json:"call_id"`
	Direction string `json:"direction"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
}

// FormRecordUpdateDTO 实例更新参数
type FormRecordUpdateDTO struct {
	ID        uint64                 `json:"id" binding:"required"`
	RecordKey string                 `json:"record_key"`
	Data      map[string]interface{} `json:"data" binding:"required"`
}

// RecordPage 实例动态分页（templateKey 为空时跨模板查询；conditions 先做白名单过滤）
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

// filterRecordConditions 按白名单过滤动态条件；非 like 操作统一按 eq 处理
func filterRecordConditions(conds []repository.RecordCondition, allowed map[string]bool) ([]repository.RecordCondition, error) {
	var filtered []repository.RecordCondition
	for _, cond := range conds {
		// 白名单之外的查询字段直接拒绝，不静默忽略，便于前端及时发现配置错误
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

// searchableWhitelist 取可搜索字段白名单；templateKey 为空时取全部模板的并集
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
	// 指定模板但未找到时，白名单为空，给出明确参数错误
	if templateKey != "" && len(allowed) == 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "模板不存在")
	}
	return allowed, nil
}

// tableFieldsUnion 取模板当前版本的表格展示字段并集（按字段名去重，标签以首个模板为准）
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

// RecordGet 实例详情
func (s *FormService) RecordGet(ctx context.Context, tenantID, id uint64) (*model.FormRecord, error) {
	return s.recordRepo.GetByID(ctx, tenantID, id)
}

// RecordCreate 创建实例：校验数据 + 生成快照 + record_key 查重
func (s *FormService) RecordCreate(ctx context.Context, tenantID uint64, dto *FormRecordSaveDTO) (*model.FormRecord, error) {
	tpl, err := s.templateRepo.GetByKey(ctx, tenantID, dto.TemplateKey)
	if err != nil || tpl == nil || tpl.ID == 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "模板不存在")
	}
	if tpl.Status != 1 {
		return nil, errors.NewBizError(errors.CodeParamError, "模板已停用，不能填报新数据")
	}
	// 以模板当前版本校验；记录绑定该版本
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

	// 业务关联：客户主档 / 工单
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

// RecordUpdate 更新实例：按记录绑定版本复核数据并重新生成快照（版本不随模板更新而改变）
func (s *FormService) RecordUpdate(ctx context.Context, tenantID uint64, dto *FormRecordUpdateDTO) error {
	record, err := s.recordRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil || record == nil || record.ID == 0 {
		return errors.NewBizError(errors.CodeParamError, "实例不存在")
	}
	tpl, err := s.templateRepo.GetByKey(ctx, tenantID, record.TemplateKey)
	if err != nil || tpl == nil || tpl.ID == 0 {
		return errors.NewBizError(errors.CodeParamError, "模板不存在")
	}
	// 用记录自己绑定的版本（存量未回填时回退当前版本）
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

// RecordDelete 删除实例
func (s *FormService) RecordDelete(ctx context.Context, tenantID, id uint64) error {
	return s.recordRepo.Delete(ctx, tenantID, id)
}

// RecordExport 导出 CSV：表头与取值翻译均用实例各自快照（模板后续修改不影响历史数据）；
// templateKey 为空时跨模板导出，首列标识所属模板，动态列取全部模板表格字段并集
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

	// 跨模板导出首列为所属模板；单模板导出无此列
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

// translateExportValue 导出值翻译：选项值转 label，多选用顿号拼接
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
