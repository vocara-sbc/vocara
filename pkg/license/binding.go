package license

import "fmt"

// InstallIDProvider 提供集群安装标识。install 模式下由服务注入：
// admin 从系统表读取，其余服务使用统一配置值或经内部接口获取。
type InstallIDProvider interface {
	// InstallID 返回当前部署的安装标识
	InstallID() (string, error)
}

// InstallIDProviderFunc 将普通函数适配为 InstallIDProvider，便于服务以闭包注入安装标识来源。
type InstallIDProviderFunc func() (string, error)

// InstallID 实现 InstallIDProvider。
func (f InstallIDProviderFunc) InstallID() (string, error) {
	return f()
}

// verifyBinding 校验当前运行设备与授权绑定信息一致。
// host 比对本机硬件指纹，install 比对安装标识，none 不校验。
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

// currentFingerprint 返回当前设备指纹；指纹采集函数可替换以便测试。
func (m *Manager) currentFingerprint() string {
	if m.fingerprintFunc != nil {
		return m.fingerprintFunc()
	}
	return CollectFingerprint().Hash
}

// resolveInstallID 解析安装标识：注入的提供者优先，其次取配置值。
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

// SetInstallIDProvider 注入安装标识提供者，供 admin 等服务以系统表支撑 install 模式。
func (m *Manager) SetInstallIDProvider(provider InstallIDProvider) {
	m.mu.Lock()
	m.installProvider = provider
	m.mu.Unlock()
}
