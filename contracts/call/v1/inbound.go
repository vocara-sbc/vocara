// Package v1 定义呼叫控制服务 HTTP 接口的请求/响应契约（v1 版本）。
//
// 本包类型是 callctrl 对外契约的权威定义，业务侧或外部对接系统据此开发，
// 编译期不依赖核心实现源码。呼入回调属于服务间内部接口，通过 internal token 鉴权。
package v1

// AdmitRequest 呼入接入请求（POST /api/v1/call/inbound/admit）。
// 客户呼入到达时由上游呼叫/排队模块调用，创建通话记录并接入排队。
type AdmitRequest struct {
	Caller     string `json:"caller"`     // 主叫号码
	CallerName string `json:"callerName"` // 主叫名称
	Called     string `json:"called"`     // 被叫号码（接入号）
	DNIS       string `json:"dnis"`       // 被叫号码（DNIS）
	ANI        string `json:"ani"`        // 主叫号码（ANI）
	TenantId   string `json:"tenantId"`   // 租户 ID
	QueueId    string `json:"queueId"`    // 队列 ID
	ALegId     string `json:"aLegId"`     // A 通道标识
	FsInstance string `json:"fsInstance"` // FreeSWITCH 实例标识
}

// RingRequest 座席振铃请求（POST /api/v1/call/inbound/ring）。
// 分配座席后向座席分机发起呼叫。
type RingRequest struct {
	CallId    string `json:"callId"`    // 通话唯一标识
	AgentId   string `json:"agentId"`   // 座席 ID
	AgentName string `json:"agentName"` // 座席名称
	Extension string `json:"extension"` // 座席分机号
	Timeout   int    `json:"timeout"`   // 振铃超时（秒）
}

// BridgeRequest 桥接座席请求（POST /api/v1/call/inbound/bridge）。
// 座席接听后桥接客户通道与座席通道。
type BridgeRequest struct {
	CallId string `json:"callId"` // 通话唯一标识
	BLegId string `json:"bLegId"` // B 通道标识
}

// HangupRequest 呼入挂机请求（POST /api/v1/call/inbound/hangup）。
type HangupRequest struct {
	CallId string `json:"callId"` // 通话唯一标识
	Cause  string `json:"cause"`  // 挂机原因
}
