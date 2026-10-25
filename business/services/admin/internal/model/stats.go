package model

import "time"

// ===== 统计报表相关模型 =====

// AgentStats 座席话务统计
type AgentStats struct {
	TenantID       uint64    `json:"tenant_id"`
	AgentID        uint64    `json:"agent_id"`
	AgentNumber       string    `json:"agent_number"`
	AgentName      string    `json:"agent_name"`
	OrgID          uint64    `json:"org_id"`
	OrgName        string    `json:"org_name"`
	StatDate       time.Time `json:"stat_date"`
	TotalCalls     int       `json:"total_calls"`      // 总通话数
	InboundCalls   int       `json:"inbound_calls"`    // 呼入数
	OutboundCalls  int       `json:"outbound_calls"`   // 呼出数
	AnsweredCalls  int       `json:"answered_calls"`   // 接听数
	MissedCalls    int       `json:"missed_calls"`     // 未接数
	TotalDuration  int       `json:"total_duration"`   // 总通话时长(秒)
	TalkDuration   int       `json:"talk_duration"`    // 通话时长(秒，不含振铃/等待)
	AvgDuration    int       `json:"avg_duration"`     // 平均通话时长(秒)
	MaxDuration    int       `json:"max_duration"`     // 最大通话时长(秒)
	WorkDuration   int       `json:"work_duration"`    // 工作时长(秒)
	ReadyDuration  int       `json:"ready_duration"`   // 就绪时长(秒)
	BusyDuration   int       `json:"busy_duration"`    // 忙碌时长(秒)
	RestDuration   int       `json:"rest_duration"`    // 休息时长(秒)
	WrapUpDuration int       `json:"wrap_up_duration"` // 会后处理时长(秒)
	OfflineDuration int      `json:"offline_duration"` // 离线时长(秒)
	Satisfaction   float64   `json:"satisfaction"`     // 平均满意度
	TransferCount  int       `json:"transfer_count"`   // 转接次数
	HoldCount      int       `json:"hold_count"`       // 保持次数
	FirstCallResolution float64 `json:"first_call_resolution"` // 一次解决率
}

// QueueStats 队列统计
type QueueStats struct {
	TenantID       uint64    `json:"tenant_id"`
	QueueID        uint64    `json:"queue_id"`
	QueueName      string    `json:"queue_name"`
	QueueCode      string    `json:"queue_code"`
	StatDate       time.Time `json:"stat_date"`
	TotalCalls     int       `json:"total_calls"`     // 总来话数
	AnsweredCalls  int       `json:"answered_calls"`  // 接听数
	AbandonedCalls int       `json:"abandoned_calls"` // 放弃数
	OverflowCalls  int       `json:"overflow_calls"`  // 溢出数
	AnswerRate     float64   `json:"answer_rate"`     // 接通率
	AbandonRate    float64   `json:"abandon_rate"`    // 放弃率
	AvgWaitTime    int       `json:"avg_wait_time"`   // 平均等待时长(秒)
	MaxWaitTime    int       `json:"max_wait_time"`   // 最大等待时长(秒)
	AvgTalkTime    int       `json:"avg_talk_time"`   // 平均通话时长(秒)
	ServiceLevel20 float64   `json:"service_level_20"` // 20秒接通率
	ServiceLevel30 float64   `json:"service_level_30"` // 30秒接通率
	MaxWaitNum     int       `json:"max_wait_num"`    // 最大等待人数
	AvgAgentReady  int       `json:"avg_agent_ready"` // 平均就绪座席数
}

// LineStats 中继线路统计
type LineStats struct {
	TenantID      uint64    `json:"tenant_id"`
	LineID        uint64    `json:"line_id"`
	LineName      string    `json:"line_name"`
	LineCode      string    `json:"line_code"`
	StatDate      time.Time `json:"stat_date"`
	TotalCalls    int       `json:"total_calls"`   // 总呼叫数
	InboundCalls  int       `json:"inbound_calls"` // 呼入数
	OutboundCalls int       `json:"outbound_calls"`// 呼出数
	AnsweredCalls int       `json:"answered_calls"` // 接听数
	FailedCalls   int       `json:"failed_calls"`  // 失败数
	SuccessRate   float64   `json:"success_rate"`  // 成功率
	AvgDuration   int       `json:"avg_duration"`  // 平均时长(秒)
	TotalDuration int       `json:"total_duration"`// 总时长(秒)
	PeakChannels  int       `json:"peak_channels"` // 峰值并发数
	AvgChannels   float64   `json:"avg_channels"`  // 平均并发数
}

// RealtimeStats 实时统计
type RealtimeStats struct {
	// 座席相关
	TotalAgents     int `json:"total_agents"`     // 总座席数
	OnlineAgents    int `json:"online_agents"`    // 在线座席数
	ReadyAgents     int `json:"ready_agents"`     // 就绪座席数
	BusyAgents      int `json:"busy_agents"`      // 忙碌座席数
	RestAgents      int `json:"rest_agents"`      // 休息座席数
	OfflineAgents   int `json:"offline_agents"`   // 离线座席数
	WrapUpAgents    int `json:"wrap_up_agents"`   // 会后处理座席数

	// 队列相关
	TotalQueues     int `json:"total_queues"`     // 总队列数
	ActiveQueues    int `json:"active_queues"`    // 活跃队列数
	WaitingCalls    int `json:"waiting_calls"`    // 等待中通话数
	TalkingCalls    int `json:"talking_calls"`    // 通话中数

	// 线路相关
	TotalLines      int `json:"total_lines"`      // 总线路数
	OnlineLines     int `json:"online_lines"`     // 在线线路数
	UsedChannels    int `json:"used_channels"`    // 已用通道数
	TotalChannels   int `json:"total_channels"`   // 总通道数

	// 今日统计
	TodayCalls      int `json:"today_calls"`      // 今日呼叫数
	TodayAnswered   int `json:"today_answered"`   // 今日接听数
	TodayMissed     int `json:"today_missed"`     // 今日未接数
}

// ===== 通用分页请求 =====

// PageRequest 通用分页请求
type PageRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// Normalize 规范化分页参数
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
