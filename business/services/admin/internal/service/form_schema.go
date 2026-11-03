package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"vocara/pkg/errors"
)

// FormSchema is a form schema represented as a field-node tree; version supports compatible schema evolution.
type FormSchema struct {
	Version    int                    `json:"version"`
	FormConfig map[string]interface{} `json:"formConfig"`
	Fields     []FormNode             `json:"fields"`
}

// FormNode is a field node. A type=grid node is interpreted as a grid using Columns; all other nodes are regular fields.
type FormNode struct {
	FormField
	Columns []GridColumn `json:"columns,omitempty"`
}

// GridColumn is a grid column; span is its width and fields contains vertically stacked field nodes.
type GridColumn struct {
	Span   int        `json:"span"`
	Fields []FormNode `json:"fields"`
}

const gridNodeType = "grid"

// walkFields traverses field nodes depth-first, invoking fn for regular nodes and recursing through grid columns; it stops when fn returns false.
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

// FormField defines a single field.
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

// FormFieldRules defines declarative validation rules shared by the frontend and backend; the frontend maps them to component validation and the backend validates them again when saving.
type FormFieldRules struct {
	Required    bool   `json:"required"`
	Pattern     string `json:"pattern"`
	PatternHint string `json:"patternHint"`
	MaxLength   int    `json:"maxLength"`
}

// FormOption is a candidate value for radio, checkbox, and select fields.
type FormOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// FieldSnapshotItem is a flattened field snapshot item captured when an instance is saved for list rendering and export.
type FieldSnapshotItem struct {
	Name    string       `json:"name"`
	Label   string       `json:"label"`
	Type    string       `json:"type"`
	Options []FormOption `json:"options,omitempty"`
}

// Layout fields produce no data and are excluded from validation and snapshots.
var layoutFieldTypes = map[string]bool{
	"divider":     true,
	"static-text": true,
	gridNodeType:  true,
}

// ParseFormSchema parses schema JSON and validates its structure.
func ParseFormSchema(raw string) (*FormSchema, error) {
	var schema FormSchema
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return nil, errors.NewBizError(errors.CodeParamError, "表单 schema 不是合法 JSON")
	}
	if len(schema.Fields) == 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "表单 schema 至少需要一个字段")
	}
	// Validate grid structure first: column count, column width, and prohibition on nested grids.
	if err := validateNodes(schema.Fields); err != nil {
		return nil, err
	}

	dataFieldCount := 0
	var bizErr *errors.BizError
	// First pass: count data fields and require non-empty names.
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
	// Second pass: require unique names throughout the tree.
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

// validateNodes validates grid nodes recursively: 2-4 columns, column spans of 4-24, and no grid nodes inside columns.
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

// FlattenSnapshot generates a flattened field snapshot containing name, label, type, and options, excluding layout and grid nodes themselves.
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

// extractOptions extracts the options array from props.
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

// SearchableFieldSet returns the searchable-field allowlist, the only fields permitted in dynamic queries.
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

// ValidateRecordData validates instance data against the schema's flattened fields, including required values, length, patterns, numeric ranges, and valid options.
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

// validateOneField validates one field's required value, length, pattern, numeric range, and options.
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

// optionContains reports whether a value matches an option, comparing JSON numbers uniformly as float64.
func optionContains(options []FormOption, value interface{}) bool {
	for _, opt := range options {
		if fmt.Sprintf("%v", opt.Value) == fmt.Sprintf("%v", value) {
			return true
		}
	}
	return false
}

// isEmptyFormValue reports whether a required value is empty: nil, an empty or whitespace-only string, or an empty array.
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
