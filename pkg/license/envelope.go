package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const algEd25519 = "Ed25519"

// b64 is the unpadded base64url encoding used by License.
var b64 = base64.RawURLEncoding

// LicenseFile is the License file envelope containing the signature algorithm, signed payload, and signature.
type LicenseFile struct {
	// Alg is the signature algorithm.
	Alg string `json:"alg"`
	// Payload is the base64url-encoded authorization claims and the signature input.
	Payload string `json:"payload"`
	// Signature is the base64url-encoded signature.
	Signature string `json:"signature"`
}

// Sign signs authorization claims with the vendor's private key and generates complete License file content.
func Sign(claims *LicenseClaims, privateKey ed25519.PrivateKey) ([]byte, error) {
	if claims == nil {
		return nil, fmt.Errorf("授权载荷不能为空")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("私钥长度非法")
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("序列化授权载荷失败: %w", err)
	}
	payload := b64.EncodeToString(raw)
	signature := ed25519.Sign(privateKey, []byte(payload))
	file := LicenseFile{
		Alg:       algEd25519,
		Payload:   payload,
		Signature: b64.EncodeToString(signature),
	}
	out, err := json.MarshalIndent(&file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化 License 文件失败: %w", err)
	}
	return out, nil
}

// Parse verifies a License file signature with the public key, parses the file, and returns the authorization claims.
// It returns an error for a signature mismatch, invalid encoding, or invalid claims.
func Parse(data []byte, publicKey ed25519.PublicKey) (*LicenseClaims, error) {
	var file LicenseFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("license 文件格式非法: %w", err)
	}
	if file.Alg != algEd25519 {
		return nil, fmt.Errorf("不支持的签名算法 %q", file.Alg)
	}
	signature, err := b64.DecodeString(file.Signature)
	if err != nil {
		return nil, fmt.Errorf("签名解码失败: %w", err)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("公钥长度非法")
	}
	if !ed25519.Verify(publicKey, []byte(file.Payload), signature) {
		return nil, fmt.Errorf("license 签名验签失败")
	}
	raw, err := b64.DecodeString(file.Payload)
	if err != nil {
		return nil, fmt.Errorf("授权载荷解码失败: %w", err)
	}
	var claims LicenseClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, fmt.Errorf("授权载荷解析失败: %w", err)
	}
	if err := claims.Validate(); err != nil {
		return nil, fmt.Errorf("授权载荷校验失败: %w", err)
	}
	return &claims, nil
}
