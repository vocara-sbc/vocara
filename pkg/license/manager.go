package license

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"sync"
	"time"
)

// ClockRollbackTolerance is the tolerance for detecting clock rollback: rollback is detected only when the current time is earlier than the recorded value by more than this amount.
// The tolerance accommodates minor NTP adjustments and virtual machine clock jitter; rollback beyond the tolerance is detected after expiration.
const ClockRollbackTolerance = 24 * time.Hour

// Config contains License runtime configuration mapped from each service's configuration section.
type Config struct {
	// File is the License file path.
	File string
	// CheckInterval is the runtime verification interval in seconds.
	CheckInterval int
	// FailOpen indicates whether business operations are allowed when signature, fingerprint, or other validation errors occur; use only temporarily for troubleshooting.
	FailOpen bool
	// InstallID is the cluster installation identifier in install mode.
	InstallID string
}

// Source provides the runtime source of the original License content, such as Redis distribution.
// A (nil, error) result indicates that the source is unavailable, and the caller falls back to the local file and snapshot.
type Source func() ([]byte, error)

// Manager manages License loading, caching, periodic verification, and status queries at runtime.
//
// All decisions occur within this process without depending on other online services; the last valid state is retained when verification fails.
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

// NewManager creates a License manager using the built-in default public key for signature verification.
func NewManager(cfg Config) *Manager {
	return NewManagerWithKey(cfg, DefaultPublicKey())
}

// NewManagerWithKey creates a manager using the specified public key for testing and multi-key scenarios.
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

// Load reads and validates the License file, then updates the in-memory state.
// If the file is missing or unreadable, the most recent valid snapshot is used; if neither is available, the state becomes invalid.
func (m *Manager) Load() error {
	now := time.Now()

	// Prefer the runtime source such as Redis distribution; fall back to the local file and snapshot if unavailable
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
		// Continue using the most recent valid snapshot; applyLicense performs rollback detection, binding validation, and state evaluation
		return m.applyLicense(snap.License, now)
	}
}

// ReloadFromBytes loads a License from the given content for hot updates distributed by admin through an internal API.
func (m *Manager) ReloadFromBytes(data []byte) error {
	return m.applyLicense(data, time.Now())
}

// SetSource injects the runtime License source; Load reads this source first and falls back to the file and snapshot on failure.
func (m *Manager) SetSource(s Source) {
	m.mu.Lock()
	m.source = s
	m.mu.Unlock()
}

// currentSource returns the injected runtime source under a read lock for concurrency safety with SetSource.
func (m *Manager) currentSource() Source {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.source
}

// applyLicense performs signature verification → clock rollback prevention → device binding → state evaluation → snapshot refresh.
func (m *Manager) applyLicense(data []byte, now time.Time) error {
	claims, err := Parse(data, m.publicKey)
	if err != nil {
		m.setResult(InvalidResult(err.Error()))
		return err
	}

	if !m.failOpen {
		// Prevent clock rollback using the "last seen time" recorded in the latest snapshot as the baseline
		if snap, snapErr := m.readSnapshot(); snapErr == nil && m.detectRollback(snap.LastSeen, now) {
			m.setResult(m.rollbackResult(claims))
			return fmt.Errorf("检测到系统时钟回拨")
		}

		// Validate device binding using the host fingerprint or cluster installation identifier
		if err := m.verifyBinding(claims); err != nil {
			m.setResult(InvalidResult(err.Error()))
			return err
		}
	}

	result := Evaluate(claims, now)

	// Valid states, including expiring and grace period, refresh the snapshot and last seen time to advance the rollback detection baseline;
	// expired/invalid states do not overwrite the snapshot, retaining the most recent usable state
	if result.State == StateValid || result.State == StateExpiring || result.State == StateGrace {
		_ = m.writeSnapshot(data, claims.LicenseID, now)
	}

	m.setResult(result)
	return nil
}

// Start loads the License once and then verifies it periodically in the background at the configured interval until ctx is canceled.
// An initial load failure does not interrupt the caller's startup; the service enters a restricted state, and subsequent periodic checks reassess the License.
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

// Result returns the current authorization evaluation result.
func (m *Manager) Result() Result {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.result
}

// Status returns the current authorization state.
func (m *Manager) Status() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.result.State
}

// Claims returns the current authorization claims.
func (m *Manager) Claims() *LicenseClaims {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.claims
}

// ModuleEnabled reports whether the specified feature module is currently available.
func (m *Manager) ModuleEnabled(key string) bool {
	m.mu.RLock()
	claims := m.claims
	m.mu.RUnlock()
	return claims.ModuleEnabled(key)
}

// OperationAllowed reports whether normal business operations such as outbound calls and write operations are currently allowed.
// valid/expiring/grace allow operations; expired/invalid deny them, while failOpen allows them.
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

// LimitAllowed reports whether additional usage is allowed for the specified quota at the current usage level.
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

// detectRollback reports whether the current time constitutes a rollback relative to the last recorded value.
func (m *Manager) detectRollback(lastSeen, now time.Time) bool {
	if lastSeen.IsZero() {
		return false
	}
	return now.Add(ClockRollbackTolerance).Before(lastSeen)
}

// rollbackResult constructs an invalid result caused by clock rollback, including claims for display.
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
