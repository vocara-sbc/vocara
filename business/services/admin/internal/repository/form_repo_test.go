package repository

import "testing"

// TestValidateConditionField 动态查询字段名必须满足标识符格式，杜绝 JSON 路径注入
func TestValidateConditionField(t *testing.T) {
	valid := []string{"customer_name", "phone", "a", "_x1", "field_2024"}
	for _, name := range valid {
		if !ValidateConditionField(name) {
			t.Errorf("ValidateConditionField(%q) = false, want true", name)
		}
	}
	invalid := []string{"", "1abc", "a.b", "a' or 1=1 --", "a$", "中文字段", "a b", "a-b"}
	for _, name := range invalid {
		if ValidateConditionField(name) {
			t.Errorf("ValidateConditionField(%q) = true, want false", name)
		}
	}
	longName := make([]byte, 65)
	for i := range longName {
		longName[i] = 'a'
	}
	if ValidateConditionField(string(longName)) {
		t.Error("65 字符字段名应被拒绝")
	}
}
