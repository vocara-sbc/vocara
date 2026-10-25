package model

// QueueInfo 队列基本信息
type QueueInfo struct {
	// QueueID 队列 ID
	QueueID int64 `json:"queueId"`
	// QueueName 队列名称
	QueueName string `json:"queueName"`
	// TenantID 租户 ID
	TenantID int64 `json:"tenantId"`
	// SkillGroupID 关联技能组 ID
	SkillGroupID int64 `json:"skillGroupId"`
	// Strategy 分配策略（预留，当前仅支持优先级+时长）
	Strategy string `json:"strategy"`
	// MaxWaitTime 最大等待时间（秒），0 表示不限制
	MaxWaitTime int `json:"maxWaitTime"`
	// MaxQueueLen 最大排队长度，0 表示不限制
	MaxQueueLen int `json:"maxQueueLen"`
	// PriorityAgingCycle 优先级老化周期（秒），每经过一个周期提升一级
	PriorityAgingCycle int `json:"priorityAgingCycle"`
}
