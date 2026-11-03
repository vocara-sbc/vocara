package model

import "time"

// DnStatus represents the extension status.
type DnStatus int

const (
	DnStatusOffline  DnStatus = 0 // offline
	DnStatusIdle     DnStatus = 1 // idle
	DnStatusBusy     DnStatus = 2 // busy (in a call)
	DnStatusRinging  DnStatus = 3 // ringing
	DnStatusHold     DnStatus = 4 // on hold
	DnStatusDisabled DnStatus = 9 // disabled
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

// Dn contains extension information (tb_dn).
type Dn struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DnID        string    `gorm:"-" json:"dn_id"` // ignored because tb_dn has no dn_id column
	DnNumber    string    `gorm:"column:dn_number;size:32;index;not null" json:"dn_number"`
	TenantID    string    `gorm:"column:tenant_id;size:64;index" json:"tenant_id"`
	DisplayName string    `gorm:"column:dn_name;size:128" json:"display_name"`
	Password    string    `gorm:"column:password;size:128" json:"-"`
	Status      int       `gorm:"column:status;default:1" json:"status"` // 1=enabled, 0=disabled
	AgentID     string    `gorm:"column:agent_id;size:64;index" json:"agent_id"` // bound agent
	RegisterIP  string    `gorm:"column:ip_address;size:64" json:"register_ip"`
	LastRegister *time.Time `gorm:"column:last_register_at" json:"last_register"`
	Description string    `gorm:"column:remark;size:255" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name.
func (Dn) TableName() string {
	return "tb_dn"
}

// ---------- DTO ----------

// DnInfoResponse contains extension information.
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

// DnCheckOnlineRequest contains extensions whose online status should be checked.
type DnCheckOnlineRequest struct {
	DnNumbers []string `json:"dn_numbers" binding:"required"`
}

// DnCheckOnlineResponse contains extension online-status results.
type DnCheckOnlineResponse struct {
	Results map[string]bool `json:"results"` // dn_number -> online
}
