// Package model defines the ACD service data models.
package model

import "time"

// Priority represents call priority.
type Priority int

const (
	// PriorityLow indicates low priority.
	PriorityLow Priority = 1
	// PriorityNormal indicates the default normal priority.
	PriorityNormal Priority = 2
	// PriorityVIP indicates VIP priority.
	PriorityVIP Priority = 3
)

// String returns the string representation of the priority.
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

// CallStatus represents a call's state in the queue.
type CallStatus string

const (
	// CallStatusWaiting indicates that the call is waiting for allocation.
	CallStatusWaiting CallStatus = "waiting"
	// CallStatusPaused indicates that allocation is paused, such as during IVR or while awaiting allocation confirmation.
	CallStatusPaused CallStatus = "paused"
	// CallStatusServing indicates that the call is assigned to an agent and in progress.
	CallStatusServing CallStatus = "serving"
	// CallStatusFinished indicates that the call has ended.
	CallStatusFinished CallStatus = "finished"
)

// CallInfo contains information about a queued call.
type CallInfo struct {
	// CallID is the unique call identifier.
	CallID string `json:"callId"`
	// Caller is the calling number.
	Caller string `json:"caller"`
	// Called is the called number.
	Called string `json:"called"`
	// QueueID is the queue ID.
	QueueID int64 `json:"queueId"`
	// SkillGroupID is the skill group ID.
	SkillGroupID int64 `json:"skillGroupId"`
	// Priority is the call priority.
	Priority Priority `json:"priority"`
	// CurrentPriority is the effective priority after aging increases.
	CurrentPriority Priority `json:"currentPriority"`
	// Status is the call status.
	Status CallStatus `json:"status"`
	// EQTime is the enqueue time.
	EQTime time.Time `json:"eqTime"`
	// PauseTime is the pause time, valid in the paused state.
	PauseTime time.Time `json:"pauseTime,omitempty"`
	// AllocateTime is the allocation time, valid in the allocated state.
	AllocateTime time.Time `json:"allocateTime,omitempty"`
	// AgentID is the allocated agent ID.
	AgentID int64 `json:"agentId,omitempty"`
	// TenantID is the tenant ID.
	TenantID int64 `json:"tenantId"`
	// CustomerInfo contains extensible customer information as a JSON string.
	CustomerInfo string `json:"customerInfo,omitempty"`
	// UUI contains User-to-User Information associated with the call.
	UUI string `json:"uui,omitempty"`
}

// WaitSeconds calculates the wait duration in seconds.
func (c *CallInfo) WaitSeconds() int64 {
	if c.Status == CallStatusWaiting || c.Status == CallStatusPaused {
		return int64(time.Since(c.EQTime).Seconds())
	}
	if c.AllocateTime.After(c.EQTime) {
		return int64(c.AllocateTime.Sub(c.EQTime).Seconds())
	}
	return 0
}
