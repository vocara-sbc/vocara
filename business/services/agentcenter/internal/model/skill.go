package model

import "time"

// SkillGroup 技能组（tb_skill_group）
type SkillGroup struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID    uint64    `gorm:"column:tenant_id;index;not null" json:"tenant_id"`
	SkillCode   string    `gorm:"column:skill_code;size:50;uniqueIndex;not null" json:"skill_code"`
	SkillName   string    `gorm:"column:skill_name;size:50;not null" json:"skill_name"`
	SkillType   int       `gorm:"column:skill_type;default:1" json:"skill_type"` // 1-呼入 2-呼出 3-混合型
	Status      int       `gorm:"column:status;default:1" json:"status"`         // 1启用 0禁用
	Priority    int       `gorm:"column:priority;default:0" json:"priority"`
	AgentCount  int       `gorm:"column:agent_count;default:0" json:"agent_count"`
	Description string    `gorm:"column:description;size:255" json:"description"`
	Remark      string    `gorm:"column:remark;size:500" json:"remark"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (SkillGroup) TableName() string {
	return "tb_skill_group"
}

// AgentSkill 座席-技能组关系（tb_agent_skill，admin 与 agentcenter 共用）
type AgentSkill struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64    `gorm:"column:tenant_id;index;not null" json:"tenant_id"`
	SkillGroupID uint64    `gorm:"column:skill_group_id;index:idx_skill_agent,unique,priority:1;not null" json:"skill_group_id"`
	AgentID      uint64    `gorm:"column:agent_id;index:idx_skill_agent,unique,priority:2;not null" json:"agent_id"`
	AgentNumber  string    `gorm:"column:agent_number;size:20;index" json:"agent_number"`
	AgentName    string    `gorm:"column:agent_name;size:50" json:"agent_name"`
	SkillLevel   int       `gorm:"column:skill_level;default:5" json:"skill_level"` // 技能等级 1-10
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名
func (AgentSkill) TableName() string {
	return "tb_agent_skill"
}

// ---------- DTO ----------

// SkillInfoResponse 技能组信息响应
type SkillInfoResponse struct {
	ID          uint64 `json:"id"`
	SkillCode   string `json:"skill_code"`
	SkillName   string `json:"skill_name"`
	TenantID    uint64 `json:"tenant_id"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Status      int    `json:"status"`
	AgentCount  int    `json:"agent_count"`
	IdleCount   int    `json:"idle_count"`
}

// SkillAgentInfo 技能组下座席信息
type SkillAgentInfo struct {
	AgentNumber string `json:"agent_number"`
	AgentName   string `json:"agent_name"`
	Status      int    `json:"status"`
	StatusName  string `json:"status_name"`
	Level       int    `json:"level"`
}
