package model

import "encoding/json"

// PushRequest is a push request used by the HTTP API.
// Other services use this structure when calling the push service over HTTP.

// PushToAgentRequest is a request to push to a specific agent.
type PushToAgentRequest struct {
	TenantID int64       `json:"tenant_id" binding:"required"` // Tenant ID
	AgentID  int64       `json:"agent_id" binding:"required"`  // Agent ID
	Event    string      `json:"event" binding:"required"`     // Event type
	Data     interface{} `json:"data"`                          // Event data
}

// PushToTenantRequest is a request to push to an entire tenant.
type PushToTenantRequest struct {
	TenantID int64       `json:"tenant_id" binding:"required"` // Tenant ID
	Event    string      `json:"event" binding:"required"`     // Event type
	Data     interface{} `json:"data"`                          // Event data
	// Optional: push to specific user types.
	UserTypes []string `json:"user_types,omitempty"` // User type filter
}

// PushToAllRequest is a request to broadcast to everyone.
type PushToAllRequest struct {
	Event string      `json:"event" binding:"required"` // Event type
	Data  interface{} `json:"data"`                      // Event data
}

// PushToTopicRequest is a request to push to a specific topic.
type PushToTopicRequest struct {
	Topic string      `json:"topic" binding:"required"` // Topic
	Event string      `json:"event" binding:"required"` // Event type
	Data  interface{} `json:"data"`                      // Event data
}

// PushBatchRequest is a batch push request.
type PushBatchRequest struct {
	Items []BatchPushItem `json:"items" binding:"required"` // List of push items
}

// BatchPushItem is an item in a batch push.
type BatchPushItem struct {
	TargetType string      `json:"target_type"` // Target type: agent/tenant/topic/all
	TargetID   string      `json:"target_id"`   // Target ID
	TenantID   int64       `json:"tenant_id"`   // Tenant ID when target_type is agent
	Event      string      `json:"event"`       // Event type
	Data       interface{} `json:"data"`        // Event data
}

// PushResponse is a push response.
type PushResponse struct {
	Success bool   `json:"success"` // Whether the push succeeded
	Count   int    `json:"count"`   // Number of connections pushed to
	Message string `json:"message"` // Message
}

// ===== Multi-node route messages =====

// RouteMessage is a cross-node route message.
// Nodes forward messages through NATS/Redis PubSub when the target agent is on another node.
type RouteMessage struct {
	TargetType string          `json:"target_type"` // Target type: agent/tenant/topic
	TargetID   string          `json:"target_id"`   // Target identifier
	TenantID   int64           `json:"tenant_id"`   // Tenant ID
	Event      string          `json:"event"`       // Event type
	EventID    string          `json:"event_id"`    // Event ID
	Timestamp  int64           `json:"timestamp"`   // Timestamp
	Payload    json.RawMessage `json:"payload"`     // Original message data
	FromNode   string          `json:"from_node"`   // Source node ID
}

// ===== Connection route information stored in Redis =====

// RouteInfo contains connection routing information.
type RouteInfo struct {
	AgentID    int64  `json:"agent_id"`    // Agent ID
	TenantID   int64  `json:"tenant_id"`   // Tenant ID
	NodeID     string `json:"node_id"`     // Node ID
	NodeAddr   string `json:"node_addr"`   // Node address
	ConnID     string `json:"conn_id"`     // Connection ID
	ServerTime int64  `json:"server_time"` // Registration time
}

// ===== Statistics =====

// Stats contains push service statistics.
type Stats struct {
	TotalConns     int            `json:"total_conns"`      // Total connections
	AgentConns     int            `json:"agent_conns"`      // Agent connections
	AdminConns     int            `json:"admin_conns"`      // Admin connections
	MonitorConns   int            `json:"monitor_conns"`    // Monitor connections
	TenantCount    int            `json:"tenant_count"`     // Number of tenants
	TotalMessages  int64          `json:"total_messages"`   // Total messages
	FailedMessages int64          `json:"failed_messages"`  // Failed messages
	Topics         map[string]int `json:"topics"`           // Subscriber count by topic
	Uptime         int64          `json:"uptime"`           // Uptime in seconds
}
