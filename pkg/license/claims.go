// Package license provides claims models, signature verification, state evaluation, and runtime management for License authorization.
//
// A License is an offline authorization file signed with the vendor's private key. Each service independently verifies signatures within its own process using the built-in public key
// without depending on an online authorization service. This package is the unified entry point for evaluating authorization state, feature modules, and capacity quotas.
package license

import (
	"fmt"
	"time"
)

// Product and authorization type constants
const (
	// ProductVocara is the product identifier.
	ProductVocara = "vocara"

	// LicenseTypeFormal is a formal authorization.
	LicenseTypeFormal = "formal"
	// LicenseTypeTrial is a trial authorization.
	LicenseTypeTrial = "trial"

	// EditionEnterprise is the enterprise edition.
	EditionEnterprise = "enterprise"
)

// Device fingerprint modes
const (
	// FingerprintModeHost binds to the media server hardware fingerprint.
	FingerprintModeHost = "host"
	// FingerprintModeInstall binds to the cluster installation identifier.
	FingerprintModeInstall = "install"
	// FingerprintModeNone does not bind to a device and is used for short-term trials.
	FingerprintModeNone = "none"
)

// Expiration evaluation constants
const (
	// DefaultGraceDays is the default grace period in days after expiration.
	DefaultGraceDays = 7
	// ExpiringWindow is the time window before expiration during which the state becomes expiring.
	ExpiringWindow = 30 * 24 * time.Hour
)

// Feature module keys: modules not explicitly declared in the License are unauthorized
const (
	// ModuleRecord enables call recording.
	ModuleRecord = "record"
	// ModuleSupervise enables monitoring, whisper, barge-in, and three-way calls.
	ModuleSupervise = "supervise"
	// ModulePredictiveDialer enables predictive and preview outbound dialing.
	ModulePredictiveDialer = "predictive_dialer"
	// ModuleReport enables statistics report export.
	ModuleReport = "report"
	// ModuleOpenAPI enables the public API.
	ModuleOpenAPI = "open_api"
	// ModuleCRMForm enables dynamic forms and tickets.
	ModuleCRMForm = "crm_form"
)

// Capacity quota keys
const (
	// LimitAgents is the number of agents that can be ready simultaneously.
	LimitAgents = "agents"
	// LimitConcurrentCalls is the number of concurrent call channels.
	LimitConcurrentCalls = "concurrent_calls"
	// LimitLines is the number of configurable trunk lines.
	LimitLines = "lines"
	// LimitTenants is the number of tenants that can be provisioned.
	LimitTenants = "tenants"
)

// LicenseClaims contains the complete content protected by the License file signature.
type LicenseClaims struct {
	// LicenseID is the unique authorization identifier used for renewal and traceability.
	LicenseID string `json:"license_id"`
	// Customer is the licensed customer name.
	Customer string `json:"customer"`
	// Product is the product identifier.
	Product string `json:"product"`
	// Edition is the edition.
	Edition string `json:"edition"`
	// LicenseType is the authorization type: formal/trial.
	LicenseType string `json:"license_type"`
	// IssuedAt is the issuance time.
	IssuedAt time.Time `json:"issued_at"`
	// NotBefore is the activation time.
	NotBefore time.Time `json:"not_before"`
	// ExpireAt is the expiration time; empty for perpetual authorization.
	ExpireAt *time.Time `json:"expire_at,omitempty"`
	// Perpetual indicates whether the authorization is perpetual.
	Perpetual bool `json:"perpetual,omitempty"`
	// GraceDays is the grace period in days after expiration.
	GraceDays int `json:"grace_days"`
	// Fingerprint contains device binding information.
	Fingerprint Fingerprint `json:"fingerprint"`
	// Modules contains feature module switches.
	Modules map[string]bool `json:"modules"`
	// Limits contains capacity quotas.
	Limits Limits `json:"limits"`
}

// Fingerprint contains device binding information.
type Fingerprint struct {
	// Mode is the binding mode: host/install/none.
	Mode string `json:"mode"`
	// Values contains the fingerprint values.
	Values []string `json:"values,omitempty"`
}

// Limits contains capacity quotas; a value of 0 for any field means unlimited.
type Limits struct {
	// Agents is the maximum number of ready agents.
	Agents int `json:"agents"`
	// ConcurrentCalls is the maximum number of concurrent calls.
	ConcurrentCalls int `json:"concurrent_calls"`
	// Lines is the maximum number of trunk lines.
	Lines int `json:"lines"`
	// Tenants is the maximum number of tenants.
	Tenants int `json:"tenants"`
}

// ModuleEnabled reports whether the specified feature module is authorized; missing claims or modules are unauthorized.
func (c *LicenseClaims) ModuleEnabled(key string) bool {
	if c == nil || c.Modules == nil {
		return false
	}
	return c.Modules[key]
}

// Grace returns the effective grace period in days, using the default value when unconfigured.
func (c *LicenseClaims) Grace() int {
	if c == nil || c.GraceDays <= 0 {
		return DefaultGraceDays
	}
	return c.GraceDays
}

// IsPerpetual reports whether the authorization is perpetual.
func (c *LicenseClaims) IsPerpetual() bool {
	return c != nil && (c.Perpetual || c.ExpireAt == nil)
}

// Limit returns the limit for the specified quota key; unknown keys return 0, meaning unlimited.
func (c *LicenseClaims) Limit(key string) int {
	if c == nil {
		return 0
	}
	switch key {
	case LimitAgents:
		return c.Limits.Agents
	case LimitConcurrentCalls:
		return c.Limits.ConcurrentCalls
	case LimitLines:
		return c.Limits.Lines
	case LimitTenants:
		return c.Limits.Tenants
	default:
		return 0
	}
}

// Validate validates required fields and basic validity of the authorization claims.
func (c *LicenseClaims) Validate() error {
	if c.LicenseID == "" {
		return fmt.Errorf("license_id 不能为空")
	}
	if c.Customer == "" {
		return fmt.Errorf("customer 不能为空")
	}
	if c.Product == "" {
		return fmt.Errorf("product 不能为空")
	}
	switch c.LicenseType {
	case LicenseTypeFormal, LicenseTypeTrial:
	default:
		return fmt.Errorf("非法授权类型 %q", c.LicenseType)
	}
	if c.ExpireAt != nil && c.ExpireAt.Before(c.NotBefore) {
		return fmt.Errorf("到期时间不能早于生效时间")
	}
	switch c.Fingerprint.Mode {
	case FingerprintModeHost, FingerprintModeInstall, FingerprintModeNone:
	default:
		return fmt.Errorf("非法指纹模式 %q", c.Fingerprint.Mode)
	}
	return nil
}
