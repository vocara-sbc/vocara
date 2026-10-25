package utils

import (
	"math"
	"strconv"
)

// IntToString 整型转字符串
func IntToString(i int) string {
	return strconv.Itoa(i)
}

// Int64ToString int64 转字符串
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// StringToInt 字符串转整型
func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// StringToIntDefault 字符串转整型（带默认值）
func StringToIntDefault(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// StringToInt64 字符串转 int64
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// StringToInt64Default 字符串转 int64（带默认值）
func StringToInt64Default(s string, def int64) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

// Float64ToString float64 转字符串
func Float64ToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// StringToFloat64 字符串转 float64
func StringToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// StringToFloat64Default 字符串转 float64（带默认值）
func StringToFloat64Default(s string, def float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

// Round 四舍五入
func Round(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Round(f*pow) / pow
}

// Ceil 向上取整
func Ceil(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Ceil(f*pow) / pow
}

// Floor 向下取整
func Floor(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Floor(f*pow) / pow
}

// MaxInt 返回较大的整数
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt 返回较小的整数
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt64 返回较大的 int64
func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MinInt64 返回较小的 int64
func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// AbsInt 整数绝对值
func AbsInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// AbsInt64 int64 绝对值
func AbsInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
