package license

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeClaims 构造带指定绑定信息的合法测试载荷（有效期一年）。
func makeClaims(fp Fingerprint) *LicenseClaims {
	now := time.Now()
	expire := now.Add(365 * 24 * time.Hour)
	return &LicenseClaims{
		LicenseID:   "LIC-TEST-BIND",
		Customer:    "测试客户",
		Product:     ProductVocara,
		Edition:     EditionEnterprise,
		LicenseType: LicenseTypeFormal,
		IssuedAt:    now,
		NotBefore:   now.Add(-time.Minute),
		ExpireAt:    &expire,
		GraceDays:   DefaultGraceDays,
		Fingerprint: fp,
		Modules:     map[string]bool{ModuleRecord: true},
	}
}

// writeSignedLicense 签发 License 并写入目录，返回路径。
func writeSignedLicense(t *testing.T, dir string, claims *LicenseClaims, priv ed25519.PrivateKey) string {
	t.Helper()
	data, err := Sign(claims, priv)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "vocara.lic")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// stubInstallProvider 固定返回安装标识的测试提供者。
type stubInstallProvider struct{ id string }

func (s stubInstallProvider) InstallID() (string, error) { return s.id, nil }

func TestBindingHostMatched(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{
		Mode:   FingerprintModeHost,
		Values: []string{"fp-host-1"},
	}), priv)

	m := NewManagerWithKey(Config{File: path}, pub)
	m.fingerprintFunc = func() string { return "fp-host-1" }
	if err := m.Load(); err != nil {
		t.Fatalf("主机指纹匹配时应加载成功: %v", err)
	}
	if m.Status() != StateValid {
		t.Fatalf("期望 valid，实际 %s", m.Status())
	}
}

func TestBindingHostMismatch(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{
		Mode:   FingerprintModeHost,
		Values: []string{"fp-other"},
	}), priv)

	m := NewManagerWithKey(Config{File: path}, pub)
	m.fingerprintFunc = func() string { return "fp-host-1" }
	if err := m.Load(); err == nil {
		t.Fatal("主机指纹不匹配时 Load 应失败")
	}
	if m.OperationAllowed() {
		t.Fatal("指纹不匹配进入 invalid，不应允许业务")
	}
}

func TestBindingInstallByConfig(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{
		Mode:   FingerprintModeInstall,
		Values: []string{"inst-1"},
	}), priv)

	m := NewManagerWithKey(Config{File: path, InstallID: "inst-1"}, pub)
	if err := m.Load(); err != nil {
		t.Fatalf("安装标识匹配时应加载成功: %v", err)
	}
	if m.Status() != StateValid {
		t.Fatalf("期望 valid，实际 %s", m.Status())
	}
}

func TestBindingInstallProviderPriority(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{
		Mode:   FingerprintModeInstall,
		Values: []string{"inst-from-provider"},
	}), priv)

	m := NewManagerWithKey(Config{File: path, InstallID: "inst-from-config"}, pub)
	m.SetInstallIDProvider(stubInstallProvider{id: "inst-from-provider"})
	if err := m.Load(); err != nil {
		t.Fatalf("提供者标识匹配时应加载成功: %v", err)
	}
}

func TestBindingInstallMissingID(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{
		Mode:   FingerprintModeInstall,
		Values: []string{"inst-1"},
	}), priv)

	m := NewManagerWithKey(Config{File: path}, pub)
	if err := m.Load(); err == nil {
		t.Fatal("install 模式无标识来源时 Load 应失败")
	}
}

func TestSnapshotFallbackWhenFileDeleted(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{Mode: FingerprintModeNone}), priv)

	m := NewManagerWithKey(Config{File: path}, pub)
	if err := m.Load(); err != nil {
		t.Fatalf("首次加载失败: %v", err)
	}

	// 删除 License 文件：以最近有效快照继续运行
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := m.Load(); err != nil {
		t.Fatalf("文件删除后应回退到快照: %v", err)
	}
	if m.Status() != StateValid {
		t.Fatalf("快照恢复后期望 valid，实际 %s", m.Status())
	}
}

func TestClockRollbackDetected(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{Mode: FingerprintModeNone}), priv)

	m := NewManagerWithKey(Config{File: path}, pub)
	if err := m.Load(); err != nil {
		t.Fatalf("首次加载失败: %v", err)
	}

	// 将快照中的"最后见到的时间"改写为 72 小时后并重新加密：当前时钟相对其构成回拨
	snap, err := m.readSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	snap.LastSeen = time.Now().Add(72 * time.Hour)
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := sealGCM(raw, deriveSnapshotKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.snapshotPath(), sealed, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := m.Load(); err == nil {
		t.Fatal("时钟回拨时 Load 应失败")
	}
	if m.Status() != StateInvalid {
		t.Fatalf("回拨后期望 invalid，实际 %s", m.Status())
	}
	if m.OperationAllowed() {
		t.Fatal("回拨态不应允许业务")
	}
}

func TestDetectRollbackUnit(t *testing.T) {
	now := time.Now()
	if !(&Manager{}).detectRollback(now.Add(72*time.Hour), now) {
		t.Fatal("早于记录值超过容差应判定回拨")
	}
	if (&Manager{}).detectRollback(now, now) {
		t.Fatal("时间一致不应判定回拨")
	}
	if (&Manager{}).detectRollback(now.Add(time.Hour), now) {
		t.Fatal("差值在容差内不应判定回拨")
	}
	if (&Manager{}).detectRollback(time.Time{}, now) {
		t.Fatal("记录值为零不应判定回拨")
	}
}

func TestFailOpenSkipsBinding(t *testing.T) {
	pub, priv := testKeyPair(t)
	dir := t.TempDir()
	path := writeSignedLicense(t, dir, makeClaims(Fingerprint{
		Mode:   FingerprintModeHost,
		Values: []string{"fp-other"},
	}), priv)

	m := NewManagerWithKey(Config{File: path, FailOpen: true}, pub)
	m.fingerprintFunc = func() string { return "fp-host-1" }
	if err := m.Load(); err != nil {
		t.Fatalf("failOpen 时应跳过绑定校验: %v", err)
	}
	if !m.OperationAllowed() {
		t.Fatal("failOpen 时应允许业务")
	}
}
