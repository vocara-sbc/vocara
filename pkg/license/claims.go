// Package license 提供 License 授权的载荷模型、签名验签、状态判定与运行期管理。
//
// License 为离线授权文件，由厂商私钥签名，各服务内置公钥在本进程内独立验签，
// 不依赖在线授权服务。授权状态、功能模块与容量配额的判定统一以本包为入口。
package license

import (
	"fmt"
	"time"
)

// 产品与授权类型常量
const (
	// ProductVocara 产品标识
	ProductVocara = "vocara"

	// LicenseTypeFormal 正式授权
	LicenseTypeFormal = "formal"
	// LicenseTypeTrial 试用授权
	LicenseTypeTrial = "trial"

	// EditionEnterprise 企业版
	EditionEnterprise = "enterprise"
)

// 设备指纹模式
const (
	// FingerprintModeHost 绑定媒体服务器硬件指纹
	FingerprintModeHost = "host"
	// FingerprintModeInstall 绑定集群安装标识
	FingerprintModeInstall = "install"
	// FingerprintModeNone 不绑定设备（用于短期试用）
	FingerprintModeNone = "none"
)

// 到期判定常量
const (
	// DefaultGraceDays 默认到期宽限天数
	DefaultGraceDays = 7
	// ExpiringWindow 到期前进入“即将到期”的时间窗
	ExpiringWindow = 30 * 24 * time.Hour
)

// 功能模块键：未在 License 中显式声明的模块视为未授权
const (
	// ModuleRecord 通话录音
	ModuleRecord = "record"
	// ModuleSupervise 监听 / 耳语 / 强插 / 三方
	ModuleSupervise = "supervise"
	// ModulePredictiveDialer 预测式 / 预览式外呼
	ModulePredictiveDialer = "predictive_dialer"
	// ModuleReport 统计报表导出
	ModuleReport = "report"
	// ModuleOpenAPI 对外开放 API
	ModuleOpenAPI = "open_api"
	// ModuleCRMForm 动态表单 / 工单
	ModuleCRMForm = "crm_form"
)

// 容量配额键
const (
	// LimitAgents 可同时处于就绪态的座席数
	LimitAgents = "agents"
	// LimitConcurrentCalls 并发通话通道数
	LimitConcurrentCalls = "concurrent_calls"
	// LimitLines 可配置的中继线路数
	LimitLines = "lines"
	// LimitTenants 可开通租户数
	LimitTenants = "tenants"
)

// LicenseClaims 授权载荷：License 文件中被签名保护的完整内容。
type LicenseClaims struct {
	// LicenseID 授权唯一编号，用于续费与追溯
	LicenseID string `json:"license_id"`
	// Customer 被授权客户名称
	Customer string `json:"customer"`
	// Product 产品标识
	Product string `json:"product"`
	// Edition 版本
	Edition string `json:"edition"`
	// LicenseType 授权类型 formal/trial
	LicenseType string `json:"license_type"`
	// IssuedAt 签发时间
	IssuedAt time.Time `json:"issued_at"`
	// NotBefore 生效时间
	NotBefore time.Time `json:"not_before"`
	// ExpireAt 到期时间；永久授权为空
	ExpireAt *time.Time `json:"expire_at,omitempty"`
	// Perpetual 是否永久授权
	Perpetual bool `json:"perpetual,omitempty"`
	// GraceDays 到期宽限天数
	GraceDays int `json:"grace_days"`
	// Fingerprint 设备绑定信息
	Fingerprint Fingerprint `json:"fingerprint"`
	// Modules 功能模块开关
	Modules map[string]bool `json:"modules"`
	// Limits 容量配额
	Limits Limits `json:"limits"`
}

// Fingerprint 设备绑定信息。
type Fingerprint struct {
	// Mode 绑定模式 host/install/none
	Mode string `json:"mode"`
	// Values 指纹值集合
	Values []string `json:"values,omitempty"`
}

// Limits 容量配额；各字段 0 表示不限制。
type Limits struct {
	// Agents 就绪座席数上限
	Agents int `json:"agents"`
	// ConcurrentCalls 并发通话数上限
	ConcurrentCalls int `json:"concurrent_calls"`
	// Lines 中继线路数上限
	Lines int `json:"lines"`
	// Tenants 租户数上限
	Tenants int `json:"tenants"`
}

// ModuleEnabled 返回指定功能模块是否被授权；载荷或模块缺失时视为未授权。
func (c *LicenseClaims) ModuleEnabled(key string) bool {
	if c == nil || c.Modules == nil {
		return false
	}
	return c.Modules[key]
}

// Grace 返回生效的宽限天数；未配置时取默认值。
func (c *LicenseClaims) Grace() int {
	if c == nil || c.GraceDays <= 0 {
		return DefaultGraceDays
	}
	return c.GraceDays
}

// IsPerpetual 是否永久授权。
func (c *LicenseClaims) IsPerpetual() bool {
	return c != nil && (c.Perpetual || c.ExpireAt == nil)
}

// Limit 返回指定配额键的上限值；未知键返回 0（不限制）。
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

// Validate 校验授权载荷的必填项与基本合法性。
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
