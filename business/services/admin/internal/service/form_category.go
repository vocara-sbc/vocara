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

// FormCategory 表单分类项
type FormCategory struct {
	Code    string `json:"code"`    // 英文编码，主键，创建后不可改
	Name    string `json:"name"`    // 显示名称
	Sort    int    `json:"sort"`    // 排序升序
	Enabled bool   `json:"enabled"` // 启用/停用
	Builtin bool   `json:"builtin"` // 内置标记
}

const (
	formCategoryConfigKey = "form_category_list"
	formCategoryTenantID  = 0
)

// builtinCategories 内置分类（顺序即现有模板取值）
func builtinCategories() []FormCategory {
	return []FormCategory{
		{Code: "customer", Name: "客户", Sort: 1, Enabled: true, Builtin: true},
		{Code: "general", Name: "通用", Sort: 2, Enabled: true, Builtin: true},
		{Code: "ticket", Name: "工单", Sort: 3, Enabled: true, Builtin: true},
	}
}

// FormCategoryService 表单分类服务：分类以一条全局 SystemConfig JSON 存储
type FormCategoryService struct {
	systemRepo *repository.SystemRepository
	tplRepo    *repository.FormTemplateRepository
}

// NewFormCategoryService 创建表单分类服务
func NewFormCategoryService() *FormCategoryService {
	return &FormCategoryService{
		systemRepo: repository.NewSystemRepository(),
		tplRepo:    repository.NewFormTemplateRepository(),
	}
}

// GetCategories 读取分类；配置缺失时初始化内置三项并落库
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

// initBuiltin 写入内置分类并返回
func (s *FormCategoryService) initBuiltin(ctx context.Context) ([]FormCategory, error) {
	list := builtinCategories()
	if err := s.persist(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

// persist 将分类序列化为 JSON 存入固定配置
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

// sortCategories 按 Sort 升序排序
func sortCategories(list []FormCategory) {
	sort.SliceStable(list, func(i, j int) bool { return list[i].Sort < list[j].Sort })
}

// isBuiltinCode 判断编码是否为内置分类
func isBuiltinCode(code string) bool {
	for _, b := range builtinCategories() {
		if b.Code == code {
			return true
		}
	}
	return false
}

// validateCategoryList 全量校验：code 格式/唯一、name 合法、内置齐全、general 保持启用
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

// SaveCategories 保存全量分类：结构校验 + 被移除非内置项的模板引用检查
func (s *FormCategoryService) SaveCategories(ctx context.Context, list []FormCategory) error {
	// 内置项标记以服务端内置集合为准，防止前端篡改
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

// ExistsCode 判断分类编码是否存在（含停用项）
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
