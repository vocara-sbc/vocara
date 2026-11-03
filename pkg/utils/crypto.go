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

// MD5 computes the MD5 hash of a string.
func MD5(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// MD5Bytes computes the MD5 hash of a byte slice.
func MD5Bytes(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])
}

// SHA256 computes the SHA256 hash of a string.
func SHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// SHA256Bytes computes the SHA256 hash of a byte slice.
func SHA256Bytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// GetLocalIP returns the local IP address.
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

// GetLocalIPs returns all local IP addresses.
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

// IsValidIP reports whether an IP address is valid.
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsInternalIP reports whether an IP address is internal.
func IsInternalIP(ip string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	// 10.0.0.0/8
	if ipAddr.IsPrivate() {
		return true
	}

	// Loopback address
	if ipAddr.IsLoopback() {
		return true
	}

	return false
}

// GetClientIP gets the client IP from an HTTP request.
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

	// Get it from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ===== Password hashing =====

// HashPassword generates a password hash using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword validates a password hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength validates password strength: 8–20 characters containing both letters and digits.
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
