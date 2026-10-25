package model

import "fmt"

// AgentStatus 座席状态枚举
type AgentStatus int

const (
	// StatusOffline 离线
	StatusOffline AgentStatus = 0
	// StatusCheckin 签入中
	StatusCheckin AgentStatus = 1
	// StatusChecked 已签入（未就绪）
	StatusChecked AgentStatus = 2
	// StatusReady 空闲就绪（可被分配）
	StatusReady AgentStatus = 3
	// StatusWaitServer 等待服务（分配后等待应答）
	StatusWaitServer AgentStatus = 4
	// StatusRinging 振铃中
	StatusRinging AgentStatus = 5
	// StatusOutbounding 外呼中
	StatusOutbounding AgentStatus = 6
	// StatusCalling 通话中
	StatusCalling AgentStatus = 7
	// StatusListening 监听中（子状态，基于 CALLING）
	StatusListening AgentStatus = 71
	// StatusThreeWay 三方通话（子状态，基于 CALLING）
	StatusThreeWay AgentStatus = 72
	// StatusAfterWork 事后处理
	StatusAfterWork AgentStatus = 8
	// StatusBusy 示忙
	StatusBusy AgentStatus = 9
)

// String 状态名称
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

// IsValid 是否有效状态
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

// CanTransition 是否可以转换到目标状态
// 状态机:
// OFFLINE → CHECKIN → CHECKED → READY ←→ BUSY
//                            ↑         ↓
//                            │    WAIT_SERVER
//                            │         ↓
//                     AFTER_WORK ← RINGING
//                            ↑         ↓
//                            └── CALLING ← OUTBOUNDING
//                           (LISTENING / THREE_WAY 子状态)
func (s AgentStatus) CanTransition(target AgentStatus) bool {
	// 相同状态（子状态切换单独处理）
	if s == target {
		return true
	}

	switch s {
	case StatusOffline:
		// 离线 → 签入中
		return target == StatusCheckin

	case StatusCheckin:
		// 签入中 → 已签入 或 离线（签入失败/取消）
		return target == StatusChecked || target == StatusOffline

	case StatusChecked:
		// 已签入 → 就绪 / 离线（签出）
		return target == StatusReady || target == StatusOffline

	case StatusReady:
		// 就绪 → 示忙 / 等待服务（被分配） / 外呼中 / 离线（签出）
		return target == StatusBusy || target == StatusWaitServer ||
			target == StatusOutbounding || target == StatusOffline

	case StatusBusy:
		// 示忙 → 就绪 / 离线（签出）
		return target == StatusReady || target == StatusOffline

	case StatusWaitServer:
		// 等待服务 → 振铃 / 就绪（分配失败回退）
		return target == StatusRinging || target == StatusReady

	case StatusRinging:
		// 振铃 → 通话中 / 事后处理（未接） / 就绪（取消振铃）
		return target == StatusCalling || target == StatusAfterWork || target == StatusReady

	case StatusOutbounding:
		// 外呼中 → 通话中 / 就绪（外呼失败）
		return target == StatusCalling || target == StatusReady

	case StatusCalling:
		// 通话中 → 事后处理 / 监听 / 三方通话 / 就绪（短通话）
		return target == StatusAfterWork || target == StatusListening ||
			target == StatusThreeWay || target == StatusReady

	case StatusListening:
		// 监听中 → 通话中（退出监听） / 事后处理
		return target == StatusCalling || target == StatusAfterWork

	case StatusThreeWay:
		// 三方通话 → 通话中（退出三方） / 事后处理
		return target == StatusCalling || target == StatusAfterWork

	case StatusAfterWork:
		// 事后处理 → 就绪 / 示忙 / 离线（签出）
		return target == StatusReady || target == StatusBusy || target == StatusOffline

	default:
		return false
	}
}

// IsIdle 是否空闲就绪（可被 ACD 分配）
func (s AgentStatus) IsIdle() bool {
	return s == StatusReady
}

// IsOnline 是否在线（非离线）
func (s AgentStatus) IsOnline() bool {
	return s != StatusOffline && s != StatusCheckin
}

// IsInCall 是否在通话相关状态
func (s AgentStatus) IsInCall() bool {
	switch s {
	case StatusWaitServer, StatusRinging, StatusOutbounding,
		StatusCalling, StatusListening, StatusThreeWay:
		return true
	default:
		return false
	}
}

// BaseStatus 获取基础状态（把子状态归一到 CALLING）
func (s AgentStatus) BaseStatus() AgentStatus {
	if s == StatusListening || s == StatusThreeWay {
		return StatusCalling
	}
	return s
}

// IsSubStatus 是否子状态
func (s AgentStatus) IsSubStatus() bool {
	return s == StatusListening || s == StatusThreeWay
}
