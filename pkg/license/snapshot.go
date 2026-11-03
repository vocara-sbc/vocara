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

// snapshotName is the local snapshot filename, a hidden file in the same directory as the License.
const snapshotName = ".vocara.lic.snap"

// snapshotPayload is the plaintext snapshot containing the original License content and the last successful verification time.
type snapshotPayload struct {
	// License is the original License file content.
	License []byte `json:"license"`
	// LastSeen is the last successful verification time, used as the baseline for clock rollback detection.
	LastSeen time.Time `json:"last_seen"`
	// LicenseID is the authorization identifier for checking snapshot ownership.
	LicenseID string `json:"license_id"`
}

// snapshotPath returns the snapshot path in the same directory as the License file.
func (m *Manager) snapshotPath() string {
	return filepath.Join(filepath.Dir(m.file), snapshotName)
}

// writeSnapshot encrypts and saves the License and current time using a key derived from device characteristics.
// The snapshot is bound to this machine: a different machine derives a different key and cannot decrypt it, so operation requires a License matching the device.
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

// readSnapshot reads and decrypts the most recent valid snapshot for this machine.
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

// deriveSnapshotKey derives an AES-256 key from the local hardware fingerprint.
func deriveSnapshotKey() []byte {
	machine := CollectFingerprint().Hash
	sum := sha256.Sum256([]byte("vocara:license:snapshot:" + machine))
	return sum[:]
}

// sealGCM encrypts with AES-GCM and prefixes the ciphertext with the nonce.
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

// openGCM decrypts AES-GCM data with a prefixed nonce.
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
