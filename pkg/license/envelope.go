package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const algEd25519 = "Ed25519"

// b64 License 使用的 base64url 编码（无填充）
var b64 = base64.RawURLEncoding

// LicenseFile License 文件信封：签名算法、被签名载荷与签名。
type LicenseFile struct {
	// Alg 签名算法
	Alg string `json:"alg"`
	// Payload base64url 编码的授权载荷，同时是签名的输入
	Payload string `json:"payload"`
	// Signature base64url 编码的签名
	Signature string `json:"signature"`
}

// Sign 使用厂商私钥对授权载荷签名，生成完整 License 文件内容。
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

// Parse 使用公钥验签并解析 License 文件，返回授权载荷。
// 签名不符、编码非法或载荷不合法时返回错误。
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
