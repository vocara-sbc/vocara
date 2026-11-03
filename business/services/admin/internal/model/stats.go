package model

import "time"

// ===== Statistics Report Models =====

// AgentStats contains agent call traffic statistics.
type AgentStats struct {
	TenantID       uint64    `json:"tenant_id"`
	AgentID        uint64    `json:"agent_id"`
	AgentNumber       string    `json:"agent_number"`
	AgentName      string    `json:"agent_name"`
	OrgID          uint64    `json:"org_id"`
	OrgName        string    `json:"org_name"`
	StatDate       time.Time `json:"stat_date"`
	TotalCalls     int       `json:"total_calls"`      // Total calls.
	InboundCalls   int       `json:"inbound_calls"`    // Inbound calls.
	OutboundCalls  int       `json:"outbound_calls"`   // Outbound calls.
	AnsweredCalls  int       `json:"answered_calls"`   // Answered calls.
	MissedCalls    int       `json:"missed_calls"`     // Missed calls.
	TotalDuration  int       `json:"total_duration"`   // Total call duration in seconds.
	TalkDuration   int       `json:"talk_duration"`    // Talk duration in seconds, excluding ringing and waiting.
	AvgDuration    int       `json:"avg_duration"`     // Average call duration in seconds.
	MaxDuration    int       `json:"max_duration"`     // Maximum call duration in seconds.
	WorkDuration   int       `json:"work_duration"`    // Work duration in seconds.
	ReadyDuration  int       `json:"ready_duration"`   // Ready duration in seconds.
	BusyDuration   int       `json:"busy_duration"`    // Busy duration in seconds.
	RestDuration   int       `json:"rest_duration"`    // Rest duration in seconds.
	WrapUpDuration int       `json:"wrap_up_duration"` // Wrap-up duration in seconds.
	OfflineDuration int      `json:"offline_duration"` // Offline duration in seconds.
	Satisfaction   float64   `json:"satisfaction"`     // Average satisfaction.
	TransferCount  int       `json:"transfer_count"`   // Transfers.
	HoldCount      int       `json:"hold_count"`       // Holds.
	FirstCallResolution float64 `json:"first_call_resolution"` // First-call resolution rate.
}

// QueueStats contains queue statistics.
type QueueStats struct {
	TenantID       uint64    `json:"tenant_id"`
	QueueID        uint64    `json:"queue_id"`
	QueueName      string    `json:"queue_name"`
	QueueCode      string    `json:"queue_code"`
	StatDate       time.Time `json:"stat_date"`
	TotalCalls     int       `json:"total_calls"`     // Total incoming calls.
	AnsweredCalls  int       `json:"answered_calls"`  // Answered calls.
	AbandonedCalls int       `json:"abandoned_calls"` // Abandoned calls.
	OverflowCalls  int       `json:"overflow_calls"`  // Overflow calls.
	AnswerRate     float64   `json:"answer_rate"`     // Answer rate.
	AbandonRate    float64   `json:"abandon_rate"`    // Abandonment rate.
	AvgWaitTime    int       `json:"avg_wait_time"`   // Average wait time in seconds.
	MaxWaitTime    int       `json:"max_wait_time"`   // Maximum wait time in seconds.
	AvgTalkTime    int       `json:"avg_talk_time"`   // Average talk time in seconds.
	ServiceLevel20 float64   `json:"service_level_20"` // Percentage answered within 20 seconds.
	ServiceLevel30 float64   `json:"service_level_30"` // Percentage answered within 30 seconds.
	MaxWaitNum     int       `json:"max_wait_num"`    // Maximum callers waiting.
	AvgAgentReady  int       `json:"avg_agent_ready"` // Average ready agents.
}

// LineStats contains trunk line statistics.
type LineStats struct {
	TenantID      uint64    `json:"tenant_id"`
	LineID        uint64    `json:"line_id"`
	LineName      string    `json:"line_name"`
	LineCode      string    `json:"line_code"`
	StatDate      time.Time `json:"stat_date"`
	TotalCalls    int       `json:"total_calls"`   // Total calls.
	InboundCalls  int       `json:"inbound_calls"` // Inbound calls.
	OutboundCalls int       `json:"outbound_calls"`// Outbound calls.
	AnsweredCalls int       `json:"answered_calls"` // Answered calls.
	FailedCalls   int       `json:"failed_calls"`  // Failed calls.
	SuccessRate   float64   `json:"success_rate"`  // Success rate.
	AvgDuration   int       `json:"avg_duration"`  // Average duration in seconds.
	TotalDuration int       `json:"total_duration"`// Total duration in seconds.
	PeakChannels  int       `json:"peak_channels"` // Peak concurrent channels.
	AvgChannels   float64   `json:"avg_channels"`  // Average concurrent channels.
}

// RealtimeStats contains real-time statistics.
type RealtimeStats struct {
	// Agent metrics.
	TotalAgents     int `json:"total_agents"`     // Total agents.
	OnlineAgents    int `json:"online_agents"`    // Online agents.
	ReadyAgents     int `json:"ready_agents"`     // Ready agents.
	BusyAgents      int `json:"busy_agents"`      // Busy agents.
	RestAgents      int `json:"rest_agents"`      // Resting agents.
	OfflineAgents   int `json:"offline_agents"`   // Offline agents.
	WrapUpAgents    int `json:"wrap_up_agents"`   // Agents in wrap-up.

	// Queue metrics.
	TotalQueues     int `json:"total_queues"`     // Total queues.
	ActiveQueues    int `json:"active_queues"`    // Active queues.
	WaitingCalls    int `json:"waiting_calls"`    // Calls waiting.
	TalkingCalls    int `json:"talking_calls"`    // Calls in progress.

	// Line metrics.
	TotalLines      int `json:"total_lines"`      // Total lines.
	OnlineLines     int `json:"online_lines"`     // Online lines.
	UsedChannels    int `json:"used_channels"`    // Used channels.
	TotalChannels   int `json:"total_channels"`   // Total channels.

	// Today's statistics.
	TodayCalls      int `json:"today_calls"`      // Calls today.
	TodayAnswered   int `json:"today_answered"`   // Answered calls today.
	TodayMissed     int `json:"today_missed"`     // Missed calls today.
}

// ===== Common Pagination Request =====

// PageRequest contains common pagination parameters.
type PageRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// Normalize normalizes pagination parameters.
func (p *PageRequest) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 200 {
		p.PageSize = 200
	}
}
