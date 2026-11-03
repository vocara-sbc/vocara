package model

import "fmt"

// AgentStatus enumerates agent states.
type AgentStatus int

const (
	// StatusOffline indicates that the agent is offline.
	StatusOffline AgentStatus = 0
	// StatusCheckin indicates that check-in is in progress.
	StatusCheckin AgentStatus = 1
	// StatusChecked indicates that the agent is checked in but not ready.
	StatusChecked AgentStatus = 2
	// StatusReady indicates that the agent is idle and available for allocation.
	StatusReady AgentStatus = 3
	// StatusWaitServer indicates that the agent is awaiting an answer after allocation.
	StatusWaitServer AgentStatus = 4
	// StatusRinging indicates that the agent is ringing.
	StatusRinging AgentStatus = 5
	// StatusOutbounding indicates that an outbound call is in progress.
	StatusOutbounding AgentStatus = 6
	// StatusCalling indicates that the agent is in a call.
	StatusCalling AgentStatus = 7
	// StatusListening indicates listen-in, a substate of CALLING.
	StatusListening AgentStatus = 71
	// StatusThreeWay indicates a three-way call, a substate of CALLING.
	StatusThreeWay AgentStatus = 72
	// StatusAfterWork indicates after-call work.
	StatusAfterWork AgentStatus = 8
	// StatusBusy indicates that the agent is busy.
	StatusBusy AgentStatus = 9
)

// String returns the state name.
func (s AgentStatus) String() string {
	switch s {
	case StatusOffline:
		return "OFFLINE"
	case StatusCheckin:
		return "CHECKIN"
	case StatusChecked:
		return "CHECKED"
	case StatusReady:
		return "READY"
	case StatusWaitServer:
		return "WAIT_SERVER"
	case StatusRinging:
		return "RINGING"
	case StatusOutbounding:
		return "OUTBOUNDING"
	case StatusCalling:
		return "CALLING"
	case StatusListening:
		return "LISTENING"
	case StatusThreeWay:
		return "THREE_WAY"
	case StatusAfterWork:
		return "AFTER_WORK"
	case StatusBusy:
		return "BUSY"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// IsValid reports whether the state is valid.
func (s AgentStatus) IsValid() bool {
	switch s {
	case StatusOffline, StatusCheckin, StatusChecked, StatusReady,
		StatusWaitServer, StatusRinging, StatusOutbounding,
		StatusCalling, StatusListening, StatusThreeWay,
		StatusAfterWork, StatusBusy:
		return true
	default:
		return false
	}
}

// CanTransition reports whether the agent can transition to the target state.
// State machine:
// OFFLINE → CHECKIN → CHECKED → READY ←→ BUSY
//                            ↑         ↓
//                            │    WAIT_SERVER
//                            │         ↓
//                     AFTER_WORK ← RINGING
//                            ↑         ↓
//                            └── CALLING ← OUTBOUNDING
//                           (LISTENING / THREE_WAY substates)
func (s AgentStatus) CanTransition(target AgentStatus) bool {
	// Same state (substate transitions are handled separately)
	if s == target {
		return true
	}

	switch s {
	case StatusOffline:
		// OFFLINE -> CHECKIN
		return target == StatusCheckin

	case StatusCheckin:
		// CHECKIN -> CHECKED or OFFLINE (check-in failed or canceled)
		return target == StatusChecked || target == StatusOffline

	case StatusChecked:
		// CHECKED -> READY or OFFLINE (checkout)
		return target == StatusReady || target == StatusOffline

	case StatusReady:
		// READY -> BUSY, WAIT_SERVER (allocated), OUTBOUNDING, or OFFLINE (checkout)
		return target == StatusBusy || target == StatusWaitServer ||
			target == StatusOutbounding || target == StatusOffline

	case StatusBusy:
		// BUSY -> READY or OFFLINE (checkout)
		return target == StatusReady || target == StatusOffline

	case StatusWaitServer:
		// WAIT_SERVER -> RINGING or READY (allocation failed)
		return target == StatusRinging || target == StatusReady

	case StatusRinging:
		// RINGING -> CALLING, AFTER_WORK (unanswered), or READY (ringing canceled)
		return target == StatusCalling || target == StatusAfterWork || target == StatusReady

	case StatusOutbounding:
		// OUTBOUNDING -> CALLING or READY (outbound call failed)
		return target == StatusCalling || target == StatusReady

	case StatusCalling:
		// CALLING -> AFTER_WORK, LISTENING, THREE_WAY, or READY (short call)
		return target == StatusAfterWork || target == StatusListening ||
			target == StatusThreeWay || target == StatusReady

	case StatusListening:
		// LISTENING -> CALLING (stop listen-in) or AFTER_WORK
		return target == StatusCalling || target == StatusAfterWork

	case StatusThreeWay:
		// THREE_WAY -> CALLING (leave three-way call) or AFTER_WORK
		return target == StatusCalling || target == StatusAfterWork

	case StatusAfterWork:
		// AFTER_WORK -> READY, BUSY, or OFFLINE (checkout)
		return target == StatusReady || target == StatusBusy || target == StatusOffline

	default:
		return false
	}
}

// IsIdle reports whether the agent is ready for ACD allocation.
func (s AgentStatus) IsIdle() bool {
	return s == StatusReady
}

// IsOnline reports whether the agent is online.
func (s AgentStatus) IsOnline() bool {
	return s != StatusOffline && s != StatusCheckin
}

// IsInCall reports whether the agent is in a call-related state.
func (s AgentStatus) IsInCall() bool {
	switch s {
	case StatusWaitServer, StatusRinging, StatusOutbounding,
		StatusCalling, StatusListening, StatusThreeWay:
		return true
	default:
		return false
	}
}

// BaseStatus returns the base state, mapping substates to CALLING.
func (s AgentStatus) BaseStatus() AgentStatus {
	if s == StatusListening || s == StatusThreeWay {
		return StatusCalling
	}
	return s
}

// IsSubStatus reports whether the state is a substate.
func (s AgentStatus) IsSubStatus() bool {
	return s == StatusListening || s == StatusThreeWay
}
