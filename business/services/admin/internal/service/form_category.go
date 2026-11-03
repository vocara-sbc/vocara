package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

var categoryCodePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{1,31}$`)

// FormCategory is a form category.
type FormCategory struct {
	Code    string `json:"code"`    // English code and primary key; immutable after creation
	Name    string `json:"name"`    // Display name
	Sort    int    `json:"sort"`    // Ascending sort order
	Enabled bool   `json:"enabled"` // Enabled or disabled
	Builtin bool   `json:"builtin"` // Built-in marker
}

const (
	formCategoryConfigKey = "form_category_list"
	formCategoryTenantID  = 0
)

// builtinCategories returns built-in categories in the order used by existing template values.
func builtinCategories() []FormCategory {
	return []FormCategory{
		{Code: "customer", Name: "客户", Sort: 1, Enabled: true, Builtin: true},
		{Code: "general", Name: "通用", Sort: 2, Enabled: true, Builtin: true},
		{Code: "ticket", Name: "工单", Sort: 3, Enabled: true, Builtin: true},
	}
}

// FormCategoryService provides form category services and stores categories as JSON in one global SystemConfig entry.
type FormCategoryService struct {
	systemRepo *repository.SystemRepository
	tplRepo    *repository.FormTemplateRepository
}

// NewFormCategoryService creates a form category service.
func NewFormCategoryService() *FormCategoryService {
	return &FormCategoryService{
		systemRepo: repository.NewSystemRepository(),
		tplRepo:    repository.NewFormTemplateRepository(),
	}
}

// GetCategories loads categories and initializes and persists the three built-in categories when configuration is missing.
func (s *FormCategoryService) GetCategories(ctx context.Context) ([]FormCategory, error) {
	cfg, err := s.systemRepo.GetSystemConfig(ctx, formCategoryTenantID, formCategoryConfigKey)
	if err != nil || cfg == nil || cfg.ConfigValue == "" {
		return s.initBuiltin(ctx)
	}
	var list []FormCategory
	if err := json.Unmarshal([]byte(cfg.ConfigValue), &list); err != nil {
		return nil, errors.NewBizError(errors.CodeParamError, "表单分类配置不是合法 JSON")
	}
	sortCategories(list)
	return list, nil
}

// initBuiltin persists and returns the built-in categories.
func (s *FormCategoryService) initBuiltin(ctx context.Context) ([]FormCategory, error) {
	list := builtinCategories()
	if err := s.persist(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

// persist serializes categories as JSON into the designated configuration entry.
func (s *FormCategoryService) persist(ctx context.Context, list []FormCategory) error {
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	cfg := &model.SystemConfig{
		TenantID:    formCategoryTenantID,
		ConfigKey:   formCategoryConfigKey,
		ConfigValue: string(b),
		ConfigType:  "json",
	}
	return s.systemRepo.SaveSystemConfig(ctx, cfg)
}

// sortCategories sorts categories by Sort in ascending order.
func sortCategories(list []FormCategory) {
	sort.SliceStable(list, func(i, j int) bool { return list[i].Sort < list[j].Sort })
}

// isBuiltinCode reports whether a code identifies a built-in category.
func isBuiltinCode(code string) bool {
	for _, b := range builtinCategories() {
		if b.Code == code {
			return true
		}
	}
	return false
}

// validateCategoryList validates the complete list: code format and uniqueness, valid names, all built-ins present, and general enabled.
func validateCategoryList(list []FormCategory) error {
	seen := map[string]bool{}
	builtinSeen := map[string]bool{}
	for _, c := range list {
		if !categoryCodePattern.MatchString(c.Code) {
			return errors.NewBizError(errors.CodeParamError, "分类编码格式不合法："+c.Code)
		}
		if seen[c.Code] {
			return errors.NewBizError(errors.CodeParamError, "分类编码重复："+c.Code)
		}
		seen[c.Code] = true
		if strings.TrimSpace(c.Name) == "" || len([]rune(c.Name)) > 32 {
			return errors.NewBizError(errors.CodeParamError, "分类名称不合法："+c.Code)
		}
		if c.Builtin {
			builtinSeen[c.Code] = true
		}
	}
	for _, b := range builtinCategories() {
		if !builtinSeen[b.Code] {
			return errors.NewBizError(errors.CodeParamError, "内置分类不可删除："+b.Code)
		}
	}
	for _, c := range list {
		if c.Code == "general" && !c.Enabled {
			return errors.NewBizError(errors.CodeParamError, "通用分类不可停用")
		}
	}
	return nil
}

// SaveCategories validates and saves the complete category list, including template-reference checks for removed non-built-in categories.
func (s *FormCategoryService) SaveCategories(ctx context.Context, list []FormCategory) error {
	// Derive built-in markers from the server-side set to prevent frontend tampering.
	for i := range list {
		if isBuiltinCode(list[i].Code) {
			list[i].Builtin = true
		}
	}
	if err := validateCategoryList(list); err != nil {
		return err
	}
	old, err := s.GetCategories(ctx)
	if err != nil {
		return err
	}
	newSet := map[string]bool{}
	for _, c := range list {
		newSet[c.Code] = true
	}
	for _, c := range old {
		if c.Builtin || newSet[c.Code] {
			continue
		}
		count, err := s.tplRepo.CountByCategory(ctx, c.Code)
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.NewBizError(errors.CodeParamError,
				fmt.Sprintf("分类 %s 下仍有 %d 个模板，不能删除", c.Name, count))
		}
	}
	sortCategories(list)
	return s.persist(ctx, list)
}

// ExistsCode reports whether a category code exists, including disabled categories.
func (s *FormCategoryService) ExistsCode(ctx context.Context, code string) (bool, error) {
	list, err := s.GetCategories(ctx)
	if err != nil {
		return false, err
	}
	for _, c := range list {
		if c.Code == code {
			return true, nil
		}
	}
	return false, nil
}
