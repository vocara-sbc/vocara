package license

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"sync"
	"time"
)

// ClockRollbackTolerance 时钟回拨判定容差：当前时间早于记录值超过该幅度才判定回拨。
// 容差用于吸收 NTP 小幅校时与虚机时钟抖动；到期后回拨超过容差即被识别。
const ClockRollbackTolerance = 24 * time.Hour

// Config License 运行配置（由各服务配置节映射）。
type Config struct {
	// File License 文件路径
	File string
	// CheckInterval 运行期复校间隔（秒）
	CheckInterval int
	// FailOpen 签名 / 指纹等异常时是否放行业务（仅排障临时使用）
	FailOpen bool
	// InstallID install 模式的集群安装标识
	InstallID string
}

// Source 提供 License 原文的运行期来源（如 Redis 下发）。
// 返回 (nil, error) 表示来源不可用，调用方回退到本地文件与快照。
type Source func() ([]byte, error)

// Manager 运行期 License 管理器：加载、缓存、周期复校与状态查询。
//
// 全部判定在本进程内完成，不依赖其他在线服务；复校失败时保留上一有效状态。
type Manager struct {
	file          string
	checkInterval time.Duration
	failOpen      bool
	installID     string
	publicKey     ed25519.PublicKey

	installProvider InstallIDProvider
	source         Source
	fingerprintFunc func() string

	mu     sync.RWMutex
	claims *LicenseClaims
	result Result
}

// NewManager 创建 License 管理器，验签公钥取内置默认值。
func NewManager(cfg Config) *Manager {
	return NewManagerWithKey(cfg, DefaultPublicKey())
}

// NewManagerWithKey 创建使用指定公钥的管理器，用于测试及多密钥场景。
func NewManagerWithKey(cfg Config, publicKey ed25519.PublicKey) *Manager {
	interval := time.Duration(cfg.CheckInterval) * time.Second
	if interval <= 0 {
		interval = time.Hour
	}
	return &Manager{
		file:          cfg.File,
		checkInterval: interval,
		failOpen:      cfg.FailOpen,
		installID:     cfg.InstallID,
		publicKey:     publicKey,
		result:        InvalidResult("License 尚未加载"),
	}
}

// Load 读取 License 文件并完成校验，更新内存状态。
// 文件缺失或不可读时，以最近一次有效快照继续运行；两者皆无则进入无效态。
func (m *Manager) Load() error {
	now := time.Now()

	// 运行期来源优先（如 Redis 下发）；来源不可用时回退本地文件与快照
	if src := m.currentSource(); src != nil {
		if data, err := src(); err == nil {
			return m.applyLicense(data, now)
		}
	}

	if data, fileErr := os.ReadFile(m.file); fileErr == nil {
		return m.applyLicense(data, now)
	} else {
		snap, snapErr := m.readSnapshot()
		if snapErr != nil {
			m.setResult(InvalidResult(fmt.Sprintf("License 文件缺失且无可用快照: %v", fileErr)))
			return fileErr
		}
		// 以最近有效快照继续运行；回拨检测、绑定校验与状态评估在 applyLicense 内统一完成
		return m.applyLicense(snap.License, now)
	}
}

// ReloadFromBytes 使用给定内容加载 License，用于 admin 经内部接口下发的热更新。
func (m *Manager) ReloadFromBytes(data []byte) error {
	return m.applyLicense(data, time.Now())
}

// SetSource 注入 License 运行期来源；Load 优先读取该来源，失败时回退文件与快照。
func (m *Manager) SetSource(s Source) {
	m.mu.Lock()
	m.source = s
	m.mu.Unlock()
}

// currentSource 返回注入的运行期来源，读取前加读锁以与 SetSource 并发安全。
func (m *Manager) currentSource() Source {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.source
}

// applyLicense 验签 → 防时钟回拨 → 设备绑定 → 状态评估 → 刷新快照。
func (m *Manager) applyLicense(data []byte, now time.Time) error {
	claims, err := Parse(data, m.publicKey)
	if err != nil {
		m.setResult(InvalidResult(err.Error()))
		return err
	}

	if !m.failOpen {
		// 防时钟回拨：以最近快照记录的"最后见到的时间"为基准
		if snap, snapErr := m.readSnapshot(); snapErr == nil && m.detectRollback(snap.LastSeen, now) {
			m.setResult(m.rollbackResult(claims))
			return fmt.Errorf("检测到系统时钟回拨")
		}

		// 设备绑定校验：主机指纹或集群安装标识
		if err := m.verifyBinding(claims); err != nil {
			m.setResult(InvalidResult(err.Error()))
			return err
		}
	}

	result := Evaluate(claims, now)

	// 有效态（含即将到期与宽限期）刷新快照与最后见到的时间，使回拨检测基准持续前进；
	// expired/invalid 不覆盖快照，保留最近可用状态
	if result.State == StateValid || result.State == StateExpiring || result.State == StateGrace {
		_ = m.writeSnapshot(data, claims.LicenseID, now)
	}

	m.setResult(result)
	return nil
}

// Start 先加载一次 License，随后按复校间隔在后台周期复校，随 ctx 取消停止。
// 首次加载失败不中断调用方启动，服务进入受限态，License 修复后于下次复校恢复。
func (m *Manager) Start(ctx context.Context) {
	_ = m.Load()
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.Load()
		}
	}
}

// Result 返回当前授权评估结果。
func (m *Manager) Result() Result {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.result
}

// Status 返回当前授权状态。
func (m *Manager) Status() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.result.State
}

// Claims 返回当前授权载荷。
func (m *Manager) Claims() *LicenseClaims {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.claims
}

// ModuleEnabled 返回指定功能模块当前是否可用。
func (m *Manager) ModuleEnabled(key string) bool {
	m.mu.RLock()
	claims := m.claims
	m.mu.RUnlock()
	return claims.ModuleEnabled(key)
}

// OperationAllowed 返回当前是否允许常规业务操作（外呼、写操作等）。
// valid/expiring/grace 允许；expired/invalid 拒绝，failOpen 时放行。
func (m *Manager) OperationAllowed() bool {
	if m.failOpen {
		return true
	}
	switch m.Status() {
	case StateValid, StateExpiring, StateGrace:
		return true
	default:
		return false
	}
}

// LimitAllowed 判定指定配额在当前使用量 current 下是否还允许新增。
func (m *Manager) LimitAllowed(key string, current int) bool {
	if m.failOpen {
		return true
	}
	m.mu.RLock()
	claims := m.claims
	m.mu.RUnlock()
	if claims == nil {
		return false
	}
	max := claims.Limit(key)
	if max == 0 {
		return true
	}
	return current < max
}

// detectRollback 判定当前时间相对最后记录值是否构成回拨。
func (m *Manager) detectRollback(lastSeen, now time.Time) bool {
	if lastSeen.IsZero() {
		return false
	}
	return now.Add(ClockRollbackTolerance).Before(lastSeen)
}

// rollbackResult 构造因时钟回拨而无效的结果，携带载荷用于展示。
func (m *Manager) rollbackResult(claims *LicenseClaims) Result {
	return Result{
		State:   StateInvalid,
		Claims:  claims,
		Message: "系统时钟异常，检测到时钟回拨，请恢复正确时钟",
	}
}

func (m *Manager) setResult(result Result) {
	m.mu.Lock()
	m.result = result
	m.claims = result.Claims
	m.mu.Unlock()
}
