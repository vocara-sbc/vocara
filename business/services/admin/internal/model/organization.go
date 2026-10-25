package model

// Organization 组织机构（部门）。
// 一棵多级组织树：ParentID/Level/Path 描述层级，LeaderID 为部门负责人。
// 座席行政上归属唯一一个部门（Agent.OrgID）；技能组（SkillGroup）是话务路由概念，与此独立。
// 对应表: tb_organization
type Organization struct {
	BaseModel
	OrgName    string `gorm:"size:100;not null;comment:机构/部门名称" json:"org_name"`
	OrgCode    string `gorm:"size:50;uniqueIndex:idx_tenant_org;comment:机构编码" json:"org_code"`
	ParentID   uint64 `gorm:"index;not null;default:0;comment:父机构ID（0为根）" json:"parent_id"`
	Level      int    `gorm:"not null;default:1;comment:层级（根=1）" json:"level"`
	Path       string `gorm:"size:1000;comment:祖先ID路径，逗号分隔" json:"path"`
	LeaderID   uint64 `gorm:"comment:负责人ID" json:"leader_id"`
	LeaderName string `gorm:"size:50;comment:负责人姓名" json:"leader_name"`
	Sort       int    `gorm:"not null;default:0;comment:同级排序" json:"sort"`
	Status     int    `gorm:"not null;default:1;comment:状态: 1-启用 0-禁用" json:"status"`
	Remark     string `gorm:"size:500;comment:备注" json:"remark"`
}

// TableName 指定组织机构表名
func (Organization) TableName() string {
	return "tb_organization"
}
