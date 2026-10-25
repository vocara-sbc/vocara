package service

import (
	"strings"
	"testing"
)

const testSchemaJSON = `{"version":1,"formConfig":{},"fields":[
{"id":"f1","type":"input","name":"customer_name","label":"客户姓名","span":12,"props":{},"rules":{"required":true,"pattern":"","patternHint":"","maxLength":50},"showInTable":true,"searchable":true,"sort":1},
{"id":"f2","type":"select","name":"tag","label":"标签","span":12,"props":{"options":[{"label":"VIP","value":"vip"},{"label":"普通","value":"normal"}],"multiple":true},"rules":{},"showInTable":true,"searchable":true,"sort":2},
{"id":"f3","type":"number","name":"age","label":"年龄","span":12,"props":{"min":0,"max":150},"rules":{},"searchable":false,"sort":3},
{"id":"f4","type":"divider","name":"d1","label":"分割线","span":24,"props":{},"rules":{},"sort":4}
]}`

func TestParseFormSchema(t *testing.T) {
	schema, err := ParseFormSchema(testSchemaJSON)
	if err != nil {
		t.Fatalf("ParseFormSchema: %v", err)
	}
	if len(schema.Fields) != 4 {
		t.Fatalf("fields = %d, want 4", len(schema.Fields))
	}
	if _, err := ParseFormSchema("{bad json"); err == nil {
		t.Fatal("坏 JSON 应报错")
	}
}

func TestFlattenSnapshot(t *testing.T) {
	schema, _ := ParseFormSchema(testSchemaJSON)
	snap := FlattenSnapshot(schema)
	if len(snap) != 3 { // divider 不进快照
		t.Fatalf("snapshot = %d, want 3", len(snap))
	}
	if snap[1].Name != "tag" || len(snap[1].Options) != 2 || snap[1].Options[0].Value != "vip" {
		t.Fatalf("select 快照 options 提取错误: %+v", snap[1])
	}
}

func TestSearchableFieldSet(t *testing.T) {
	schema, _ := ParseFormSchema(testSchemaJSON)
	set := SearchableFieldSet(schema)
	if !set["customer_name"] || !set["tag"] {
		t.Fatal("searchable 字段应入白名单")
	}
	if set["age"] || set["d1"] {
		t.Fatal("非 searchable 字段不应入白名单")
	}
}

func TestValidateRecordData(t *testing.T) {
	schema, _ := ParseFormSchema(testSchemaJSON)

	// 必填缺失
	if err := ValidateRecordData(schema, map[string]interface{}{}); err == nil {
		t.Fatal("缺必填字段应报错")
	}
	// 合法数据
	ok := map[string]interface{}{"customer_name": "张三", "tag": []interface{}{"vip", "normal"}, "age": float64(30)}
	if err := ValidateRecordData(schema, ok); err != nil {
		t.Fatalf("合法数据不应报错: %v", err)
	}
	// 超长
	if err := ValidateRecordData(schema, map[string]interface{}{"customer_name": strings.Repeat("长", 51)}); err == nil {
		t.Fatal("超长应报错")
	}
	// 选项外值
	if err := ValidateRecordData(schema, map[string]interface{}{"customer_name": "张三", "tag": []interface{}{"vip", "evil"}}); err == nil {
		t.Fatal("选项外值应报错")
	}
	// 数字越界
	if err := ValidateRecordData(schema, map[string]interface{}{"customer_name": "张三", "age": float64(200)}); err == nil {
		t.Fatal("数字越界应报错")
	}
	// 必填字段空串
	if err := ValidateRecordData(schema, map[string]interface{}{"customer_name": "  "}); err == nil {
		t.Fatal("必填空串应报错")
	}
}

func TestNormalizeRecordKey(t *testing.T) {
	// 空串归一为 nil（落库 NULL，不触发唯一键冲突）
	if got := normalizeRecordKey(""); got != nil {
		t.Fatalf("空串应归一为 nil, got %q", *got)
	}
	// 非空返回原值指针
	got := normalizeRecordKey("order-123")
	if got == nil || *got != "order-123" {
		t.Fatalf("非空应返回原值指针, got %v", got)
	}
	// 两次调用返回独立指针，互不共享底层变量
	a := normalizeRecordKey("k")
	b := normalizeRecordKey("k")
	if a == b {
		t.Fatal("两次调用不应共享同一指针")
	}
}
