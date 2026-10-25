package license

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FingerprintCollection 指纹采集结果：原始项与组合哈希。
type FingerprintCollection struct {
	// Items 采集到的原始硬件项
	Items map[string]string `json:"items"`
	// Hash 全部项组合后的 SHA-256，作为 License 绑定的机器码
	Hash string `json:"hash"`
}

// FingerprintRequest 指纹申请文件：客户提交给厂商用于签发 License。
type FingerprintRequest struct {
	// Customer 客户名称
	Customer string `json:"customer"`
	// IssuedAt 采集时间
	IssuedAt time.Time `json:"issued_at"`
	// Items 原始硬件项，便于签发侧核对
	Items map[string]string `json:"items"`
	// Hash 组合哈希（机器码）
	Hash string `json:"hash"`
}

// CollectFingerprint 采集本机指纹：跨平台基础项 + 平台特定项，组合做 SHA-256。
func CollectFingerprint() FingerprintCollection {
	items := map[string]string{}
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		items["hostname"] = hostname
	}
	items["cpu_cores"] = strconv.Itoa(runtime.NumCPU())
	if mac := firstPhysicalMAC(); mac != "" {
		items["mac"] = mac
	}
	for key, value := range collectPlatformItems() {
		if value != "" {
			items[key] = value
		}
	}
	return FingerprintCollection{Items: items, Hash: hashItems(items)}
}

// firstPhysicalMAC 返回首个在用、非环回网卡的物理地址。
func firstPhysicalMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String()
	}
	return ""
}

// hashItems 对按序拼接的硬件项做 SHA-256，保证相同输入得到相同机器码。
func hashItems(items map[string]string) string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(items[key])
		builder.WriteString(";")
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}
