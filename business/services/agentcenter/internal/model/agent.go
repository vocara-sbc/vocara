package model

import (
	"time"
)

// Agent 座席信息（数据库表 tb_agent）
type Agent struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentNumber string `gorm:"column:agent_number;size:20;uniqueIndex:uk_tenant_agent_number,priority:2;not null" json:"agent_number"`
	AgentName   string    `gorm:"column:agent_name;size:128;not null" json:"agent_name"`
	Password    string    `gorm:"column:password;size:128;not null" json:"-"`
	TenantID    string    `gorm:"column:tenant_id;size:64;index;not null" json:"tenant_id"`
	DnNumber    string    `gorm:"column:dn_number;size:32" json:"dn_number"`
	Email       string    `gorm:"column:email;size:128" json:"email"`
	Phone       string    `gorm:"column:mobile;size:32" json:"phone"`
	Status      int       `gorm:"column:status;default:1" json:"status"` // 1启用 0禁用
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (Agent) TableName() string {
	return "tb_agent"
}

// AgentSession 座席会话（Redis HASH）
type AgentSession struct {
	AgentID        string   `json:"agent_id"`
	AgentName      string   `json:"agent_name"`
	TenantID       string   `json:"tenant_id"`
	Status         int      `json:"status"` // AgentStatus
	DnNumber       string   `json:"dn_number"`
	SkillGroups    []string `json:"skill_groups"`
	CurrentCallID  string   `json:"current_call_id"`
	LastReadyTime  int64    `json:"last_ready_time"` // 秒级时间戳
	LastActiveTime int64    `json:"last_active_time"`
	CheckinTime    int64    `json:"checkin_time"`
	IP             string   `json:"ip"`
	Extension      string   `json:"extension"`
}

// GetStatus 获取状态枚举
func (s *AgentSession) GetStatus() AgentStatus {
	return AgentStatus(s.Status)
}

// OnlineAgent 在线座席记录（tb_online_agent）
type OnlineAgent struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID       string    `gorm:"column:agent_id;size:64" json:"agent_id"`
	AgentName     string    `gorm:"column:agent_name;size:128" json:"agent_name"`
	TenantID      string    `gorm:"column:tenant_id;size:64;index" json:"tenant_id"`
	DnNumber      string    `gorm:"column:dn_number;size:32" json:"dn_number"`
	Status        int       `gorm:"column:status;default:0" json:"status"` // AgentStatus
	StatusName    string    `gorm:"-" json:"status_name"`                    // 无落库列
	IP            string    `gorm:"column:node_addr;size:64" json:"ip"`      // 映射到 node_addr
	CheckinTime   time.Time `gorm:"column:checkin_time" json:"checkin_time"`
	LastHeartbeat time.Time `gorm:"-" json:"last_heartbeat"`                 // 无落库列
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (OnlineAgent) TableName() string {
	return "tb_online_agent"
}

// AgentServiceRecord 座席服务记录（tb_agent_service_record）
type AgentServiceRecord struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordID      string    `gorm:"column:record_id;size:64;uniqueIndex;not null" json:"record_id"`
	AgentID       string    `gorm:"column:agent_id;size:64;index;not null" json:"agent_id"`
	TenantID      string    `gorm:"column:tenant_id;size:64;index;not null" json:"tenant_id"`
	CallID        string    `gorm:"column:call_id;size:64;index" json:"call_id"`
	ServiceType   int       `gorm:"column:service_type;default:1" json:"service_type"` // 1呼入 2外呼 3内部
	CheckinTime   time.Time  `gorm:"column:checkin_time" json:"checkin_time"`
	CheckoutTime  *time.Time `gorm:"column:checkout_time" json:"checkout_time"`
	TalkDuration  int       `gorm:"column:talk_duration;default:0" json:"talk_duration"` // 秒
	HoldDuration  int       `gorm:"column:hold_duration;default:0" json:"hold_duration"`
	ReadyDuration int       `gorm:"column:ready_duration;default:0" json:"ready_duration"`
	BusyDuration  int       `gorm:"column:busy_duration;default:0" json:"busy_duration"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名
func (AgentServiceRecord) TableName() string {
	return "tb_agent_service_record"
}

// ---------- DTO ----------

// LoginRequest 登录请求
type LoginRequest struct {
	AgentID  string `json:"agent_id"  binding:"required"`
	Password string `json:"password"  binding:"required"`
	TenantID string `json:"tenant_id" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	ExpiresIn int    `json:"expires_in"` // 秒
}

// CheckinRequest 签入请求
type CheckinRequest struct {
	DnNumber string   `json:"dn_number" binding:"required"`
	Skills   []string `json:"skills"`
	IP       string   `json:"ip"`
}

// CheckoutRequest 签出请求
type CheckoutRequest struct {
	Reason string `json:"reason"`
}

// StatusChangeRequest 状态变更请求
type StatusChangeRequest struct {
	Reason string `json:"reason"`
}

// AgentInfoResponse 座席信息响应
type AgentInfoResponse struct {
	AgentID       string   `json:"agent_id"`
	AgentName     string   `json:"agent_name"`
	TenantID      string   `json:"tenant_id"`
	Status        int      `json:"status"`
	StatusName    string   `json:"status_name"`
	DnNumber      string   `json:"dn_number"`
	SkillGroups   []string `json:"skill_groups"`
	CurrentCallID string   `json:"current_call_id"`
	LastReadyTime int64    `json:"last_ready_time"`
	CheckinTime   int64    `json:"checkin_time"`
}

// AgentStatusResponse 座席状态响应
type AgentStatusResponse struct {
	AgentID    string `json:"agent_id"`
	Status     int    `json:"status"`
	StatusName string `json:"status_name"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	AgentID string `json:"agent_id"`
	Status  int    `json:"status"`
}
