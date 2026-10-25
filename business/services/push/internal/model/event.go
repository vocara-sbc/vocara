// Package model 定义推送服务的数据模型
// 包括事件类型、消息格式、连接模型等核心数据结构
package model

import "time"

// ===== 事件类型常量 =====
// 所有 WebSocket 推送事件的类型定义，前端通过 event 字段区分消息类型

const (
	// ===== 座席相关事件 =====
	// 注意：事件类型值采用点分隔格式，与 pkg/eventbus 包中的定义保持一致
	EventAgentStatusChange = "agent.status_change" // 座席状态变更
	EventAgentCheckIn      = "agent.checkin"       // 座席签入
	EventAgentCheckOut     = "agent.checkout"      // 座席签出
	EventAgentBusy         = "agent.busy"          // 座席示忙
	EventAgentReady        = "agent.ready"         // 座席示闲

	// ===== 通话相关事件 =====
	EventCallRinging     = "call.ringing"     // 来电振铃
	EventCallAnswered    = "call.answered"    // 通话接听
	EventCallBridged     = "call.bridged"     // 通话桥接
	EventCallHeld        = "call.held"        // 通话保持
	EventCallUnheld      = "call.unheld"      // 通话恢复
	EventCallHangup      = "call.hangup"      // 通话结束
	EventCallTransferred = "call.transferred" // 通话转接

	// ===== 队列/监控相关事件 =====
	EventQueueUpdate        = "queue.update"         // 队列状态更新
	EventAgentMonitorUpdate = "agent.monitor_update" // 座席监控数据更新
	EventStatsUpdate        = "stats.update"         // 统计数据更新

	// ===== 系统相关事件 =====
	EventHeartbeat    = "heartbeat"      // 心跳（无前缀，保持不变）
	EventSystemNotice = "system.notice"  // 系统通知
	EventError        = "error"          // 错误通知（无前缀，保持不变）
)

// ===== 消息协议版本 =====
const (
	// MessageVersion 当前消息协议版本
	MessageVersion = "1.0"
)

// ===== 客户端消息类型（客户端 -> 服务端）=====
const (
	ClientMsgSubscribe   = "subscribe"   // 订阅主题
	ClientMsgUnsubscribe = "unsubscribe" // 取消订阅
	ClientMsgHeartbeat   = "heartbeat"   // 心跳
	ClientMsgPong        = "pong"        // Pong 响应
	ClientMsgReady       = "ready"       // 座席示闲
	ClientMsgBusy        = "busy"        // 座席示忙
	ClientMsgSignout     = "signout"     // 座席签出
)

// ===== 用户类型 =====
const (
	UserTypeAgent   = "agent"   // 座席
	UserTypeAdmin   = "admin"   // 管理员
	UserTypeMonitor = "monitor" // 监控大屏
)

// ===== 连接状态 =====
const (
	ConnStateOnline  = "online"  // 在线
	ConnStateOffline = "offline" // 离线
)

// ===== 订阅主题前缀 =====
const (
	// TopicAgentPrefix 座席个人主题前缀，后跟 agentId
	TopicAgentPrefix = "agent:"
	// TopicTenantPrefix 租户主题前缀，后跟 tenantId
	TopicTenantPrefix = "tenant:"
	// TopicQueuePrefix 队列主题前缀，后跟 queueId
	TopicQueuePrefix = "queue:"
	// TopicMonitorPrefix 监控主题前缀
	TopicMonitorPrefix = "monitor:"
	// TopicBroadcast 广播主题
	TopicBroadcast = "broadcast"
)

// Message WebSocket 消息通用格式
// 所有服务端推送的消息都遵循此格式
type Message struct {
	Event     string      `json:"event"`      // 事件类型
	EventID   string      `json:"event_id"`   // 事件唯一ID（用于幂等和追踪）
	Timestamp int64       `json:"timestamp"`  // 事件时间戳（毫秒）
	Version   string      `json:"version"`    // 消息协议版本
	Data      interface{} `json:"data"`       // 事件数据（不同事件类型结构不同）
}

// NewMessage 创建新的消息
func NewMessage(event string, data interface{}) *Message {
	return &Message{
		Event:     event,
		EventID:   generateEventID(),
		Timestamp: time.Now().UnixMilli(),
		Version:   MessageVersion,
		Data:      data,
	}
}

// NewMessageWithID 创建带指定 ID 的消息（用于跨节点转发时保持事件ID一致）
func NewMessageWithID(event, eventID string, timestamp int64, data interface{}) *Message {
	return &Message{
		Event:     event,
		EventID:   eventID,
		Timestamp: timestamp,
		Version:   MessageVersion,
		Data:      data,
	}
}

// generateEventID 生成事件唯一ID
func generateEventID() string {
	// 使用简单的时间戳+随机数，生产环境可替换为UUID
	return "evt_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString 生成指定长度的随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[int(now>>uint(i*5))%len(letters)]
	}
	return string(b)
}

// ===== 客户端消息格式 =====

// ClientMessage 客户端发送给服务端的消息
type ClientMessage struct {
	Action string                 `json:"action"` // 操作类型: subscribe/unsubscribe/heartbeat
	Data   map[string]interface{} `json:"data"`   // 操作数据
}

// SubscribeData 订阅操作数据
type SubscribeData struct {
	Topics []string `json:"topics"` // 订阅的主题列表
}

// UnsubscribeData 取消订阅操作数据
type UnsubscribeData struct {
	Topics []string `json:"topics"` // 取消订阅的主题列表
}

// ===== 座席状态数据 =====

// AgentStatusData 座席状态变更事件数据
type AgentStatusData struct {
	AgentID     int64  `json:"agent_id"`     // 座席ID
	TenantID    int64  `json:"tenant_id"`    // 租户ID
	AgentName   string `json:"agent_name"`   // 座席名称
	Status      string `json:"status"`       // 当前状态
	StatusText  string `json:"status_text"`  // 状态描述
	PrevStatus  string `json:"prev_status"`  // 上一个状态
	Extension   string `json:"extension"`    // 分机号
	SkillGroups []int64 `json:"skill_groups"` // 技能组列表
}

// ===== 通话事件数据 =====

// CallEventData 通话事件数据
type CallEventData struct {
	CallUUID    string `json:"call_uuid"`    // 通话唯一标识
	AgentID     int64  `json:"agent_id"`     // 座席ID
	TenantID    int64  `json:"tenant_id"`    // 租户ID
	QueueID     int64  `json:"queue_id"`     // 队列ID（0表示非队列来电）
	Direction   string `json:"direction"`    // 通话方向: inbound/outbound/internal
	CallerNum   string `json:"caller_num"`   // 主叫号码
	CalledNum   string `json:"called_num"`   // 被叫号码
	ANI         string `json:"ani"`          // 主叫号码显示
	DNIS        string `json:"dnis"`         // 被叫号码显示
	StartTime   int64  `json:"start_time"`   // 通话开始时间
	AnswerTime  int64  `json:"answer_time"`  // 接听时间
	EndTime     int64  `json:"end_time"`     // 结束时间
	HangupCause string `json:"hangup_cause"` // 挂机原因
	Duration    int64  `json:"duration"`     // 通话时长（秒）
	HoldCount   int    `json:"hold_count"`   // 保持次数
}

// ===== 单通话排队事件数据 =====

// CallQueueEventData 单通话排队事件数据
// 对应 queue.enqueue / queue.dequeue / queue.abandoned / queue.timeout 事件
type CallQueueEventData struct {
	CallID    string `json:"call_id"`              // 通话ID
	QueueID   int64  `json:"queue_id"`             // 队列ID
	TenantID  int64  `json:"tenant_id"`            // 租户ID
	CallerNum string `json:"caller_num"`           // 主叫号码
	CalledNum string `json:"called_num"`           // 被叫号码
	Priority  int    `json:"priority"`             // 优先级
	Position  int    `json:"position"`             // 排队位置
	Reason    string `json:"reason,omitempty"`     // 出队原因（仅 dequeue/abandoned/timeout）
	WaitTime  int64  `json:"wait_time,omitempty"`  // 等待时间秒数（仅 dequeue/abandoned/timeout）
}

// ===== 队列整体统计数据 =====

// QueueStatusData 队列状态更新数据（队列整体统计）
// 对应 queue.stats_update 事件
type QueueStatusData struct {
	QueueID      int64   `json:"queue_id"`       // 队列ID
	TenantID     int64   `json:"tenant_id"`      // 租户ID
	QueueName    string  `json:"queue_name"`     // 队列名称
	WaitingNum   int     `json:"waiting_num"`    // 等待人数
	AgentNum     int     `json:"agent_num"`      // 队列座席数
	IdleAgentNum int     `json:"idle_agent_num"` // 空闲座席数
	BusyAgentNum int     `json:"busy_agent_num"` // 忙碌座席数
	AvgWaitTime  int64   `json:"avg_wait_time"`  // 平均等待时间（秒）
	MaxWaitTime  int64   `json:"max_wait_time"`  // 最长等待时间（秒）
	SlaLevel     float64 `json:"sla_level"`      // SLA 达标率
}

// ===== 座席监控数据 =====

// AgentMonitorData 座席监控数据
type AgentMonitorData struct {
	AgentID       int64  `json:"agent_id"`       // 座席ID
	TenantID      int64  `json:"tenant_id"`      // 租户ID
	AgentName     string `json:"agent_name"`     // 座席名称
	Status        string `json:"status"`         // 当前状态
	StatusDuration int64 `json:"status_duration"` // 当前状态持续时间（秒）
	Extension     string `json:"extension"`      // 分机号
	TodayCalls    int    `json:"today_calls"`    // 今日通话数
	TodayTalkTime int64  `json:"today_talk_time"` // 今日通话时长（秒）
	TodayAvgTalkTime int64 `json:"today_avg_talk_time"` // 今日平均通话时长（秒）
	QueueIDs      []int64 `json:"queue_ids"`     // 所属队列ID列表
}

// ===== 统计数据 =====

// StatsData 统计数据更新
type StatsData struct {
	TenantID   int64  `json:"tenant_id"`   // 租户ID
	StatsType  string `json:"stats_type"`  // 统计类型: realtime/hourly/daily
	Data       map[string]interface{} `json:"data"` // 统计数据
	UpdateTime int64  `json:"update_time"` // 更新时间
}

// ===== 系统通知数据 =====

// SystemNoticeData 系统通知数据
type SystemNoticeData struct {
	NoticeID   string `json:"notice_id"`   // 通知ID
	Type       string `json:"type"`        // 通知类型: info/warning/error/urgent
	Title      string `json:"title"`       // 通知标题
	Content    string `json:"content"`     // 通知内容
	Level      int    `json:"level"`       // 优先级 1-5
	CreateTime int64  `json:"create_time"` // 创建时间
}

// ===== 错误通知数据 =====

// ErrorData 错误通知数据
type ErrorData struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误消息
	Detail  string `json:"detail"`  // 详细信息
}

// ===== 心跳数据 =====

// HeartbeatData 心跳数据
type HeartbeatData struct {
	ServerTime int64 `json:"server_time"` // 服务端时间戳（毫秒）
	Load       int   `json:"load"`        // 服务端当前连接数
}
