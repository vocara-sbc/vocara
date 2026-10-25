package utils

import "github.com/google/uuid"

// UUID 生成 UUID v4 字符串
func UUID() string {
	return uuid.New().String()
}

// UUIDShort 生成不带连字符的 UUID
func UUIDShort() string {
	u := uuid.New()
	var buf [32]byte
	for i, x := range [16]int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	} {
		buf[i*2] = hexTable[u[x]>>4]
		buf[i*2+1] = hexTable[u[x]&0x0f]
	}
	return string(buf[:])
}

// hexTable 十六进制字符表
var hexTable = []byte("0123456789abcdef")

// UUIDv1 生成基于时间的 UUID v1
func UUIDv1() (string, error) {
	u, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// IsUUID 判断字符串是否为有效的 UUID
func IsUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
