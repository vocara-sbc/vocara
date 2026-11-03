package utils

import (
	"math"
	"strconv"
)

// IntToString converts an integer to a string.
func IntToString(i int) string {
	return strconv.Itoa(i)
}

// Int64ToString converts an int64 to a string.
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// StringToInt converts a string to an integer.
func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// StringToIntDefault converts a string to an integer with a default value.
func StringToIntDefault(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// StringToInt64 converts a string to an int64.
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// StringToInt64Default converts a string to an int64 with a default value.
func StringToInt64Default(s string, def int64) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

// Float64ToString converts a float64 to a string.
func Float64ToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// StringToFloat64 converts a string to a float64.
func StringToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// StringToFloat64Default converts a string to a float64 with a default value.
func StringToFloat64Default(s string, def float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

// Round rounds to the specified number of decimal places.
func Round(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Round(f*pow) / pow
}

// Ceil rounds up to the specified number of decimal places.
func Ceil(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Ceil(f*pow) / pow
}

// Floor rounds down to the specified number of decimal places.
func Floor(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Floor(f*pow) / pow
}

// MaxInt returns the larger integer.
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the smaller integer.
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt64 returns the larger int64.
func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MinInt64 returns the smaller int64.
func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// AbsInt returns the absolute value of an integer.
func AbsInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// AbsInt64 returns the absolute value of an int64.
func AbsInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
