// Package model 定义管理后台的数据模型
// 包含所有数据库表对应的 GORM 模型结构体
package model

import (
	"time"
)

// BaseModel 基础模型，包含通用字段
type BaseModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID  uint64    `gorm:"index;not null;default:0;comment:租户ID" json:"tenant_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// Tenant 企业/租户模型
// 对应表: tb_tenant
type Tenant struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;comment:租户ID" json:"id"`
	TenantName  string    `gorm:"size:100;not null;comment:租户名称" json:"tenant_name"`
	TenantCode  string    `gorm:"size:50;uniqueIndex;not null;comment:租户编码" json:"tenant_code"`
	ContactName string    `gorm:"size:50;comment:联系人" json:"contact_name"`
	ContactPhone string   `gorm:"size:20;comment:联系电话" json:"contact_phone"`
	ContactEmail string   `gorm:"size:100;comment:联系邮箱" json:"contact_email"`
	Address     string    `gorm:"size:255;comment:地址" json:"address"`
	Status      int       `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	ExpireTime  time.Time `gorm:"comment:过期时间" json:"expire_time"`
	MaxAgents   int       `gorm:"not null;default:0;comment:最大座席数" json:"max_agents"`
	MaxLines    int       `gorm:"not null;default:0;comment:最大线路数" json:"max_lines"`
	Remark      string    `gorm:"size:500;comment:备注" json:"remark"`
	CreatedAt   time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "tb_tenant"
}

// Agent 座席模型
// 对应表: tb_agent
type Agent struct {
	BaseModel
	AgentNumber string `gorm:"column:agent_number;size:20;not null;comment:座席工号" json:"agent_number"`
	AgentName   string `gorm:"size:50;not null;comment:座席姓名" json:"agent_name"`
	Password    string `gorm:"size:100;not null;comment:密码(加密)" json:"-"`
	Mobile      string `gorm:"size:20;comment:手机号" json:"mobile"`
	Email       string `gorm:"size:100;comment:邮箱" json:"email"`
	Avatar      string `gorm:"size:255;comment:头像URL" json:"avatar"`
	Gender      int    `gorm:"default:0;comment:性别: 0-未知 1-男 2-女" json:"gender"`
	DN          string `gorm:"column:dn_number;size:20;comment:绑定分机号" json:"dn"`
	Role        string `gorm:"size:20;default:agent;comment:角色: admin-管理员 supervisor-班长 agent-座席" json:"role"`
	Status      int    `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	WorkStatus  int    `gorm:"not null;default:0;comment:工作状态: 0-离线 1-空闲 2-忙碌 3-通话中 4-休息 5-会后处理" json:"work_status"`
	OrgID       uint64 `gorm:"index;comment:所属组织机构ID" json:"org_id"`
	SkillLevels string `gorm:"type:text;comment:技能等级(JSON)" json:"skill_levels"`
	// 指针类型：座席未登录时为 NULL，而非零值时间（避免 0000-00-00 触发 MySQL 1292）
	LastLoginAt *time.Time `gorm:"comment:最后登录时间" json:"last_login_at"`
	LastLoginIP string    `gorm:"size:50;comment:最后登录IP" json:"last_login_ip"`
	Remark      string    `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定表名
func (Agent) TableName() string {
	return "tb_agent"
}

// SkillGroup 技能组模型
// 对应表: tb_skill_group
type SkillGroup struct {
	BaseModel
	SkillName  string `gorm:"size:50;not null;comment:技能组名称" json:"skill_name"`
	SkillCode  string `gorm:"size:50;uniqueIndex:idx_tenant_skill;not null;comment:技能组编码" json:"skill_code"`
	SkillType  int    `gorm:"default:1;comment:技能类型: 1-呼入 2-呼出 3-混合型" json:"skill_type"`
	Status     int    `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	Priority   int    `gorm:"default:0;comment:优先级" json:"priority"`
	AgentCount int    `gorm:"default:0;comment:座席数量" json:"agent_count"`
	Remark     string `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定表名
func (SkillGroup) TableName() string {
	return "tb_skill_group"
}

// AgentSkill 座席-技能组关系模型
// 对应表: tb_agent_skill
type AgentSkill struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID   uint64    `gorm:"index;not null;comment:租户ID" json:"tenant_id"`
	SkillGroupID uint64 `gorm:"column:skill_group_id;index:idx_skill_agent;not null;comment:技能组ID" json:"skill_group_id"`
	AgentID      uint64 `gorm:"index:idx_skill_agent;not null;comment:座席ID" json:"agent_id"`
	AgentNumber  string `gorm:"size:20;comment:座席工号" json:"agent_number"`
	AgentName  string    `gorm:"size:50;comment:座席姓名" json:"agent_name"`
	SkillLevel int       `gorm:"default:5;comment:技能等级 1-10" json:"skill_level"`
	CreatedAt  time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// TableName 指定表名
func (AgentSkill) TableName() string {
	return "tb_agent_skill"
}

// DN 分机模型
// 对应表: tb_dn
type DN struct {
	BaseModel
	DNNum       string `gorm:"column:dn_number;size:20;not null;comment:分机号码" json:"dn_num"`
	DNName      string `gorm:"size:50;comment:分机名称" json:"dn_name"`
	Password    string `gorm:"size:100;comment:注册密码" json:"-"`
	DNType      int    `gorm:"default:1;comment:分机类型: 1-座席分机 2-通用分机" json:"dn_type"`
	AgentID     uint64 `gorm:"index;comment:绑定座席ID" json:"agent_id"`
	AgentNumber string `gorm:"size:20;comment:绑定座席工号" json:"agent_number"`
	Status      int    `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	RegisterStatus int `gorm:"default:0;comment:注册状态: 0-未注册 1-已注册" json:"register_status"`
	RegisterIP   string `gorm:"column:ip_address;size:50;comment:注册IP" json:"register_ip"`
	RegisterUA   string `gorm:"column:user_agent;size:100;comment:注册UA" json:"register_ua"`
	// 指针类型：未注册时分机为 NULL，而非零值时间；否则零值会被写成 0000-00-00 触发 MySQL 1292
	LastRegisterAt *time.Time `gorm:"comment:最后注册时间" json:"last_register_at"`
	FSInstanceID uint64 `gorm:"comment:所属FS实例ID" json:"fs_instance_id"`
	Remark      string `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定表名
func (DN) TableName() string {
	return "tb_dn"
}

// Queue 队列模型
// 对应表: tb_queue
type Queue struct {
	BaseModel
	QueueName    string  `gorm:"size:50;not null;comment:队列名称" json:"queue_name"`
	QueueCode    string  `gorm:"size:50;uniqueIndex:idx_tenant_queue;not null;comment:队列编码" json:"queue_code"`
	QueueType    int     `gorm:"default:1;comment:队列类型: 1-呼入 2-呼出" json:"queue_type"`
	Strategy     string  `gorm:"size:30;default:round_robin;comment:分配策略: round_robin-轮询 longest_idle-最长空闲 skill_level-技能等级" json:"strategy"`
	MaxWaitTime  int     `gorm:"default:60;comment:最大等待时间(秒)" json:"max_wait_time"`
	MaxWaitNum   int     `gorm:"default:0;comment:最大等待人数 0-不限制" json:"max_wait_num"`
	OverflowType int     `gorm:"default:0;comment:溢出处理: 0-挂断 1-转队列 2-转IVR 3-转座席" json:"overflow_type"`
	OverflowValue string `gorm:"size:100;comment:溢出值" json:"overflow_value"`
	NoAgentType  int     `gorm:"default:0;comment:无座席处理: 0-等待 1-溢出 2-挂断" json:"no_agent_type"`
	WrapUpTime   int     `gorm:"default:0;comment:会后处理时间(秒)" json:"wrap_up_time"`
	SkillIDs     string  `gorm:"type:text;comment:关联技能组ID列表(JSON)" json:"skill_ids"`
	Status       int     `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	WaitNum      int     `gorm:"default:0;comment:当前等待数" json:"wait_num"`
	AgentReadyNum int    `gorm:"default:0;comment:就绪座席数" json:"agent_ready_num"`
	AgentBusyNum int     `gorm:"default:0;comment:忙碌座席数" json:"agent_busy_num"`
	SoundFile    string  `gorm:"size:255;comment:等待音文件" json:"sound_file"`
	Remark       string  `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定表名
func (Queue) TableName() string {
	return "tb_queue"
}

// InboundRoute 呼入路由模型
// 对应表: tb_inbound_route
type InboundRoute struct {
	BaseModel
	RouteName      string `gorm:"size:50;not null;comment:路由名称" json:"route_name"`
	RouteCode      string `gorm:"size:50;uniqueIndex:uk_route_code;not null;comment:路由编码" json:"route_code"`
	DID            string `gorm:"column:did;size:30;index;not null;comment:呼入接入号(DID)" json:"did"`
	WelcomeSound   string `gorm:"size:255;comment:欢迎音文件" json:"welcome_sound"`
	WaitSound      string `gorm:"size:255;comment:等待音文件" json:"wait_sound"`
	QueueID        uint64 `gorm:"index;comment:目标队列ID" json:"queue_id"`
	QueueName      string `gorm:"size:50;comment:目标队列名称" json:"queue_name"`
	SkillGroupID   uint64 `gorm:"index;comment:目标技能组ID" json:"skill_group_id"`
	WorkTimeType   int    `gorm:"default:0;comment:工作时间类型: 0-全天候 1-自定义" json:"work_time_type"`
	WorkTimeConfig string `gorm:"type:text;comment:工作时间配置(JSON)" json:"work_time_config"`
	HolidayIDs     string `gorm:"type:text;comment:节假日ID列表(JSON)" json:"holiday_ids"`
	OffWorkType    int    `gorm:"default:0;comment:非工作时间处理: 0-挂断 1-留言 2-转座席 3-转队列" json:"off_work_type"`
	OffWorkValue   string `gorm:"size:255;comment:非工作时间处理值" json:"off_work_value"`
	OffWorkSound   string `gorm:"size:255;comment:非工作时间提示音" json:"off_work_sound"`
	Status         int    `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	Sort           int    `gorm:"default:0;comment:排序" json:"sort"`
	Remark         string `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定表名
func (InboundRoute) TableName() string {
	return "tb_inbound_route"
}

// CallRecord 通话记录模型
// 对应表: tb_call_record
type CallRecord struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID       uint64    `gorm:"index;not null;comment:租户ID" json:"tenant_id"`
	CallID         string    `gorm:"size:64;uniqueIndex:uk_call_id,priority:1;not null;comment:通话唯一ID" json:"call_id"`
	CallType       int       `gorm:"index;not null;comment:通话类型: 1-呼入 2-呼出 3-内部呼叫" json:"call_type"`
	CallerNumber   string    `gorm:"column:caller;size:30;index;comment:主叫号码" json:"caller_number"`
	CalledNumber   string    `gorm:"column:called;size:30;index;comment:被叫号码" json:"called_number"`
	DID            string    `gorm:"column:did;size:30;index;comment:呼入接入号(DID)" json:"did"`
	AgentID        uint64    `gorm:"index;comment:座席ID" json:"agent_id"`
	AgentNumber    string    `gorm:"size:20;comment:座席工号" json:"agent_number"`
	AgentName      string    `gorm:"size:50;comment:座席姓名" json:"agent_name"`
	DNNumber       string    `gorm:"column:dn_number;size:20;comment:分机号" json:"dn_number"`
	QueueID        uint64    `gorm:"index;comment:队列ID" json:"queue_id"`
	QueueName      string    `gorm:"size:50;comment:队列名称" json:"queue_name"`
	SkillGroupID   uint64    `gorm:"column:skill_group_id;comment:技能组ID" json:"skill_group_id"`
	SkillName      string    `gorm:"size:50;comment:技能组名称" json:"skill_name"`
	// StartTime 同时属于复合主键与 uk_call_id 唯一索引：
	// tb_call_record 按 YEAR(start_time) 分区，分区表的所有唯一索引必须包含分区列
	StartTime      time.Time `gorm:"primaryKey;uniqueIndex:uk_call_id,priority:2;index:idx_start_time;comment:开始时间" json:"start_time"`
	AnswerTime     time.Time `gorm:"comment:接听时间" json:"answer_time"`
	EndTime        time.Time `gorm:"index;comment:结束时间" json:"end_time"`
	TotalDuration  int       `gorm:"default:0;comment:总时长(秒)" json:"total_duration"`
	TalkDuration   int       `gorm:"default:0;comment:通话时长(秒)" json:"talk_duration"`
	WaitDuration   int       `gorm:"default:0;comment:等待时长(秒)" json:"wait_duration"`
	WrapUpDuration int       `gorm:"default:0;comment:会后处理时长(秒)" json:"wrap_up_duration"`
	CallStatus     int       `gorm:"default:0;comment:通话状态: 0-未接 1-已接 2-已挂断 3-转接" json:"call_status"`
	AnswerStatus   int       `gorm:"default:0;comment:接听状态: 0-未接 1-已接" json:"answer_status"`
	HangupCause    string    `gorm:"size:100;comment:挂断原因" json:"hangup_cause"`
	HangupSide     int       `gorm:"column:hangup_direction;default:0;comment:挂断方: 0-未知 1-主叫 2-被叫 3-系统" json:"hangup_side"`
	RecordFile     string    `gorm:"size:500;comment:录音文件路径" json:"record_file"`
	RecordURL      string    `gorm:"size:500;comment:录音文件URL" json:"record_url"`
	RecordDuration int       `gorm:"default:0;comment:录音时长(秒)" json:"record_duration"`
	TransferCount  int       `gorm:"default:0;comment:转接次数" json:"transfer_count"`
	HoldCount      int       `gorm:"default:0;comment:保持次数" json:"hold_count"`
	HoldDuration   int       `gorm:"default:0;comment:保持时长(秒)" json:"hold_duration"`
	Satisfaction   int       `gorm:"default:0;comment:满意度评分 0-5" json:"satisfaction"`
	Remark         string    `gorm:"size:500;comment:备注" json:"remark"`
	CreatedAt      time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (CallRecord) TableName() string {
	return "tb_call_record"
}

// Line 线路模型
// 对应表: tb_line
type Line struct {
	BaseModel
	LineName       string `gorm:"size:50;not null;comment:线路名称" json:"line_name"`
	LineCode       string `gorm:"size:50;uniqueIndex:uk_line_code;not null;comment:线路编码" json:"line_code"`
	LineType       int    `gorm:"default:1;comment:线路类型: 1-SIP中继 2-PRI中继 3-模拟线" json:"line_type"`
	Provider       string `gorm:"size:50;comment:运营商" json:"provider"`
	Gateway        string `gorm:"column:gateway_ip;size:100;comment:网关地址" json:"gateway"`
	Port           int    `gorm:"column:gateway_port;default:5060;comment:网关端口" json:"port"`
	Username       string `gorm:"size:50;comment:用户名" json:"username"`
	Password       string `gorm:"size:100;comment:密码" json:"-"`
	CallerID       string `gorm:"size:30;comment:主叫号码" json:"caller_id"`
	MaxChannels    int    `gorm:"default:30;comment:最大并发数" json:"max_channels"`
	UsedChannels   int    `gorm:"default:0;comment:已用并发数" json:"used_channels"`
	Status         int    `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	RegisterStatus int    `gorm:"default:0;comment:注册状态: 0-未注册 1-已注册" json:"register_status"`
	SkillIDs       string `gorm:"type:text;comment:关联技能组ID列表(JSON)" json:"skill_ids"`
	FSInstanceID   uint64 `gorm:"comment:所属FS实例ID" json:"fs_instance_id"`
	LastRegisterAt *time.Time `gorm:"comment:最后注册时间" json:"last_register_at"`
	Remark         string `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定表名
func (Line) TableName() string {
	return "tb_line"
}

// FSInstance FS 实例模型
// 对应表: tb_fs_instance
type FSInstance struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	InstanceName   string    `gorm:"-" json:"instance_name"`
	InstanceCode   string    `gorm:"column:instance_code;size:50;uniqueIndex:uk_instance_code;not null;comment:实例编码" json:"instance_code"`
	Host           string    `gorm:"column:ip_address;size:100;not null;comment:主机地址" json:"host"`
	ESLPort        int       `gorm:"default:8021;comment:ESL端口" json:"esl_port"`
	ESLPassword    string    `gorm:"size:50;comment:ESL密码" json:"-"`
	SIPPort        int       `gorm:"column:sip_port;default:5060;comment:SIP端口" json:"sip_port"`
	Status         int       `gorm:"default:0;comment:状态: 0-离线 1-在线" json:"status"`
	MaxSessions    int       `gorm:"default:1000;comment:最大会话数" json:"max_sessions"`
	CurrentSession int       `gorm:"default:0;comment:当前会话数" json:"current_session"`
	CPUUsage       float64   `gorm:"default:0;comment:CPU使用率%" json:"cpu_usage"`
	MemUsage       float64   `gorm:"default:0;comment:内存使用率%" json:"mem_usage"`
	LastHeartbeat  *time.Time `gorm:"comment:最后心跳时间" json:"last_heartbeat"`
	Version        string    `gorm:"size:50;comment:版本号" json:"version"`
	Remark         string    `gorm:"size:500;comment:备注" json:"remark"`
	CreatedAt      time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (FSInstance) TableName() string {
	return "tb_fs_instance"
}

// HolidaySchedule 节假日模型
// 对应表: tb_holiday_schedule
type HolidaySchedule struct {
	BaseModel
	HolidayName  string    `gorm:"size:50;not null;comment:节假日名称" json:"holiday_name"`
	HolidayType  int       `gorm:"default:1;comment:类型: 1-法定假日 2-自定义" json:"holiday_type"`
	StartDate    time.Time `gorm:"comment:开始日期" json:"start_date"`
	EndDate      time.Time `gorm:"comment:结束日期" json:"end_date"`
	TimeConfig   string    `gorm:"type:text;comment:时间配置(JSON)" json:"time_config"`
	Status       int       `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	Remark       string    `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定表名
func (HolidaySchedule) TableName() string {
	return "tb_holiday_schedule"
}

// SystemConfig 参数配置模型
// 对应表: tb_param_config
type SystemConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID    uint64    `gorm:"index;default:0;comment:租户ID 0表示全局" json:"tenant_id"`
	ConfigKey   string    `gorm:"size:100;uniqueIndex:idx_tenant_key;not null;comment:配置键" json:"config_key"`
	ConfigValue string    `gorm:"type:text;comment:配置值" json:"config_value"`
	ConfigType  string    `gorm:"size:20;default:string;comment:配置类型: string/int/bool/json" json:"config_type"`
	ConfigDesc  string    `gorm:"size:255;comment:配置描述" json:"config_desc"`
	IsSystem    int       `gorm:"default:0;comment:是否系统配置: 0-否 1-是" json:"is_system"`
	CreatedAt   time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "tb_param_config"
}

// OperationLog 操作日志模型
// 对应表: tb_operation_log
type OperationLog struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID      uint64    `gorm:"index;comment:租户ID" json:"tenant_id"`
	OperationType string    `gorm:"size:20;comment:操作类型: login/logout/add/update/delete/query" json:"operation_type"`
	Module        string    `gorm:"size:50;index;comment:操作模块" json:"module"`
	Operation     string    `gorm:"size:200;comment:操作内容" json:"operation"`
	Method        string    `gorm:"size:10;comment:请求方法" json:"method"`
	URL           string    `gorm:"size:255;comment:请求URL" json:"url"`
	RequestData   string    `gorm:"type:text;comment:请求参数" json:"request_data"`
	ResponseData  string    `gorm:"type:text;comment:响应数据" json:"response_data"`
	UserID        uint64    `gorm:"index;comment:操作用户ID" json:"user_id"`
	Username      string    `gorm:"size:50;comment:操作用户名" json:"username"`
	IP            string    `gorm:"size:50;comment:操作IP" json:"ip"`
	UserAgent     string    `gorm:"size:255;comment:用户代理" json:"user_agent"`
	Status        int       `gorm:"default:1;comment:状态: 1-成功 0-失败" json:"status"`
	ErrorMsg      string    `gorm:"size:500;comment:错误信息" json:"error_msg"`
	CostTime      int64     `gorm:"comment:耗时(毫秒)" json:"cost_time"`
	TraceID       string    `gorm:"size:64;index;comment:追踪ID" json:"trace_id"`
	CreatedAt     time.Time `gorm:"index;comment:创建时间" json:"created_at"`
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "tb_operation_log"
}
