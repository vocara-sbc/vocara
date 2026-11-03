package model

// QueueInfo contains basic queue information.
type QueueInfo struct {
	// QueueID is the queue ID.
	QueueID int64 `json:"queueId"`
	// QueueName is the queue name.
	QueueName string `json:"queueName"`
	// TenantID is the tenant ID.
	TenantID int64 `json:"tenantId"`
	// SkillGroupID is the associated skill group ID.
	SkillGroupID int64 `json:"skillGroupId"`
	// Strategy is the allocation strategy; it is reserved, and only priority plus wait duration is supported.
	Strategy string `json:"strategy"`
	// MaxWaitTime is the maximum wait time in seconds; 0 means unlimited.
	MaxWaitTime int `json:"maxWaitTime"`
	// MaxQueueLen is the maximum queue length; 0 means unlimited.
	MaxQueueLen int `json:"maxQueueLen"`
	// PriorityAgingCycle is the priority aging cycle in seconds, with priority increasing by one level per cycle.
	PriorityAgingCycle int `json:"priorityAgingCycle"`
}
