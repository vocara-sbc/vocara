package model

import "time"

// QueueStats contains real-time queue statistics.
type QueueStats struct {
	// QueueID is the queue ID.
	QueueID int64 `json:"queueId"`
	// QueueName is the queue name.
	QueueName string `json:"queueName"`
	// TenantID is the tenant ID.
	TenantID int64 `json:"tenantId"`
	// WaitingCount is the number of waiting calls, equal to the active queue length.
	WaitingCount int64 `json:"waitingCount"`
	// PausedCount is the number of paused calls, equal to the paused queue length.
	PausedCount int64 `json:"pausedCount"`
	// ServingCount is the number of calls assigned to agents.
	ServingCount int64 `json:"servingCount"`
	// TotalToday is the total number of calls enqueued today.
	TotalToday int64 `json:"totalToday"`
	// AnsweredToday is the number of calls answered today.
	AnsweredToday int64 `json:"answeredToday"`
	// AbandonedToday is the number of calls abandoned by customers today.
	AbandonedToday int64 `json:"abandonedToday"`
	// AvgWaitTime is today's average wait time in seconds.
	AvgWaitTime float64 `json:"avgWaitTime"`
	// MaxWaitTime is the current queue's longest wait time in seconds.
	MaxWaitTime int64 `json:"maxWaitTime"`
	// LongestCallID is the ID of the call with the longest current wait.
	LongestCallID string `json:"longestCallId,omitempty"`
	// SLA is the service level, measured as the proportion of calls answered within 20 seconds.
	SLA float64 `json:"sla"`
	// UpdateTime is the statistics update time.
	UpdateTime time.Time `json:"updateTime"`
}
