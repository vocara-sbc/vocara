// Package model 定义 ACD 服务的数据模型
package model

import "time"

// Priority 通话优先级
type Priority int

const (
	// PriorityLow 低优先级
	PriorityLow Priority = 1
	// PriorityNormal 普通优先级（默认）
	PriorityNormal Priority = 2
	// PriorityVIP VIP 优先级
	PriorityVIP Priority = 3
)

// String 返回优先级的字符串表示
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityVIP:
		return "vip"
	default:
		return "unknown"
	}
}

// CallStatus 通话在队列中的状态
type CallStatus string

const (
	// CallStatusWaiting 等待分配中
	CallStatusWaiting CallStatus = "waiting"
	// CallStatusPaused 暂停分配（IVR 中、已分配待确认等）
	CallStatusPaused CallStatus = "paused"
	// CallStatusServing 已分配座席，通话中
	CallStatusServing CallStatus = "serving"
	// CallStatusFinished 已结束
	CallStatusFinished CallStatus = "finished"
)

// CallInfo 排队通话信息
type CallInfo struct {
	// CallID 通话唯一标识
	CallID string `json:"callId"`
	// Caller 主叫号码
	Caller string `json:"caller"`
	// Called 被叫号码
	Called string `json:"called"`
	// QueueID 队列 ID
	QueueID int64 `json:"queueId"`
	// SkillGroupID 技能组 ID
	SkillGroupID int64 `json:"skillGroupId"`
	// Priority 优先级
	Priority Priority `json:"priority"`
	// CurrentPriority 当前实际优先级（考虑老化提升后）
	CurrentPriority Priority `json:"currentPriority"`
	// Status 通话状态
	Status CallStatus `json:"status"`
	// EQTime 入队时间
	EQTime time.Time `json:"eqTime"`
	// PauseTime 暂停时间（暂停状态下有效）
	PauseTime time.Time `json:"pauseTime,omitempty"`
	// AllocateTime 分配时间（已分配状态下有效）
	AllocateTime time.Time `json:"allocateTime,omitempty"`
	// AgentID 分配的座席 ID
	AgentID int64 `json:"agentId,omitempty"`
	// TenantID 租户 ID
	TenantID int64 `json:"tenantId"`
	// CustomerInfo 客户信息（JSON 字符串，便于扩展）
	CustomerInfo string `json:"customerInfo,omitempty"`
	// UUI 随路数据（User-to-User Information）
	UUI string `json:"uui,omitempty"`
}

// WaitSeconds 计算等待时长（秒）
func (c *CallInfo) WaitSeconds() int64 {
	if c.Status == CallStatusWaiting || c.Status == CallStatusPaused {
		return int64(time.Since(c.EQTime).Seconds())
	}
	if c.AllocateTime.After(c.EQTime) {
		return int64(c.AllocateTime.Sub(c.EQTime).Seconds())
	}
	return 0
}
