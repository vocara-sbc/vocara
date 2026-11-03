package model

import "time"

// Customer is the customer master record. Customers are identified by phone number, unique within a tenant, so multiple incoming calls are aggregated under one customer.
// Database table: tb_customer
type Customer struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID    uint64    `gorm:"not null;default:0;uniqueIndex:uk_tenant_phone,priority:1;comment:租户ID" json:"tenant_id"`
	Phone       string    `gorm:"size:32;not null;uniqueIndex:uk_tenant_phone,priority:2;comment:客户手机号" json:"phone"`
	Name        string    `gorm:"size:128;comment:客户姓名" json:"name"`
	Data        string    `gorm:"type:mediumtext;comment:客户信息聚合 JSON" json:"data"`
	CallCount   int       `gorm:"not null;default:0;comment:累计来电/填报次数" json:"call_count"`
	FirstSeenAt time.Time `gorm:"comment:首次出现" json:"first_seen_at"`
	LastSeenAt  time.Time `gorm:"comment:最近出现" json:"last_seen_at"`
	CreatedAt   time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName specifies the customer master record table name.
func (Customer) TableName() string { return "tb_customer" }

// Business type constants for the biz_type field of templates and records.
const (
	BizTypeNone     = "none"
	BizTypeCustomer = "customer"
	BizTypeTicket   = "ticket"
)

// Ticket status constants.
const (
	TicketStatusOpen   = "open"
	TicketStatusClosed = "closed"
)
