package license

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// snapshotName 本地快照文件名（与 License 同目录，隐藏文件）。
const snapshotName = ".vocara.lic.snap"

// snapshotPayload 快照明文：License 原文与最后成功校验时间。
type snapshotPayload struct {
	// License License 文件原始内容
	License []byte `json:"license"`
	// LastSeen 最后成功校验时间，作为时钟回拨检测基准
	LastSeen time.Time `json:"last_seen"`
	// LicenseID 授权编号，便于核对快照归属
	LicenseID string `json:"license_id"`
}

// snapshotPath 返回快照路径：与 License 文件同目录。
func (m *Manager) snapshotPath() string {
	return filepath.Join(filepath.Dir(m.file), snapshotName)
}

// writeSnapshot 用设备特征派生的密钥加密保存 License 与当前时间。
// 快照绑本机：换机后密钥变化，快照不可解密，只能凭匹配设备的 License 运行。
func (m *Manager) writeSnapshot(licenseData []byte, licenseID string, now time.Time) error {
	raw, err := json.Marshal(&snapshotPayload{
		License:   licenseData,
		LastSeen:   now,
		LicenseID: licenseID,
	})
	if err != nil {
		return fmt.Errorf("序列化快照失败: %w", err)
	}
	sealed, err := sealGCM(raw, deriveSnapshotKey())
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.snapshotPath(), sealed, 0o600); err != nil {
		return fmt.Errorf("写入快照失败: %w", err)
	}
	return nil
}

// readSnapshot 读取并解密本机最近一次有效快照。
func (m *Manager) readSnapshot() (*snapshotPayload, error) {
	sealed, err := os.ReadFile(m.snapshotPath())
	if err != nil {
		return nil, err
	}
	raw, err := openGCM(sealed, deriveSnapshotKey())
	if err != nil {
		return nil, fmt.Errorf("快照解密失败: %w", err)
	}
	var payload snapshotPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("快照解析失败: %w", err)
	}
	if len(payload.License) == 0 {
		return nil, fmt.Errorf("快照内容为空")
	}
	return &payload, nil
}

// deriveSnapshotKey 以本机硬件指纹派生 AES-256 密钥。
func deriveSnapshotKey() []byte {
	machine := CollectFingerprint().Hash
	sum := sha256.Sum256([]byte("vocara:license:snapshot:" + machine))
	return sum[:]
}

// sealGCM 使用 AES-GCM 加密，nonce 前置到密文。
func sealGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建分组密码失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// openGCM 解密 AES-GCM 数据（nonce 前置）。
func openGCM(sealed, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return nil, fmt.Errorf("快照数据长度非法")
	}
	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
