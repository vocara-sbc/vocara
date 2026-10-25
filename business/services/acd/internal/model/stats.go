package model

import "time"

// QueueStats 队列实时统计数据
type QueueStats struct {
	// QueueID 队列 ID
	QueueID int64 `json:"queueId"`
	// QueueName 队列名称
	QueueName string `json:"queueName"`
	// TenantID 租户 ID
	TenantID int64 `json:"tenantId"`
	// WaitingCount 等待中通话数（active 队列长度）
	WaitingCount int64 `json:"waitingCount"`
	// PausedCount 暂停中通话数（paused 队列长度）
	PausedCount int64 `json:"pausedCount"`
	// ServingCount 通话中数（已分配座席）
	ServingCount int64 `json:"servingCount"`
	// TotalToday 今日总入队数
	TotalToday int64 `json:"totalToday"`
	// AnsweredToday 今日已接听数
	AnsweredToday int64 `json:"answeredToday"`
	// AbandonedToday 今日放弃数（客户挂机）
	AbandonedToday int64 `json:"abandonedToday"`
	// AvgWaitTime 今日平均等待时长（秒）
	AvgWaitTime float64 `json:"avgWaitTime"`
	// MaxWaitTime 当前队列最长等待时间（秒）
	MaxWaitTime int64 `json:"maxWaitTime"`
	// LongestCallID 当前等待最久的通话 ID
	LongestCallID string `json:"longestCallId,omitempty"`
	// SLA 服务水平（20秒内接听比例）
	SLA float64 `json:"sla"`
	// UpdateTime 统计更新时间
	UpdateTime time.Time `json:"updateTime"`
}
