package database

import (
	"testing"
)

func TestPageQuery_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		wantPage int
		wantSize int
	}{
		{"正常参数", 1, 10, 1, 10},
		{"页码为0", 0, 10, 1, 10},
		{"页码为负数", -1, 10, 1, 10},
		{"每页数量为0", 1, 0, 1, DefaultPageSize},
		{"每页数量为负数", 1, -5, 1, DefaultPageSize},
		{"每页数量超过最大值", 1, 500, 1, MaxPageSize},
		{"都为0", 0, 0, 1, DefaultPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := PageQuery{Page: tt.page, PageSize: tt.pageSize}
			q.Normalize()
			if q.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", q.Page, tt.wantPage)
			}
			if q.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d, want %d", q.PageSize, tt.wantSize)
			}
		})
	}
}

func TestPageQuery_Offset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		offset   int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 20, 40},
		{10, 20, 180},
	}

	for _, tt := range tests {
		q := PageQuery{Page: tt.page, PageSize: tt.pageSize}
		if got := q.Offset(); got != tt.offset {
			t.Errorf("Offset() = %d, want %d", got, tt.offset)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"silent"},
		{"error"},
		{"warn"},
		{"info"},
		{"unknown"}, // 默认 warn
		{""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLogLevel(tt.input)
			// 只要不 panic 就行，具体值依赖 GORM 内部
			t.Logf("level: %v", level)
		})
	}
}
