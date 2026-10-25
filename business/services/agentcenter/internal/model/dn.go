package model

import "time"

// DnStatus 分机状态
type DnStatus int

const (
	DnStatusOffline  DnStatus = 0 // 离线
	DnStatusIdle     DnStatus = 1 // 空闲
	DnStatusBusy     DnStatus = 2 // 忙碌（通话中）
	DnStatusRinging  DnStatus = 3 // 振铃
	DnStatusHold     DnStatus = 4 // 保持
	DnStatusDisabled DnStatus = 9 // 禁用
)

func (s DnStatus) String() string {
	switch s {
	case DnStatusOffline:
		return "OFFLINE"
	case DnStatusIdle:
		return "IDLE"
	case DnStatusBusy:
		return "BUSY"
	case DnStatusRinging:
		return "RINGING"
	case DnStatusHold:
		return "HOLD"
	case DnStatusDisabled:
		return "DISABLED"
	default:
		return "UNKNOWN"
	}
}

// Dn 分机信息（tb_dn）
type Dn struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DnID        string    `gorm:"-" json:"dn_id"` // 当前 tb_dn 表无 dn_id 列，忽略
	DnNumber    string    `gorm:"column:dn_number;size:32;index;not null" json:"dn_number"`
	TenantID    string    `gorm:"column:tenant_id;size:64;index" json:"tenant_id"`
	DisplayName string    `gorm:"column:dn_name;size:128" json:"display_name"`
	Password    string    `gorm:"column:password;size:128" json:"-"`
	Status      int       `gorm:"column:status;default:1" json:"status"` // 1启用 0禁用
	AgentID     string    `gorm:"column:agent_id;size:64;index" json:"agent_id"` // 绑定的座席
	RegisterIP  string    `gorm:"column:ip_address;size:64" json:"register_ip"`
	LastRegister *time.Time `gorm:"column:last_register_at" json:"last_register"`
	Description string    `gorm:"column:remark;size:255" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (Dn) TableName() string {
	return "tb_dn"
}

// ---------- DTO ----------

// DnInfoResponse 分机信息响应
type DnInfoResponse struct {
	DnID         string `json:"dn_id"`
	DnNumber     string `json:"dn_number"`
	TenantID     string `json:"tenant_id"`
	DisplayName  string `json:"display_name"`
	Status       int    `json:"status"`
	StatusName   string `json:"status_name"`
	AgentID      string `json:"agent_id"`
	RegisterIP   string `json:"register_ip"`
	LastRegister string `json:"last_register"`
	Online       bool   `json:"online"`
}

// DnCheckOnlineRequest 检测分机在线请求
type DnCheckOnlineRequest struct {
	DnNumbers []string `json:"dn_numbers" binding:"required"`
}

// DnCheckOnlineResponse 检测分机在线响应
type DnCheckOnlineResponse struct {
	Results map[string]bool `json:"results"` // dn_number -> online
}
