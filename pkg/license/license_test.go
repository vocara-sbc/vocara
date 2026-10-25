package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testKeyPair 生成仅供测试使用的一次性密钥对。
func testKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成测试密钥失败: %v", err)
	}
	return pub, priv
}

// ptr 返回时间指针。
func ptr(t time.Time) *time.Time { return &t }

// sampleClaims 构造一份合法的测试授权载荷。
func sampleClaims(expire *time.Time, perpetual bool) *LicenseClaims {
	now := time.Now()
	return &LicenseClaims{
		LicenseID:   "LIC-TEST-1",
		Customer:    "测试客户",
		Product:     ProductVocara,
		Edition:     EditionEnterprise,
		LicenseType: LicenseTypeFormal,
		IssuedAt:    now,
		NotBefore:   now.Add(-time.Minute),
		ExpireAt:    expire,
		Perpetual:   perpetual,
		GraceDays:   DefaultGraceDays,
		Fingerprint: Fingerprint{Mode: FingerprintModeNone},
		Modules: map[string]bool{
			ModuleRecord:    true,
			ModuleSupervise: true,
		},
		Limits: Limits{Agents: 10, ConcurrentCalls: 20},
	}
}

func TestSignAndParse(t *testing.T) {
	pub, priv := testKeyPair(t)
	expire := time.Now().Add(365 * 24 * time.Hour)
	claims := sampleClaims(&expire, false)

	data, err := Sign(claims, priv)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	parsed, err := Parse(data, pub)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if parsed.LicenseID != claims.LicenseID || parsed.Customer != claims.Customer {
		t.Fatal("解析出的载荷与原文不一致")
	}
	if !parsed.ModuleEnabled(ModuleRecord) {
		t.Fatal("record 模块应被授权")
	}
	if parsed.ModuleEnabled(ModuleOpenAPI) {
		t.Fatal("open_api 未在载荷中声明，应未授权")
	}
	if parsed.Limit(LimitAgents) != 10 {
		t.Fatal("座席配额应为 10")
	}
}

func TestParseTamperedPayload(t *testing.T) {
	pub, priv := testKeyPair(t)
	expire := time.Now().Add(365 * 24 * time.Hour)
	raw, err := Sign(sampleClaims(&expire, false), priv)
	if err != nil {
		t.Fatal(err)
	}
	var file LicenseFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	// 改动被签名的 payload：签名输入随之改变，验签必须失败
	pb := []byte(file.Payload)
	mid := len(pb) / 2
	if pb[mid] == 'A' {
		pb[mid] = 'B'
	} else {
		pb[mid] = 'A'
	}
	file.Payload = string(pb)
	tampered, err := json.Marshal(&file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(tampered, pub); err == nil {
		t.Fatal("payload 被篡改后 Parse 应失败")
	}
}

func TestParseWithWrongKey(t *testing.T) {
	_, priv := testKeyPair(t)
	pub2, _ := testKeyPair(t)
	expire := time.Now().Add(365 * 24 * time.Hour)
	data, err := Sign(sampleClaims(&expire, false), priv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(data, pub2); err == nil {
		t.Fatal("使用不匹配的公钥验签应失败")
	}
}

func TestEvaluateStates(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		claims *LicenseClaims
		want   State
	}{
		{"永久授权", sampleClaims(nil, true), StateValid},
		{"长期有效", sampleClaims(ptr(now.AddDate(1, 0, 0)), false), StateValid},
		{"即将到期", sampleClaims(ptr(now.AddDate(0, 0, 10)), false), StateExpiring},
		{"宽限期", sampleClaims(ptr(now.AddDate(0, 0, -3)), false), StateGrace},
		{"已失效", sampleClaims(ptr(now.AddDate(0, 0, -30)), false), StateExpired},
	}
	for _, c := range cases {
		got := Evaluate(c.claims, now)
		if got.State != c.want {
			t.Errorf("%s: 期望状态 %s，实际 %s", c.name, c.want, got.State)
		}
	}
}

func TestManagerLoadFromFile(t *testing.T) {
	pub, priv := testKeyPair(t)
	expire := time.Now().Add(365 * 24 * time.Hour)
	data, err := Sign(sampleClaims(&expire, false), priv)
	if err != nil {
		t.Fatal(err)
	}
	licPath := filepath.Join(t.TempDir(), "vocara.lic")
	if err := os.WriteFile(licPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewManagerWithKey(Config{File: licPath, CheckInterval: 60}, pub)
	if err := m.Load(); err != nil {
		t.Fatalf("Manager 加载失败: %v", err)
	}
	if m.Status() != StateValid {
		t.Fatalf("期望状态 valid，实际 %s", m.Status())
	}
	if !m.OperationAllowed() {
		t.Fatal("valid 状态应允许业务操作")
	}
	if !m.ModuleEnabled(ModuleRecord) {
		t.Fatal("record 模块应可用")
	}
	if !m.LimitAllowed(LimitAgents, 9) {
		t.Fatal("当前座席 9、上限 10，应允许新增")
	}
	if m.LimitAllowed(LimitAgents, 10) {
		t.Fatal("当前座席已达上限 10，不应再允许新增")
	}
}

func TestManagerInvalidWithoutFile(t *testing.T) {
	pub, _ := testKeyPair(t)
	m := NewManagerWithKey(Config{File: filepath.Join(t.TempDir(), "missing.lic")}, pub)
	if err := m.Load(); err == nil {
		t.Fatal("文件缺失时 Load 应返回错误")
	}
	if m.OperationAllowed() {
		t.Fatal("invalid 状态不应允许业务操作")
	}
}
