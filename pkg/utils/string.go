// Package utils 提供常用工具函数
// 包括字符串工具、UUID、时间工具、数字工具、IP工具、加密工具、结构体拷贝等
package utils

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

// 随机字符串字符集
const (
	letterBytes    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberBytes    = "0123456789"
	letterNumBytes = letterBytes + numberBytes
	hexBytes       = "0123456789abcdef"
)

var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// RandString 生成指定长度的随机字符串（大小写字母）
func RandString(n int) string {
	return randStringFrom(n, letterBytes)
}

// RandNumString 生成指定长度的随机数字字符串
func RandNumString(n int) string {
	return randStringFrom(n, numberBytes)
}

// RandAlphaNum 生成指定长度的随机字母数字字符串
func RandAlphaNum(n int) string {
	return randStringFrom(n, letterNumBytes)
}

// RandHex 生成指定长度的随机十六进制字符串
func RandHex(n int) string {
	return randStringFrom(n, hexBytes)
}

// randStringFrom 从指定字符集生成随机字符串
func randStringFrom(n int, charset string) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	rngMu.Lock()
	defer rngMu.Unlock()
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}

// Substring 截取字符串（支持中文）
func Substring(s string, start, length int) string {
	runes := []rune(s)
	if start < 0 {
		start = 0
	}
	if start >= len(runes) {
		return ""
	}
	end := start + length
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

// Reverse 反转字符串
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Contains 字符串是否在切片中
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Join 字符串切片连接
func Join(elems []string, sep string) string {
	return strings.Join(elems, sep)
}

// IsBlank 判断字符串是否为空（空白字符视为空）
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotBlank 判断字符串是否不为空
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

// IsEmpty 判断字符串是否为空
func IsEmpty(s string) bool {
	return s == ""
}

// IsNotEmpty 判断字符串是否不为空
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// CamelToSnake 驼峰转下划线
func CamelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, toLower(r))
	}
	return string(result)
}

// SnakeToCamel 下划线转驼峰
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// toLower 字符转小写
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// PadLeft 左侧填充
func PadLeft(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	pad := make([]rune, length-len(s))
	for i := range pad {
		pad[i] = padChar
	}
	return string(pad) + s
}

// PadRight 右侧填充
func PadRight(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	pad := make([]rune, length-len(s))
	for i := range pad {
		pad[i] = padChar
	}
	return s + string(pad)
}

// Truncate 截断字符串
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return Substring(s, 0, maxLen) + "..."
}

// FirstUpper 首字母大写
func FirstUpper(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] = runes[0] - 32
	}
	return string(runes)
}

// FirstLower 首字母小写
func FirstLower(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if runes[0] >= 'A' && runes[0] <= 'Z' {
		runes[0] = runes[0] + 32
	}
	return string(runes)
}
