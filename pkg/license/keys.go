package license

import (
	"crypto/ed25519"
	"encoding/hex"
)

// defaultPublicKeyHex 内置 License 验签公钥（hex），对应厂商离线保管的 Ed25519 私钥。
const defaultPublicKeyHex = "9451922590501aea8b83208ceac0b372acc604d0f9981afd178bb27b41ecc782"

// DefaultPublicKey 返回内置的 License 验签公钥。
func DefaultPublicKey() ed25519.PublicKey {
	raw, err := hex.DecodeString(defaultPublicKeyHex)
	if err != nil {
		panic("license: 内置公钥非法")
	}
	return ed25519.PublicKey(raw)
}

// 运行期分发用的 Redis key 约定：admin 写入，其余服务启动与周期复校时读取。
const (
	// LicenseSourceKey License 原文分发 key。
	LicenseSourceKey = "vocara:license:current"
	// InstallIDSourceKey 集群安装标识分发 key。
	InstallIDSourceKey = "vocara:license:install_id"
	// LicenseChangedChannel 授权变更通知频道：admin 导入/重载后发布，其余服务订阅后即时复校。
	LicenseChangedChannel = "vocara:license:changed"
)
