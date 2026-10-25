package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"vocara/pkg/errors"
)

// FormSchema 表单 schema（字段节点树，version 便于兼容演进）
type FormSchema struct {
	Version    int                    `json:"version"`
	FormConfig map[string]interface{} `json:"formConfig"`
	Fields     []FormNode             `json:"fields"`
}

// FormNode 字段节点：type=grid 时按栅格节点解释（使用 Columns），其余为普通字段节点
type FormNode struct {
	FormField
	Columns []GridColumn `json:"columns,omitempty"`
}

// GridColumn 栅格列：span 为列宽，fields 为列内纵向堆叠的字段节点
type GridColumn struct {
	Span   int        `json:"span"`
	Fields []FormNode `json:"fields"`
}

const gridNodeType = "grid"

// walkFields 深度遍历字段节点：普通节点执行 fn，栅格节点逐列递归；fn 返回 false 时终止
func walkFields(nodes []FormNode, fn func(f FormField) bool) {
	for i := range nodes {
		n := &nodes[i]
		if n.Type == gridNodeType {
			for c := range n.Columns {
				walkFields(n.Columns[c].Fields, fn)
			}
			continue
		}
		if !fn(n.FormField) {
			return
		}
	}
}

// FormField 单个字段定义
type FormField struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Label        string                 `json:"label"`
	Span         int                    `json:"span"`
	Props        map[string]interface{} `json:"props"`
	Rules        FormFieldRules         `json:"rules"`
	DefaultValue interface{}            `json:"defaultValue"`
	ShowInTable  bool                   `json:"showInTable"`
	TableWidth   int                    `json:"tableWidth"`
	Searchable   bool                   `json:"searchable"`
	Sort         int                    `json:"sort"`
}

// FormFieldRules 声明式校验规则（前后端同源，前端翻译成组件校验，后端保存时复核）
type FormFieldRules struct {
	Required    bool   `json:"required"`
	Pattern     string `json:"pattern"`
	PatternHint string `json:"patternHint"`
	MaxLength   int    `json:"maxLength"`
}

// FormOption 选项（radio/checkbox/select 的候选值）
type FormOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// FieldSnapshotItem 拍平字段快照项（实例保存时固化，供列表翻译与导出）
type FieldSnapshotItem struct {
	Name    string       `json:"name"`
	Label   string       `json:"label"`
	Type    string       `json:"type"`
	Options []FormOption `json:"options,omitempty"`
}

// 布局类字段不产生数据、不参与校验与快照
var layoutFieldTypes = map[string]bool{
	"divider":     true,
	"static-text": true,
	gridNodeType:  true,
}

// ParseFormSchema 解析 schema JSON 并做结构校验
func ParseFormSchema(raw string) (*FormSchema, error) {
	var schema FormSchema
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return nil, errors.NewBizError(errors.CodeParamError, "表单 schema 不是合法 JSON")
	}
	if len(schema.Fields) == 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "表单 schema 至少需要一个字段")
	}
	// 栅格节点结构校验先行：列数、列宽、禁止栅格嵌套
	if err := validateNodes(schema.Fields); err != nil {
		return nil, err
	}

	dataFieldCount := 0
	var bizErr *errors.BizError
	// 第一遍：数据字段计数 + name 非空
	walkFields(schema.Fields, func(f FormField) bool {
		if layoutFieldTypes[f.Type] {
			return true
		}
		dataFieldCount++
		if f.Name == "" {
			bizErr = errors.NewBizError(errors.CodeParamError, fmt.Sprintf("字段 %s 缺少 name", f.Label))
			return false
		}
		return true
	})
	if bizErr != nil {
		return nil, bizErr
	}
	// 第二遍：全树 name 唯一
	names := map[string]bool{}
	walkFields(schema.Fields, func(f FormField) bool {
		if layoutFieldTypes[f.Type] {
			return true
		}
		if names[f.Name] {
			bizErr = errors.NewBizError(errors.CodeParamError, fmt.Sprintf("字段 name %s 重复", f.Name))
			return false
		}
		names[f.Name] = true
		return true
	})
	if bizErr != nil {
		return nil, bizErr
	}
	if dataFieldCount == 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "表单至少需要一个数据字段")
	}
	return &schema, nil
}

// validateNodes 校验栅格节点：列数 2–4、列 span 4–24、列内禁止栅格节点；递归校验各列
func validateNodes(nodes []FormNode) error {
	for i := range nodes {
		n := &nodes[i]
		if n.Type != gridNodeType {
			continue
		}
		if len(n.Columns) < 2 || len(n.Columns) > 4 {
			return errors.NewBizError(errors.CodeParamError, "栅格列数必须在2到4之间")
		}
		for c := range n.Columns {
			col := &n.Columns[c]
			if col.Span < 4 || col.Span > 24 {
				return errors.NewBizError(errors.CodeParamError, "栅格列宽必须在4到24之间")
			}
			for k := range col.Fields {
				if col.Fields[k].Type == gridNodeType {
					return errors.NewBizError(errors.CodeParamError, "栅格列内不能再放置栅格")
				}
			}
			if err := validateNodes(col.Fields); err != nil {
				return err
			}
		}
	}
	return nil
}

// FlattenSnapshot 生成拍平字段快照（name/label/type/options），布局与栅格节点本身除外
func FlattenSnapshot(schema *FormSchema) []FieldSnapshotItem {
	var snap []FieldSnapshotItem
	walkFields(schema.Fields, func(f FormField) bool {
		if layoutFieldTypes[f.Type] {
			return true
		}
		item := FieldSnapshotItem{Name: f.Name, Label: f.Label, Type: f.Type}
		if f.Type == "radio" || f.Type == "checkbox" || f.Type == "select" {
			item.Options = extractOptions(f.Props)
		}
		snap = append(snap, item)
		return true
	})
	return snap
}

// extractOptions 从 props 提取选项数组
func extractOptions(props map[string]interface{}) []FormOption {
	raw, ok := props["options"].([]interface{})
	if !ok {
		return nil
	}
	var options []FormOption
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		label, _ := m["label"].(string)
		options = append(options, FormOption{Label: label, Value: m["value"]})
	}
	return options
}

// SearchableFieldSet 返回可搜索字段白名单（动态查询唯一允许的字段集合）
func SearchableFieldSet(schema *FormSchema) map[string]bool {
	set := map[string]bool{}
	walkFields(schema.Fields, func(f FormField) bool {
		if f.Searchable && !layoutFieldTypes[f.Type] {
			set[f.Name] = true
		}
		return true
	})
	return set
}

// ValidateRecordData 按 schema 拍平字段复核实例数据：必填/长度/正则/数字范围/选项合法性
func ValidateRecordData(schema *FormSchema, data map[string]interface{}) error {
	var bizErr error
	walkFields(schema.Fields, func(f FormField) bool {
		if layoutFieldTypes[f.Type] {
			return true
		}
		if err := validateOneField(f, data); err != nil {
			bizErr = err
			return false
		}
		return true
	})
	return bizErr
}

// validateOneField 复核单个字段：必填/长度/正则/数字范围/选项合法性
func validateOneField(f FormField, data map[string]interface{}) error {
	value, exists := data[f.Name]
	if f.Rules.Required && isEmptyFormValue(value) {
		return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s不能为空", f.Label))
	}
	if !exists || value == nil {
		return nil
	}
	switch f.Type {
	case "input", "textarea":
		text, ok := value.(string)
		if !ok {
			return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s必须是文本", f.Label))
		}
		if f.Rules.MaxLength > 0 && len([]rune(text)) > f.Rules.MaxLength {
			return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s长度不能超过%d字", f.Label, f.Rules.MaxLength))
		}
		if f.Rules.Pattern != "" && text != "" {
			matched, err := regexp.MatchString(f.Rules.Pattern, text)
			if err == nil && !matched {
				hint := f.Rules.PatternHint
				if hint == "" {
					hint = fmt.Sprintf("%s格式不正确", f.Label)
				}
				return errors.NewBizError(errors.CodeParamError, hint)
			}
		}
	case "number":
		num, ok := value.(float64)
		if !ok {
			return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s必须是数字", f.Label))
		}
		if min, ok := f.Props["min"].(float64); ok && num < min {
			return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s不能小于%v", f.Label, min))
		}
		if max, ok := f.Props["max"].(float64); ok && num > max {
			return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s不能大于%v", f.Label, max))
		}
	case "radio", "select":
		if f.Props["multiple"] == true {
			values, ok := value.([]interface{})
			if !ok {
				return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s必须是数组", f.Label))
			}
			for _, v := range values {
				if !optionContains(extractOptions(f.Props), v) {
					return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s包含无效选项", f.Label))
				}
			}
		} else if !optionContains(extractOptions(f.Props), value) {
			return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s包含无效选项", f.Label))
		}
	case "checkbox":
		values, ok := value.([]interface{})
		if !ok {
			return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s必须是数组", f.Label))
		}
		for _, v := range values {
			if !optionContains(extractOptions(f.Props), v) {
				return errors.NewBizError(errors.CodeParamError, fmt.Sprintf("%s包含无效选项", f.Label))
			}
		}
	}
	return nil
}

// optionContains 判断值是否命中选项（JSON 数字统一为 float64 比较）
func optionContains(options []FormOption, value interface{}) bool {
	for _, opt := range options {
		if fmt.Sprintf("%v", opt.Value) == fmt.Sprintf("%v", value) {
			return true
		}
	}
	return false
}

// isEmptyFormValue 必填判空：nil、空串、纯空白串、空数组
func isEmptyFormValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	}
	return false
}
