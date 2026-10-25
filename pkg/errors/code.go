package errors

// 错误码命名规范：
//   0        - 成功
//   4xxyy    - 客户端错误（4xx 为 HTTP 状态码，yy 为细分码）
//   5xxyy    - 服务端错误
//   100xx    - 座席相关
//   200xx    - 队列相关
//   300xx    - 通话相关
//   400xx    - 技能组相关
//   500xx    - 系统/基础设施相关

// ===== 成功 =====
const (
	CodeSuccess = 0
	MsgSuccess  = "success"
)

// ===== 系统通用错误 =====
const (
	// 系统错误 500xx
	CodeSystemError     = 50000
	MsgSystemError      = "系统错误"
	CodeServiceError    = 50001
	MsgServiceError     = "服务异常"
	CodeDBError         = 50002
	MsgDBError          = "数据库错误"
	CodeRedisError      = 50003
	MsgRedisError       = "Redis 错误"
	CodeNetworkError    = 50004
	MsgNetworkError     = "网络错误"
	CodeTimeoutError    = 50005
	MsgTimeoutError     = "请求超时"
	CodeConfigError     = 50006
	MsgConfigError      = "配置错误"
)

// ===== 客户端错误 =====
const (
	// 参数错误 400xx
	CodeParamError    = 40000
	MsgParamError     = "参数错误"
	CodeParamInvalid  = 40001
	MsgParamInvalid   = "参数无效"
	CodeParamMissing  = 40002
	MsgParamMissing   = "参数缺失"
	CodeParamFormat   = 40003
	MsgParamFormat    = "参数格式错误"
	CodeParamTooLong  = 40004
	MsgParamTooLong   = "参数过长"

	// 未授权 401xx
	CodeUnauthorized   = 40100
	MsgUnauthorized    = "未授权"
	CodeTokenInvalid   = 40101
	MsgTokenInvalid    = "Token 无效"
	CodeTokenExpired   = 40102
	MsgTokenExpired    = "Token 已过期"
	CodeTokenMissing   = 40103
	MsgTokenMissing    = "Token 缺失"
	CodeLoginRequired  = 40104
	MsgLoginRequired   = "请先登录"
	CodeSessionExpired = 40105
	MsgSessionExpired  = "会话已过期"

	// 禁止访问 403xx
	CodeForbidden    = 40300
	MsgForbidden     = "禁止访问"
	CodeNoPermission = 40301
	MsgNoPermission  = "无权限"
	CodeAccessDenied = 40302
	MsgAccessDenied  = "访问被拒绝"

	// 资源不存在 404xx
	CodeNotFound    = 40400
	MsgNotFound     = "资源不存在"
	CodeUserNotFound = 40401
	MsgUserNotFound  = "用户不存在"

	// 冲突 409xx
	CodeConflict       = 40900
	MsgConflict        = "资源冲突"
	CodeAlreadyExists  = 40901
	MsgAlreadyExists   = "资源已存在"

	// 请求过多 429xx
	CodeTooManyRequests = 42900
	MsgTooManyRequests  = "请求过于频繁"
	CodeRateLimit       = 42901
	MsgRateLimit        = "触发限流"
)

// ===== 座席相关错误 100xx =====
const (
	CodeAgentNotFound          = 10000
	MsgAgentNotFound           = "座席不存在"
	CodeAgentNotCheckedIn      = 10001
	MsgAgentNotCheckedIn       = "座席未签入"
	CodeAgentAlreadyCheckedIn  = 10002
	MsgAgentAlreadyCheckedIn   = "座席已签入"
	CodeAgentBusy              = 10003
	MsgAgentBusy               = "座席忙"
	CodeAgentOffline           = 10004
	MsgAgentOffline            = "座席已离线"
	CodeAgentStatusInvalid     = 10005
	MsgAgentStatusInvalid      = "座席状态无效"
	CodeAgentPasswordWrong     = 10006
	MsgAgentPasswordWrong      = "密码错误"
	CodeAgentAlreadyOnline     = 10007
	MsgAgentAlreadyOnline      = "座席已在线"
	CodeAgentNotReady          = 10008
	MsgAgentNotReady           = "座席未就绪"
	CodeAgentStateTransition   = 10009
	MsgAgentStateTransition    = "状态转换不合法"
	CodeAgentNoIdle            = 10010
	MsgAgentNoIdle             = "无空闲座席"
	CodeDNNotFound             = 10011
	MsgDNNotFound              = "分机不存在"
	CodeDNNotAvailable         = 10012
	MsgDNNotAvailable          = "分机不可用"
	CodeAcquireLockFail        = 10013
	MsgAcquireLockFail         = "获取分布式锁失败"
)

// ===== 队列相关错误 200xx =====
const (
	CodeQueueNotFound        = 20001
	MsgQueueNotFound         = "队列不存在"
	CodeQueueFull            = 20002
	MsgQueueFull             = "队列已满"
	CodeQueueEmpty           = 20003
	MsgQueueEmpty            = "队列为空"
	CodeQueueNotAvailable    = 20004
	MsgQueueNotAvailable     = "队列不可用"
	CodeQueueAlreadyExists   = 20005
	MsgQueueAlreadyExists    = "队列已存在"
	CodeSkillNotFound        = 20006
	MsgSkillNotFound         = "技能组不存在"
	CodeSkillAlreadyExists   = 20007
	MsgSkillAlreadyExists    = "技能组已存在"
)

// ===== 通话相关错误 300xx =====
const (
	CodeCallNotFound       = 30001
	MsgCallNotFound        = "通话不存在"
	CodeCallAlreadyExists  = 30002
	MsgCallAlreadyExists   = "通话已存在"
	CodeCallEnded          = 30003
	MsgCallEnded           = "通话已结束"
	CodeCallStateError     = 30004
	MsgCallStateError      = "通话状态错误"
	CodeCallBridgeFail     = 30005
	MsgCallBridgeFail      = "通话桥接失败"
	CodeCallHangup         = 30006
	MsgCallHangup          = "通话已挂断"
	CodeCallHoldFail       = 30007
	MsgCallHoldFail        = "通话保持失败"
	CodeCallTransferFail   = 30008
	MsgCallTransferFail    = "通话转移失败"
	CodeCallOriginateFail  = 30009
	MsgCallOriginateFail   = "外呼失败"
	CodeCallRecordFail     = 30010
	MsgCallRecordFail      = "录音失败"
)

// ===== ESL 相关错误 350xx =====
const (
	CodeESLNotConnected   = 35001
	MsgESLNotConnected    = "ESL 未连接"
	CodeESLConnectFail    = 35002
	MsgESLConnectFail     = "ESL 连接失败"
	CodeESLAuthFail       = 35003
	MsgESLAuthFail        = "ESL 认证失败"
	CodeESLCommandFail    = 35004
	MsgESLCommandFail     = "ESL 命令执行失败"
	CodeESLTimeout        = 35005
	MsgESLTimeout         = "ESL 命令超时"
)

// ===== NATS 消息相关错误 600xx =====
const (
	CodeNATSNotConnected = 60001
	MsgNATSNotConnected  = "NATS 未连接"
	CodeNATSPublishFail  = 60002
	MsgNATSPublishFail   = "NATS 发布失败"
	CodeNATSSubscribeFail = 60003
	MsgNATSSubscribeFail  = "NATS 订阅失败"
)

// ===== License 授权相关错误 700xx =====
const (
	// CodeLicenseMissing License 文件缺失
	CodeLicenseMissing = 70001
	MsgLicenseMissing  = "License 文件缺失"
	// CodeLicenseInvalid License 无效：签名或格式错误
	CodeLicenseInvalid = 70002
	MsgLicenseInvalid  = "License 无效"
	// CodeLicenseExpired License 已到期且超过宽限期
	CodeLicenseExpired = 70003
	MsgLicenseExpired  = "License 已到期"
	// CodeLicenseFingerprintMismatch 设备指纹与授权不匹配
	CodeLicenseFingerprintMismatch = 70004
	MsgLicenseFingerprintMismatch  = "设备指纹不匹配"
	// CodeLicenseLimitExceeded 容量配额超限
	CodeLicenseLimitExceeded = 70005
	MsgLicenseLimitExceeded  = "容量超限"
	// CodeLicenseModuleDisabled 功能模块未授权
	CodeLicenseModuleDisabled = 70006
	MsgLicenseModuleDisabled  = "功能模块未授权"
	// CodeLicenseClockRollback 检测到系统时钟回拨
	CodeLicenseClockRollback = 70007
	MsgLicenseClockRollback  = "系统时钟异常"
)

// ===== 预定义错误实例 =====
var (
	// 成功
	ErrSuccess = NewBizError(CodeSuccess, MsgSuccess)

	// 系统错误
	ErrSystemError  = NewBizError(CodeSystemError, MsgSystemError)
	ErrServiceError = NewBizError(CodeServiceError, MsgServiceError)
	ErrDBError      = NewBizError(CodeDBError, MsgDBError)
	ErrRedisError   = NewBizError(CodeRedisError, MsgRedisError)
	ErrTimeoutError = NewBizError(CodeTimeoutError, MsgTimeoutError)

	// 参数错误
	ErrParamError   = NewBizError(CodeParamError, MsgParamError)
	ErrParamInvalid = NewBizError(CodeParamInvalid, MsgParamInvalid)
	ErrParamMissing = NewBizError(CodeParamMissing, MsgParamMissing)

	// 未授权
	ErrUnauthorized  = NewBizError(CodeUnauthorized, MsgUnauthorized)
	ErrTokenInvalid  = NewBizError(CodeTokenInvalid, MsgTokenInvalid)
	ErrTokenExpired  = NewBizError(CodeTokenExpired, MsgTokenExpired)
	ErrTokenMissing  = NewBizError(CodeTokenMissing, MsgTokenMissing)
	ErrLoginRequired = NewBizError(CodeLoginRequired, MsgLoginRequired)
	ErrSessionExpired = NewBizError(CodeSessionExpired, MsgSessionExpired)

	// 禁止访问
	ErrForbidden    = NewBizError(CodeForbidden, MsgForbidden)
	ErrNoPermission = NewBizError(CodeNoPermission, MsgNoPermission)

	// 资源不存在
	ErrNotFound     = NewBizError(CodeNotFound, MsgNotFound)
	ErrUserNotFound = NewBizError(CodeUserNotFound, MsgUserNotFound)

	// 冲突
	ErrConflict      = NewBizError(CodeConflict, MsgConflict)
	ErrAlreadyExists = NewBizError(CodeAlreadyExists, MsgAlreadyExists)

	// 限流
	ErrTooManyRequests = NewBizError(CodeTooManyRequests, MsgTooManyRequests)
	ErrRateLimit       = NewBizError(CodeRateLimit, MsgRateLimit)

	// 座席相关
	ErrAgentNotFound        = NewBizError(CodeAgentNotFound, MsgAgentNotFound)
	ErrAgentNotCheckedIn    = NewBizError(CodeAgentNotCheckedIn, MsgAgentNotCheckedIn)
	ErrAgentAlreadyCheckedIn = NewBizError(CodeAgentAlreadyCheckedIn, MsgAgentAlreadyCheckedIn)
	ErrAgentBusy            = NewBizError(CodeAgentBusy, MsgAgentBusy)
	ErrAgentOffline         = NewBizError(CodeAgentOffline, MsgAgentOffline)
	ErrAgentPasswordWrong   = NewBizError(CodeAgentPasswordWrong, MsgAgentPasswordWrong)
	ErrAgentStatusInvalid   = NewBizError(CodeAgentStatusInvalid, MsgAgentStatusInvalid)
	ErrAgentStateTransition = NewBizError(CodeAgentStateTransition, MsgAgentStateTransition)
	ErrAgentNoIdle          = NewBizError(CodeAgentNoIdle, MsgAgentNoIdle)
	ErrDNNotFound           = NewBizError(CodeDNNotFound, MsgDNNotFound)
	ErrDNNotAvailable       = NewBizError(CodeDNNotAvailable, MsgDNNotAvailable)
	ErrAcquireLockFail      = NewBizError(CodeAcquireLockFail, MsgAcquireLockFail)

	// 队列相关
	ErrQueueNotFound      = NewBizError(CodeQueueNotFound, MsgQueueNotFound)
	ErrQueueFull          = NewBizError(CodeQueueFull, MsgQueueFull)
	ErrQueueEmpty         = NewBizError(CodeQueueEmpty, MsgQueueEmpty)
	ErrQueueNotAvailable  = NewBizError(CodeQueueNotAvailable, MsgQueueNotAvailable)
	ErrQueueAlreadyExists = NewBizError(CodeQueueAlreadyExists, MsgQueueAlreadyExists)
	ErrSkillNotFound      = NewBizError(CodeSkillNotFound, MsgSkillNotFound)

	// 通话相关
	ErrCallNotFound      = NewBizError(CodeCallNotFound, MsgCallNotFound)
	ErrCallAlreadyExists = NewBizError(CodeCallAlreadyExists, MsgCallAlreadyExists)
	ErrCallEnded         = NewBizError(CodeCallEnded, MsgCallEnded)
	ErrCallStateError    = NewBizError(CodeCallStateError, MsgCallStateError)
	ErrCallBridgeFail    = NewBizError(CodeCallBridgeFail, MsgCallBridgeFail)
	ErrCallHoldFail      = NewBizError(CodeCallHoldFail, MsgCallHoldFail)
	ErrCallTransferFail  = NewBizError(CodeCallTransferFail, MsgCallTransferFail)
	ErrCallOriginateFail = NewBizError(CodeCallOriginateFail, MsgCallOriginateFail)
	ErrCallRecordFail    = NewBizError(CodeCallRecordFail, MsgCallRecordFail)

	// ESL 相关
	ErrESLNotConnected = NewBizError(CodeESLNotConnected, MsgESLNotConnected)
	ErrESLConnectFail  = NewBizError(CodeESLConnectFail, MsgESLConnectFail)
	ErrESLAuthFail     = NewBizError(CodeESLAuthFail, MsgESLAuthFail)
	ErrESLCommandFail  = NewBizError(CodeESLCommandFail, MsgESLCommandFail)

	// NATS 相关
	ErrNATSNotConnected  = NewBizError(CodeNATSNotConnected, MsgNATSNotConnected)
	ErrNATSPublishFail   = NewBizError(CodeNATSPublishFail, MsgNATSPublishFail)
	ErrNATSSubscribeFail = NewBizError(CodeNATSSubscribeFail, MsgNATSSubscribeFail)

	// License 授权相关
	ErrLicenseMissing             = NewBizError(CodeLicenseMissing, MsgLicenseMissing)
	ErrLicenseInvalid             = NewBizError(CodeLicenseInvalid, MsgLicenseInvalid)
	ErrLicenseExpired             = NewBizError(CodeLicenseExpired, MsgLicenseExpired)
	ErrLicenseFingerprintMismatch = NewBizError(CodeLicenseFingerprintMismatch, MsgLicenseFingerprintMismatch)
	ErrLicenseLimitExceeded       = NewBizError(CodeLicenseLimitExceeded, MsgLicenseLimitExceeded)
	ErrLicenseModuleDisabled      = NewBizError(CodeLicenseModuleDisabled, MsgLicenseModuleDisabled)
	ErrLicenseClockRollback       = NewBizError(CodeLicenseClockRollback, MsgLicenseClockRollback)
)
