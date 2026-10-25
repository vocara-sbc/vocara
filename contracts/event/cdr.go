package event

import "time"

// CDRData 话单事件数据（事件类型 cdr.created）。
//
// callctrl 在通话结束后发布统一话单，供计费、报表等系统消费。
// 呼叫类型、通话状态、转接类型等枚举值以字符串形式传输，取值口径与话单契约文档一致。
type CDRData struct {
	ID         uint64 `json:"id"`         // 话单主键
	CallId     string `json:"callId"`     // 通话唯一标识
	TenantId   string `json:"tenantId"`   // 租户 ID
	CallType   string `json:"callType"`   // 呼叫类型：inbound | outbound
	Caller     string `json:"caller"`     // 主叫号码
	CallerName string `json:"callerName"` // 主叫名称
	Called     string `json:"called"`     // 被叫号码
	CalledName string `json:"calledName"` // 被叫名称

	AgentId   string `json:"agentId"`   // 座席 ID
	AgentName string `json:"agentName"` // 座席名称
	Extension string `json:"extension"` // 座席分机号
	QueueId   string `json:"queueId"`   // 队列 ID

	ALegId string `json:"aLegId"` // A 通道标识
	BLegId string `json:"bLegId"` // B 通道标识

	StartTime  time.Time  `json:"startTime"`  // 通话开始时间
	AnswerTime *time.Time `json:"answerTime"` // 接听时间
	EndTime    time.Time  `json:"endTime"`    // 通话结束时间
	RingTime   *time.Time `json:"ringTime"`   // 振铃开始时间
	BridgeTime *time.Time `json:"bridgeTime"` // 桥接时间

	RingDuration  int `json:"ringDuration"`  // 振铃时长（秒）
	TalkDuration  int `json:"talkDuration"`  // 通话时长（秒，从桥接到挂断）
	TotalDuration int `json:"totalDuration"` // 总时长（秒，从创建到结束）
	WaitDuration  int `json:"waitDuration"`  // 排队等待时长（秒）

	Status      string `json:"status"`      // 通话最终状态
	HangupCause string `json:"hangupCause"` // 挂机原因
	HungupBy    string `json:"hungupBy"`    // 挂机方
	Success     bool   `json:"success"`     // 是否接通

	RecordFile     string `json:"recordFile"`     // 录音文件路径
	RecordDuration int    `json:"recordDuration"` // 录音时长（秒）

	TransferType   string `json:"transferType"`   // 转接类型
	TransferFrom   string `json:"transferFrom"`   // 转接来源
	TransferTo     string `json:"transferTo"`     // 转接目标
	TransferResult string `json:"transferResult"` // 转接结果

	FsInstance string `json:"fsInstance"` // FreeSWITCH 实例标识
	ExtraData  string `json:"extraData"`  // 扩展数据（JSON 字符串）

	CreatedAt time.Time `json:"createdAt"` // 创建时间
	UpdatedAt time.Time `json:"updatedAt"` // 更新时间
}
