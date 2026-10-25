package utils

import (
	"strings"
	"testing"
)

func TestRandString(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"长度10", 10},
		{"长度32", 32},
		{"长度0", 0},
		{"长度-1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandString(tt.n)
			expectedLen := 0
			if tt.n > 0 {
				expectedLen = tt.n
			}
			if len(result) != expectedLen {
				t.Errorf("length = %d, want %d", len(result), expectedLen)
			}
			// 检查是否都是字母
			for _, c := range result {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
					t.Errorf("unexpected char: %c", c)
				}
			}
		})
	}
}

func TestRandNumString(t *testing.T) {
	s := RandNumString(10)
	if len(s) != 10 {
		t.Errorf("length = %d, want 10", len(s))
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Errorf("unexpected char: %c", c)
		}
	}
}

func TestRandAlphaNum(t *testing.T) {
	s := RandAlphaNum(20)
	if len(s) != 20 {
		t.Errorf("length = %d, want 20", len(s))
	}
}

func TestSubstring(t *testing.T) {
	tests := []struct {
		s      string
		start  int
		length int
		want   string
	}{
		{"hello world", 0, 5, "hello"},
		{"hello world", 6, 5, "world"},
		{"hello world", 0, 100, "hello world"},
		{"hello world", 100, 10, ""},
		{"你好世界", 0, 2, "你好"},
		{"你好世界", 2, 2, "世界"},
		{"test", -1, 2, "te"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := Substring(tt.s, tt.start, tt.length)
			if got != tt.want {
				t.Errorf("Substring(%q, %d, %d) = %q, want %q",
					tt.s, tt.start, tt.length, got, tt.want)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "olleh"},
		{"abc", "cba"},
		{"", ""},
		{"a", "a"},
		{"你好", "好你"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Reverse(tt.input)
			if got != tt.want {
				t.Errorf("Reverse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !Contains(slice, "a") {
		t.Error("Contains should return true for 'a'")
	}
	if Contains(slice, "d") {
		t.Error("Contains should return false for 'd'")
	}
}

func TestIsBlank(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{" ", true},
		{"  ", true},
		{"\t", true},
		{"\n", true},
		{"a", false},
		{" a ", false},
	}

	for _, tt := range tests {
		t.Run("blank:"+tt.input, func(t *testing.T) {
			if got := IsBlank(tt.input); got != tt.want {
				t.Errorf("IsBlank(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CamelCase", "camel_case"},
		{"camelCase", "camel_case"},
		{"Camel", "camel"},
		{"HTTPServer", "h_t_t_p_server"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CamelToSnake(tt.input)
			if got != tt.want {
				t.Errorf("CamelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"snake_case", "SnakeCase"},
		{"hello_world", "HelloWorld"},
		{"test", "Test"},
		{"a_b_c", "ABC"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SnakeToCamel(tt.input)
			if got != tt.want {
				t.Errorf("SnakeToCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		s      string
		length int
		char   rune
		want   string
	}{
		{"123", 5, '0', "00123"},
		{"abc", 3, '0', "abc"},
		{"", 3, '*', "***"},
		{"longstring", 5, '0', "longstring"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := PadLeft(tt.s, tt.length, tt.char)
			if got != tt.want {
				t.Errorf("PadLeft(%q, %d, %c) = %q, want %q",
					tt.s, tt.length, tt.char, got, tt.want)
			}
		})
	}
}

func TestFirstUpper(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"", ""},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FirstUpper(tt.input)
			if got != tt.want {
				t.Errorf("FirstUpper(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFirstLower(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello", "hello"},
		{"hello", "hello"},
		{"", ""},
		{"A", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FirstLower(tt.input)
			if got != tt.want {
				t.Errorf("FirstLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		wantLen int
	}{
		{"hello", 10, 5},
		{"hello world", 5, 8}, // 5 + "..."
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := Truncate(tt.s, tt.maxLen)
			if len([]rune(got)) != tt.wantLen {
				t.Errorf("Truncate(%q, %d) length = %d, want %d",
					tt.s, tt.maxLen, len([]rune(got)), tt.wantLen)
			}
			if len(tt.s) > tt.maxLen && !strings.HasSuffix(got, "...") {
				t.Error("should end with '...'")
			}
		})
	}
}
