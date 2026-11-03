// Package model defines the data models for the push service,
// including core data structures such as event types, message formats, and connection models.
package model

import "time"

// ===== Event type constants =====
// Type definitions for all WebSocket push events; frontends use the event field to distinguish message types.

const (
	// ===== Agent events =====
	// Event type values use a dot-separated format consistent with the definitions in pkg/eventbus.
	EventAgentStatusChange = "agent.status_change" // Agent status change
	EventAgentCheckIn      = "agent.checkin"       // Agent sign-in
	EventAgentCheckOut     = "agent.checkout"      // Agent sign-out
	EventAgentBusy         = "agent.busy"          // Agent busy
	EventAgentReady        = "agent.ready"         // Agent ready

	// ===== Call events =====
	EventCallRinging     = "call.ringing"     // Incoming call ringing
	EventCallAnswered    = "call.answered"    // Call answered
	EventCallBridged     = "call.bridged"     // Call bridged
	EventCallHeld        = "call.held"        // Call held
	EventCallUnheld      = "call.unheld"      // Call resumed
	EventCallHangup      = "call.hangup"      // Call ended
	EventCallTransferred = "call.transferred" // Call transferred

	// ===== Queue and monitoring events =====
	EventQueueUpdate        = "queue.update"         // Queue status update
	EventAgentMonitorUpdate = "agent.monitor_update" // Agent monitoring data update
	EventStatsUpdate        = "stats.update"         // Statistics update

	// ===== System events =====
	EventHeartbeat    = "heartbeat"      // Heartbeat without a prefix
	EventSystemNotice = "system.notice"  // System notice
	EventError        = "error"          // Error notice without a prefix
)

// ===== Message protocol version =====
const (
	// MessageVersion is the current message protocol version.
	MessageVersion = "1.0"
)

// ===== Client message types (client -> server) =====
const (
	ClientMsgSubscribe   = "subscribe"   // Subscribe to topics
	ClientMsgUnsubscribe = "unsubscribe" // Unsubscribe
	ClientMsgHeartbeat   = "heartbeat"   // Heartbeat
	ClientMsgPong        = "pong"        // Pong response
	ClientMsgReady       = "ready"       // Agent ready
	ClientMsgBusy        = "busy"        // Agent busy
	ClientMsgSignout     = "signout"     // Agent sign-out
)

// ===== User types =====
const (
	UserTypeAgent   = "agent"   // Agent
	UserTypeAdmin   = "admin"   // Administrator
	UserTypeMonitor = "monitor" // Monitoring dashboard
)

// ===== Connection states =====
const (
	ConnStateOnline  = "online"  // Online
	ConnStateOffline = "offline" // Offline
)

// ===== Subscription topic prefixes =====
const (
	// TopicAgentPrefix is the agent personal topic prefix, followed by agentId.
	TopicAgentPrefix = "agent:"
	// TopicTenantPrefix is the tenant topic prefix, followed by tenantId.
	TopicTenantPrefix = "tenant:"
	// TopicQueuePrefix is the queue topic prefix, followed by queueId.
	TopicQueuePrefix = "queue:"
	// TopicMonitorPrefix is the monitoring topic prefix.
	TopicMonitorPrefix = "monitor:"
	// TopicBroadcast is the broadcast topic.
	TopicBroadcast = "broadcast"
)

// Message is the common WebSocket message format.
// All messages pushed by the server use this format.
type Message struct {
	Event     string      `json:"event"`      // Event type
	EventID   string      `json:"event_id"`   // Unique event ID for idempotency and tracing
	Timestamp int64       `json:"timestamp"`  // Event timestamp in milliseconds
	Version   string      `json:"version"`    // Message protocol version
	Data      interface{} `json:"data"`       // Event data with a structure specific to the event type
}

// NewMessage creates a new message.
func NewMessage(event string, data interface{}) *Message {
	return &Message{
		Event:     event,
		EventID:   generateEventID(),
		Timestamp: time.Now().UnixMilli(),
		Version:   MessageVersion,
		Data:      data,
	}
}

// NewMessageWithID creates a message with the specified ID to preserve the event ID during cross-node forwarding.
func NewMessageWithID(event, eventID string, timestamp int64, data interface{}) *Message {
	return &Message{
		Event:     event,
		EventID:   eventID,
		Timestamp: timestamp,
		Version:   MessageVersion,
		Data:      data,
	}
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	// Use a timestamp and random number; production deployments can use a UUID instead.
	return "evt_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of the specified length.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[int(now>>uint(i*5))%len(letters)]
	}
	return string(b)
}

// ===== Client message format =====

// ClientMessage is a message sent by a client to the server.
type ClientMessage struct {
	Action string                 `json:"action"` // Action type: subscribe/unsubscribe/heartbeat
	Data   map[string]interface{} `json:"data"`   // Action data
}

// SubscribeData contains subscription action data.
type SubscribeData struct {
	Topics []string `json:"topics"` // Topics to subscribe to
}

// UnsubscribeData contains unsubscription action data.
type UnsubscribeData struct {
	Topics []string `json:"topics"` // Topics to unsubscribe from
}

// ===== Agent status data =====

// AgentStatusData contains agent status change event data.
type AgentStatusData struct {
	AgentID     int64  `json:"agent_id"`     // Agent ID
	TenantID    int64  `json:"tenant_id"`    // Tenant ID
	AgentName   string `json:"agent_name"`   // Agent name
	Status      string `json:"status"`       // Current status
	StatusText  string `json:"status_text"`  // Status description
	PrevStatus  string `json:"prev_status"`  // Previous status
	Extension   string `json:"extension"`    // Extension number
	SkillGroups []int64 `json:"skill_groups"` // Skill group list
}

// ===== Call event data =====

// CallEventData contains call event data.
type CallEventData struct {
	CallUUID    string `json:"call_uuid"`    // Unique call identifier
	AgentID     int64  `json:"agent_id"`     // Agent ID
	TenantID    int64  `json:"tenant_id"`    // Tenant ID
	QueueID     int64  `json:"queue_id"`     // Queue ID; 0 indicates a call that did not arrive through a queue
	Direction   string `json:"direction"`    // Call direction: inbound/outbound/internal
	CallerNum   string `json:"caller_num"`   // Caller number
	CalledNum   string `json:"called_num"`   // Called number
	ANI         string `json:"ani"`          // Caller number display
	DNIS        string `json:"dnis"`         // Called number display
	StartTime   int64  `json:"start_time"`   // Call start time
	AnswerTime  int64  `json:"answer_time"`  // Answer time
	EndTime     int64  `json:"end_time"`     // End time
	HangupCause string `json:"hangup_cause"` // Hangup cause
	Duration    int64  `json:"duration"`     // Call duration in seconds
	HoldCount   int    `json:"hold_count"`   // Number of holds
}

// ===== Single-call queue event data =====

// CallQueueEventData contains queue event data for a single call.
// It corresponds to queue.enqueue, queue.dequeue, queue.abandoned, and queue.timeout events.
type CallQueueEventData struct {
	CallID    string `json:"call_id"`              // Call ID
	QueueID   int64  `json:"queue_id"`             // Queue ID
	TenantID  int64  `json:"tenant_id"`            // Tenant ID
	CallerNum string `json:"caller_num"`           // Caller number
	CalledNum string `json:"called_num"`           // Called number
	Priority  int    `json:"priority"`             // Priority
	Position  int    `json:"position"`             // Queue position
	Reason    string `json:"reason,omitempty"`     // Dequeue reason, only for dequeue/abandoned/timeout
	WaitTime  int64  `json:"wait_time,omitempty"`  // Wait time in seconds, only for dequeue/abandoned/timeout
}

// ===== Aggregate queue statistics =====

// QueueStatusData contains queue status update data with aggregate queue statistics.
// It corresponds to queue.stats_update events.
type QueueStatusData struct {
	QueueID      int64   `json:"queue_id"`       // Queue ID
	TenantID     int64   `json:"tenant_id"`      // Tenant ID
	QueueName    string  `json:"queue_name"`     // Queue name
	WaitingNum   int     `json:"waiting_num"`    // Number of waiting callers
	AgentNum     int     `json:"agent_num"`      // Number of agents in the queue
	IdleAgentNum int     `json:"idle_agent_num"` // Number of idle agents
	BusyAgentNum int     `json:"busy_agent_num"` // Number of busy agents
	AvgWaitTime  int64   `json:"avg_wait_time"`  // Average wait time in seconds
	MaxWaitTime  int64   `json:"max_wait_time"`  // Maximum wait time in seconds
	SlaLevel     float64 `json:"sla_level"`      // SLA attainment rate
}

// ===== Agent monitoring data =====

// AgentMonitorData contains agent monitoring data.
type AgentMonitorData struct {
	AgentID       int64  `json:"agent_id"`       // Agent ID
	TenantID      int64  `json:"tenant_id"`      // Tenant ID
	AgentName     string `json:"agent_name"`     // Agent name
	Status        string `json:"status"`         // Current status
	StatusDuration int64 `json:"status_duration"` // Duration of the current status in seconds
	Extension     string `json:"extension"`      // Extension number
	TodayCalls    int    `json:"today_calls"`    // Calls today
	TodayTalkTime int64  `json:"today_talk_time"` // Talk time today in seconds
	TodayAvgTalkTime int64 `json:"today_avg_talk_time"` // Average talk time today in seconds
	QueueIDs      []int64 `json:"queue_ids"`     // IDs of queues the agent belongs to
}

// ===== Statistics =====

// StatsData contains a statistics update.
type StatsData struct {
	TenantID   int64  `json:"tenant_id"`   // Tenant ID
	StatsType  string `json:"stats_type"`  // Statistics type: realtime/hourly/daily
	Data       map[string]interface{} `json:"data"` // Statistics data
	UpdateTime int64  `json:"update_time"` // Update time
}

// ===== System notice data =====

// SystemNoticeData contains system notice data.
type SystemNoticeData struct {
	NoticeID   string `json:"notice_id"`   // Notice ID
	Type       string `json:"type"`        // Notice type: info/warning/error/urgent
	Title      string `json:"title"`       // Notice title
	Content    string `json:"content"`     // Notice content
	Level      int    `json:"level"`       // Priority from 1 to 5
	CreateTime int64  `json:"create_time"` // Creation time
}

// ===== Error notice data =====

// ErrorData contains error notice data.
type ErrorData struct {
	Code    int    `json:"code"`    // Error code
	Message string `json:"message"` // Error message
	Detail  string `json:"detail"`  // Detailed information
}

// ===== Heartbeat data =====

// HeartbeatData contains heartbeat data.
type HeartbeatData struct {
	ServerTime int64 `json:"server_time"` // Server timestamp in milliseconds
	Load       int   `json:"load"`        // Current server connection count
}
