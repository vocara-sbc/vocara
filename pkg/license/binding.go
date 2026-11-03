package license

import "fmt"

// InstallIDProvider provides the cluster installation identifier. Services inject it in install mode:
// admin reads it from a system table, while other services use the shared configuration value or obtain it through an internal API.
type InstallIDProvider interface {
	// InstallID returns the installation identifier for the current deployment.
	InstallID() (string, error)
}

// InstallIDProviderFunc adapts an ordinary function to InstallIDProvider so services can inject the installation identifier source as a closure.
type InstallIDProviderFunc func() (string, error)

// InstallID implements InstallIDProvider.
func (f InstallIDProviderFunc) InstallID() (string, error) {
	return f()
}

// verifyBinding validates that the current runtime device matches the authorization binding.
// host compares the local hardware fingerprint, install compares the installation identifier, and none skips validation.
func (m *Manager) verifyBinding(claims *LicenseClaims) error {
	binding := claims.Fingerprint
	if binding.Mode == FingerprintModeNone {
		return nil
	}

	var current string
	switch binding.Mode {
	case FingerprintModeHost:
		current = m.currentFingerprint()
	case FingerprintModeInstall:
		id, err := m.resolveInstallID()
		if err != nil {
			return err
		}
		current = id
	default:
		return fmt.Errorf("非法指纹模式 %q", binding.Mode)
	}

	for _, value := range binding.Values {
		if value == current {
			return nil
		}
	}
	return fmt.Errorf("设备指纹不匹配：当前设备 %s，授权绑定 %v", current, binding.Values)
}

// currentFingerprint returns the current device fingerprint; the collection function can be replaced for testing.
func (m *Manager) currentFingerprint() string {
	if m.fingerprintFunc != nil {
		return m.fingerprintFunc()
	}
	return CollectFingerprint().Hash
}

// resolveInstallID resolves the installation identifier, preferring the injected provider and then using the configured value.
func (m *Manager) resolveInstallID() (string, error) {
	m.mu.RLock()
	provider := m.installProvider
	m.mu.RUnlock()
	if provider != nil {
		id, err := provider.InstallID()
		if err != nil {
			return "", fmt.Errorf("读取安装标识失败: %w", err)
		}
		if id != "" {
			return id, nil
		}
	}
	if m.installID != "" {
		return m.installID, nil
	}
	return "", fmt.Errorf("install 模式需要安装标识，请配置 license.install_id 或由系统提供")
}

// SetInstallIDProvider injects an installation identifier provider so services such as admin can support install mode with a system table.
func (m *Manager) SetInstallIDProvider(provider InstallIDProvider) {
	m.mu.Lock()
	m.installProvider = provider
	m.mu.Unlock()
}
