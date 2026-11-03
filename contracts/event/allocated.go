// Package event defines event data structures shared across business and core boundaries.
//
// This package contains only transport contracts, has no dependencies on business or core implementations, and can be imported by both business and core modules.
package event

// AllocatedCall contains call allocation success event data (event type queue.dequeue).
//
// ACD publishes it after call traffic is allocated to an agent. callctrl subscribes to it and originates a call to the agent's extension,
// then bridges the customer call leg after the agent answers.
type AllocatedCall struct {
	CallID       string `json:"call_id"`       // Unique call ID
	QueueID      int64  `json:"queue_id"`      // Queue ID
	SkillGroupID int64  `json:"skill_group_id"` // Skill group ID
	TenantID     int64  `json:"tenant_id"`     // Tenant ID
	AgentID      string `json:"agent_id"`      // Agent ID
	DnNumber     string `json:"dn_number"`     // Agent extension number
	Caller       string `json:"caller"`        // Caller number
	Called       string `json:"called"`        // Called number
}
