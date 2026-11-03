package event

import "time"

// CDRData contains CDR event data (event type cdr.created).
//
// callctrl publishes a unified CDR after a call ends for consumption by billing, reporting, and other systems.
// Enumerated values such as call type, call status, and transfer type are transported as strings and follow the CDR contract documentation.
type CDRData struct {
	ID         uint64 `json:"id"`         // CDR primary key
	CallId     string `json:"callId"`     // Unique call ID
	TenantId   string `json:"tenantId"`   // Tenant ID
	CallType   string `json:"callType"`   // Call type: inbound | outbound
	Caller     string `json:"caller"`     // Caller number
	CallerName string `json:"callerName"` // Caller name
	Called     string `json:"called"`     // Called number
	CalledName string `json:"calledName"` // Called party name

	AgentId   string `json:"agentId"`   // Agent ID
	AgentName string `json:"agentName"` // Agent name
	Extension string `json:"extension"` // Agent extension number
	QueueId   string `json:"queueId"`   // Queue ID

	ALegId string `json:"aLegId"` // A-leg channel ID
	BLegId string `json:"bLegId"` // B-leg channel ID

	StartTime  time.Time  `json:"startTime"`  // Call start time
	AnswerTime *time.Time `json:"answerTime"` // Answer time
	EndTime    time.Time  `json:"endTime"`    // Call end time
	RingTime   *time.Time `json:"ringTime"`   // Ringing start time
	BridgeTime *time.Time `json:"bridgeTime"` // Bridge time

	RingDuration  int `json:"ringDuration"`  // Ringing duration in seconds
	TalkDuration  int `json:"talkDuration"`  // Talk duration in seconds, from bridge to hangup
	TotalDuration int `json:"totalDuration"` // Total duration in seconds, from creation to completion
	WaitDuration  int `json:"waitDuration"`  // Queue wait duration in seconds

	Status      string `json:"status"`      // Final call status
	HangupCause string `json:"hangupCause"` // Hangup cause
	HungupBy    string `json:"hungupBy"`    // Party that hung up
	Success     bool   `json:"success"`     // Whether the call was answered

	RecordFile     string `json:"recordFile"`     // Recording file path
	RecordDuration int    `json:"recordDuration"` // Recording duration in seconds

	TransferType   string `json:"transferType"`   // Transfer type
	TransferFrom   string `json:"transferFrom"`   // Transfer source
	TransferTo     string `json:"transferTo"`     // Transfer destination
	TransferResult string `json:"transferResult"` // Transfer result

	FsInstance string `json:"fsInstance"` // FreeSWITCH instance ID
	ExtraData  string `json:"extraData"`  // Extension data (JSON string)

	CreatedAt time.Time `json:"createdAt"` // Creation time
	UpdatedAt time.Time `json:"updatedAt"` // Update time
}
