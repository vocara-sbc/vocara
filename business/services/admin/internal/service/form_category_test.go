package service

import "testing"

func TestBuiltinCategories(t *testing.T) {
	b := builtinCategories()
	if len(b) != 3 {
		t.Fatalf("内置应有 3 项，实际 %d", len(b))
	}
	codes := map[string]bool{}
	for _, c := range b {
		codes[c.Code] = true
	}
	for _, code := range []string{"customer", "general", "ticket"} {
		if !codes[code] {
			t.Fatalf("内置缺少 %s", code)
		}
	}
}

func TestValidateCategoryList(t *testing.T) {
	builtin := builtinCategories()

	bad := append(append([]FormCategory{}, builtin...), FormCategory{Code: "1x", Name: "X"})
	if err := validateCategoryList(bad); err == nil {
		t.Fatal("非法 code 应拒绝")
	}

	dup := append(append([]FormCategory{}, builtin...), FormCategory{Code: "customer", Name: "X"})
	if err := validateCategoryList(dup); err == nil {
		t.Fatal("重复 code 应拒绝")
	}

	if err := validateCategoryList([]FormCategory{builtin[0]}); err == nil {
		t.Fatal("缺少内置项应拒绝")
	}

	gen := append([]FormCategory{}, builtin...)
	for i := range gen {
		if gen[i].Code == "general" {
			gen[i].Enabled = false
		}
	}
	if err := validateCategoryList(gen); err == nil {
		t.Fatal("general 停用应拒绝")
	}

	ok := append(append([]FormCategory{}, builtin...),
		FormCategory{Code: "follow", Name: "回访", Enabled: true})
	if err := validateCategoryList(ok); err != nil {
		t.Fatalf("合法列表应通过，实际 %v", err)
	}

	noname := append([]FormCategory{}, ok...)
	noname[len(noname)-1].Name = ""
	if err := validateCategoryList(noname); err == nil {
		t.Fatal("空 name 应拒绝")
	}
}
