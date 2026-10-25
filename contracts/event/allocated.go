// Package event 定义跨业务/核心边界共享的事件数据结构。
//
// 本包只承载传输契约，不依赖任何业务或核心实现，业务区与核心区均可引用。
package event

// AllocatedCall 通话分配成功事件数据（事件类型 queue.dequeue）。
//
// ACD 在话务分配到座席后发布；callctrl 订阅后向座席分机发起 originate，
// 座席接听后桥接客户通话腿。
type AllocatedCall struct {
	CallID       string `json:"call_id"`       // 通话唯一标识
	QueueID      int64  `json:"queue_id"`      // 队列 ID
	SkillGroupID int64  `json:"skill_group_id"` // 技能组 ID
	TenantID     int64  `json:"tenant_id"`     // 租户 ID
	AgentID      string `json:"agent_id"`      // 座席 ID
	DnNumber     string `json:"dn_number"`     // 座席分机号
	Caller       string `json:"caller"`        // 主叫号码
	Called       string `json:"called"`        // 被叫号码
}
