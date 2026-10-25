package model

import "time"

// FormTemplate 表单模板主表：身份与可变元数据；具体 schema 按版本存于版本表
// 复合唯一索引 uk_tenant_template(tenant_id, template_key)，与建表基线一致
// 对应表: tb_form_template
type FormTemplate struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID         uint64    `gorm:"not null;default:0;uniqueIndex:uk_tenant_template,priority:1;comment:租户ID" json:"tenant_id"`
	TemplateKey      string    `gorm:"size:64;not null;uniqueIndex:uk_tenant_template,priority:2;comment:模板标识" json:"template_key"`
	TemplateName     string    `gorm:"size:128;not null;comment:模板名称" json:"template_name"`
	Category         string    `gorm:"size:32;not null;default:general;comment:分类编码" json:"category"`
	BizType          string    `gorm:"size:20;not null;default:none;comment:业务类型: none/customer/ticket" json:"biz_type"`
	CurrentVersionID uint64    `gorm:"not null;default:0;comment:当前版本ID" json:"current_version_id"`
	Status           int       `gorm:"not null;default:1;comment:状态: 1-启用 0-停用" json:"status"`
	Remark           string    `gorm:"size:500;comment:备注" json:"remark"`
	CreatedAt        time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt        time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定模板主表名
func (FormTemplate) TableName() string { return "tb_form_template" }

// FormTemplateVersion 表单模板版本：schema 不可变，每次编辑保存新增一行
// 复合唯一索引 uk_template_version(template_id, version)
// 对应表: tb_form_template_version
type FormTemplateVersion struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID   uint64    `gorm:"not null;index:idx_tenant;comment:租户ID" json:"tenant_id"`
	TemplateID uint64    `gorm:"not null;uniqueIndex:uk_template_version,priority:1;comment:模板ID" json:"template_id"`
	Version    int       `gorm:"not null;uniqueIndex:uk_template_version,priority:2;comment:版本号" json:"version"`
	Schema     string    `gorm:"type:mediumtext;not null;comment:该版本 schema JSON" json:"schema"`
	CreatedAt  time.Time `gorm:"comment:发布时间" json:"created_at"`
}

// TableName 指定版本表名
func (FormTemplateVersion) TableName() string { return "tb_form_template_version" }

// FormRecord 表单实例记录
// 索引与建表基线一致：uk_tenant_record_key(tenant_id,template_key,record_key)、
// idx_tenant_template(tenant_id,template_id)
// 对应表: tb_form_record
type FormRecord struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID          uint64    `gorm:"not null;default:0;uniqueIndex:uk_tenant_record_key,priority:1;index:idx_tenant_template,priority:1;comment:租户ID" json:"tenant_id"`
	TemplateID        uint64    `gorm:"not null;comment:模板ID" json:"template_id"`
	TemplateVersionID uint64    `gorm:"not null;default:0;comment:绑定的模板版本ID" json:"template_version_id"`
	TemplateKey       string    `gorm:"size:64;not null;uniqueIndex:uk_tenant_record_key,priority:2;index:idx_tenant_template,priority:2;comment:模板标识" json:"template_key"`
	BizType           string    `gorm:"size:20;not null;default:none;index;comment:业务类型（创建时从模板复制）" json:"biz_type"`
	BizStatus         string    `gorm:"size:20;not null;default:'';comment:工单状态 open/closed" json:"biz_status"`
	CustomerID        uint64    `gorm:"index;comment:关联客户主档" json:"customer_id"`
	CallID            string    `gorm:"size:64;index;comment:关联通话ID" json:"call_id"`
	Direction         string    `gorm:"size:20;comment:呼叫方向 inbound/outbound" json:"direction"`
	AgentID           string    `gorm:"size:64;comment:座席ID" json:"agent_id"`
	AgentName         string    `gorm:"size:128;comment:座席名称" json:"agent_name"`
	RecordKey         *string   `gorm:"size:128;uniqueIndex:uk_tenant_record_key,priority:3;comment:业务主键" json:"record_key"`
	Data              string    `gorm:"type:mediumtext;not null;comment:表单数据 JSON" json:"data"`
	FieldSnapshot     string    `gorm:"type:mediumtext;comment:拍平字段快照" json:"field_snapshot"`
	CreatedAt         time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt         time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定实例表名
func (FormRecord) TableName() string { return "tb_form_record" }
