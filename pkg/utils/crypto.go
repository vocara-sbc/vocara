package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// MD5 计算字符串的 MD5 哈希值
func MD5(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// MD5Bytes 计算字节数组的 MD5 哈希值
func MD5Bytes(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])
}

// SHA256 计算字符串的 SHA256 哈希值
func SHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// SHA256Bytes 计算字节数组的 SHA256 哈希值
func SHA256Bytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// GetLocalIP 获取本机 IP 地址
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// GetLocalIPs 获取本机所有 IP 地址
func GetLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

// IsValidIP 判断是否为有效 IP
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsInternalIP 判断是否为内网 IP
func IsInternalIP(ip string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	// 10.0.0.0/8
	if ipAddr.IsPrivate() {
		return true
	}

	// 回环地址
	if ipAddr.IsLoopback() {
		return true
	}

	return false
}

// GetClientIP 从 HTTP 请求中获取客户端 IP
func GetClientIP(r *http.Request) string {
	// X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" && IsValidIP(ip) {
				return ip
			}
		}
	}

	// X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		xri = strings.TrimSpace(xri)
		if IsValidIP(xri) {
			return xri
		}
	}

	// 从 RemoteAddr 获取
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ===== 密码哈希 =====

// HashPassword 使用 bcrypt 生成密码哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 校验密码哈希
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength 校验密码强度：长度 8–20 位，须同时包含字母与数字
func ValidatePasswordStrength(password string) bool {
	if len(password) < 8 || len(password) > 20 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, ch := range password {
		switch {
		case ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z':
			hasLetter = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}
