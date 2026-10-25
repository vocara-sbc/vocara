package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ===== Task 1: walkFields =====

func TestWalkFieldsFlat(t *testing.T) {
	nodes := []FormNode{
		{FormField: FormField{Type: "input", Name: "a"}},
		{FormField: FormField{Type: "divider", Name: ""}},
	}
	var got []string
	walkFields(nodes, func(f FormField) bool { got = append(got, f.Name); return true })
	if len(got) != 2 || got[0] != "a" {
		t.Fatalf("平铺节点应收 2 个字段，实际 %v", got)
	}
}

func TestWalkFieldsGrid(t *testing.T) {
	nodes := []FormNode{{
		FormField: FormField{Type: gridNodeType, Name: ""},
		Columns: []GridColumn{{Span: 12, Fields: []FormNode{
			{FormField: FormField{Type: "input", Name: "in_col"}},
		}}},
	}}
	var got []string
	walkFields(nodes, func(f FormField) bool { got = append(got, f.Name); return true })
	if len(got) != 1 || got[0] != "in_col" {
		t.Fatalf("栅格内字段应被收集，实际 %v", got)
	}
}

func TestWalkFieldsEarlyStop(t *testing.T) {
	nodes := []FormNode{
		{FormField: FormField{Type: "input", Name: "a"}},
		{FormField: FormField{Type: "input", Name: "b"}},
	}
	count := 0
	walkFields(nodes, func(f FormField) bool { count++; return false })
	if count != 1 {
		t.Fatalf("fn 返回 false 应在首个字段终止，实际执行 %d 次", count)
	}
}

// ===== Task 2: 含栅格 schema 的校验与快照 =====

func gridSchema(inner []FormNode) *FormSchema {
	return &FormSchema{Fields: []FormNode{{
		FormField: FormField{Type: gridNodeType},
		Columns: []GridColumn{
			{Span: 12, Fields: inner},
			{Span: 12},
		},
	}}}
}

func TestParseGridOK(t *testing.T) {
	s := gridSchema([]FormNode{{FormField: FormField{Type: "input", Name: "x", Label: "X"}}})
	if _, err := ParseFormSchema(mustMarshal(t, s)); err != nil {
		t.Fatalf("合法含栅格 schema 应通过，实际 %v", err)
	}
}

func TestParseGridDuplicateNameAcrossTree(t *testing.T) {
	s := &FormSchema{Fields: []FormNode{
		{FormField: FormField{Type: "input", Name: "dup", Label: "A"}},
		{FormField: FormField{Type: gridNodeType}, Columns: []GridColumn{
			{Span: 12, Fields: []FormNode{{FormField: FormField{Type: "input", Name: "dup", Label: "B"}}}},
		}},
	}}
	if _, err := ParseFormSchema(mustMarshal(t, s)); err == nil {
		t.Fatal("全树 name 重复应拒绝")
	}
}

func TestParseNestedGridRejected(t *testing.T) {
	s := gridSchema([]FormNode{{FormField: FormField{Type: gridNodeType}}})
	_, err := ParseFormSchema(mustMarshal(t, s))
	if err == nil || !strings.Contains(err.Error(), "栅格") {
		t.Fatalf("列内出现栅格应拒绝并提示栅格，实际 %v", err)
	}
}

func TestParseColumnCountBounds(t *testing.T) {
	s := &FormSchema{Fields: []FormNode{{
		FormField: FormField{Type: gridNodeType},
		Columns:   []GridColumn{{Span: 24}}, // 仅 1 列
	}}}
	if _, err := ParseFormSchema(mustMarshal(t, s)); err == nil {
		t.Fatal("列数越界（<2）应拒绝")
	}
}

func TestFlattenSnapshotGrid(t *testing.T) {
	s := gridSchema([]FormNode{{FormField: FormField{Type: "input", Name: "x", Label: "X"}}})
	snap := FlattenSnapshot(s)
	if len(snap) != 1 || snap[0].Name != "x" {
		t.Fatalf("快照应包含栅格内字段，实际 %+v", snap)
	}
}
