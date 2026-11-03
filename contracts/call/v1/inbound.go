// Package v1 defines the request and response contracts for the call control service HTTP API (v1).
//
// The types in this package are the authoritative definitions of callctrl's external contracts for business services and external integrations.
// They have no compile-time dependency on core implementation source code. Inbound callbacks are internal service-to-service APIs authenticated with an internal token.
package v1

// AdmitRequest is an inbound admission request (POST /api/v1/call/inbound/admit).
// The upstream call or queueing module invokes it when a customer call arrives to create a call record and admit the call to a queue.
type AdmitRequest struct {
	Caller     string `json:"caller"`     // Caller number
	CallerName string `json:"callerName"` // Caller name
	Called     string `json:"called"`     // Called number (access number)
	DNIS       string `json:"dnis"`       // Called number (DNIS)
	ANI        string `json:"ani"`        // Caller number (ANI)
	TenantId   string `json:"tenantId"`   // Tenant ID
	QueueId    string `json:"queueId"`    // Queue ID
	ALegId     string `json:"aLegId"`     // A-leg channel ID
	FsInstance string `json:"fsInstance"` // FreeSWITCH instance ID
}

// RingRequest is an agent ringing request (POST /api/v1/call/inbound/ring).
// It calls the agent's extension after an agent is allocated.
type RingRequest struct {
	CallId    string `json:"callId"`    // Unique call ID
	AgentId   string `json:"agentId"`   // Agent ID
	AgentName string `json:"agentName"` // Agent name
	Extension string `json:"extension"` // Agent extension number
	Timeout   int    `json:"timeout"`   // Ringing timeout in seconds
}

// BridgeRequest is an agent bridging request (POST /api/v1/call/inbound/bridge).
// It bridges the customer channel and agent channel after the agent answers.
type BridgeRequest struct {
	CallId string `json:"callId"` // Unique call ID
	BLegId string `json:"bLegId"` // B-leg channel ID
}

// HangupRequest is an inbound hangup request (POST /api/v1/call/inbound/hangup).
type HangupRequest struct {
	CallId string `json:"callId"` // Unique call ID
	Cause  string `json:"cause"`  // Hangup cause
}
