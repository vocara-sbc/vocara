// Package utils provides common utility functions.
// It includes string, UUID, time, number, IP, encryption, and struct copying utilities.
package utils

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Character sets for random strings
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

// RandString generates a random string of uppercase and lowercase letters with the specified length.
func RandString(n int) string {
	return randStringFrom(n, letterBytes)
}

// RandNumString generates a random numeric string with the specified length.
func RandNumString(n int) string {
	return randStringFrom(n, numberBytes)
}

// RandAlphaNum generates a random alphanumeric string with the specified length.
func RandAlphaNum(n int) string {
	return randStringFrom(n, letterNumBytes)
}

// RandHex generates a random hexadecimal string with the specified length.
func RandHex(n int) string {
	return randStringFrom(n, hexBytes)
}

// randStringFrom generates a random string from the specified character set.
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

// Substring extracts a substring with Unicode character support.
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

// Reverse reverses a string.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Contains reports whether a string is in a slice.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Join joins a string slice.
func Join(elems []string, sep string) string {
	return strings.Join(elems, sep)
}

// IsBlank reports whether a string is empty, treating whitespace-only strings as empty.
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotBlank reports whether a string is not empty.
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

// IsEmpty reports whether a string is empty.
func IsEmpty(s string) bool {
	return s == ""
}

// IsNotEmpty reports whether a string is not empty.
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// CamelToSnake converts camel case to snake case.
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

// SnakeToCamel converts snake case to camel case.
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// toLower converts a character to lowercase.
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// PadLeft pads on the left.
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

// PadRight pads on the right.
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

// Truncate truncates a string.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return Substring(s, 0, maxLen) + "..."
}

// FirstUpper capitalizes the first letter.
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

// FirstLower lowercases the first letter.
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
