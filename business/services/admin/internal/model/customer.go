package model

import "time"

// Customer 客户主档：按手机号（租户内唯一）识别，多次来电聚合为同一客户
// 对应表: tb_customer
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

// TableName 指定客户主档表名
func (Customer) TableName() string { return "tb_customer" }

// 业务类型常量：模板/记录的 biz_type
const (
	BizTypeNone     = "none"
	BizTypeCustomer = "customer"
	BizTypeTicket   = "ticket"
)

// 工单状态常量
const (
	TicketStatusOpen   = "open"
	TicketStatusClosed = "closed"
)
