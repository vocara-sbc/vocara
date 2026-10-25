// Package rbac 基于角色的权限控制（轻量实现，预留切换至 Casbin 等权限框架的扩展点）
//
// 设计原则：
// - 权限判断统一走 RequirePermission / RequireRole 中间件
// - 前端菜单用权限码控制，不用角色名硬编码
// - 以后换权限系统，只改中间件内部实现，业务代码不动
package rbac

// 角色定义
const (
	RoleAdmin      = "admin"      // 超级管理员
	RoleSupervisor = "supervisor" // 班长/主管
	RoleAgent      = "agent"      // 普通座席
)

// 权限码定义（menu: 菜单权限, action: 操作权限）
// 命名规范：模块:子模块:操作
const (
	// ========== 仪表盘 ==========
	PermMenuDashboard = "menu:dashboard:view"

	// ========== 座席管理 ==========
	PermMenuAgentList   = "menu:agent:view"     // 座席列表菜单
	PermActionAgentAdd  = "action:agent:add"    // 新增座席
	PermActionAgentEdit = "action:agent:edit"   // 编辑座席
	PermActionAgentDel  = "action:agent:delete" // 删除座席

	PermMenuAgentGroup = "menu:agent_group:view" // 座席组菜单
	PermMenuAgentSkill = "menu:agent_skill:view" // 技能组菜单
	PermMenuAgentDn    = "menu:agent_dn:view"    // 分机管理菜单

	// ========== 队列管理 ==========
	PermMenuQueueList    = "menu:queue:view"         // 队列配置菜单
	PermMenuQueueMonitor = "menu:queue_monitor:view" // 实时监控菜单

	// ========== 呼入接待 ==========
	PermMenuAdmit = "menu:admit:view"

	// ========== 通话记录 ==========
	PermMenuRecordInbound  = "menu:record:inbound"  // 呼入记录
	PermMenuRecordOutbound = "menu:record:outbound" // 呼出记录
	PermMenuRecordActive   = "menu:record:active"   // 实时通话

	// ========== 线路资源 ==========
	PermMenuLines = "menu:lines:view"

	// ========== 统计报表 ==========
	PermMenuStatsAgent = "menu:stats:agent" // 座席统计
	PermMenuStatsQueue = "menu:stats:queue" // 队列统计
	PermMenuStatsLine  = "menu:stats:line"  // 线路统计
	PermMenuStatsTrend = "menu:stats:trend" // 趋势分析

	// ========== 系统管理 ==========
	PermMenuSystemParam   = "menu:system:param"   // 参数配置
	PermMenuSystemLog     = "menu:system:log"     // 操作日志
	PermMenuSystemFs      = "menu:system:fs"      // FS 实例
	PermMenuSystemTenant  = "menu:system:tenant"  // 租户管理
	PermMenuSystemLicense = "menu:system:license" // 授权许可

	// ========== 座席工作台 ==========
	PermMenuWorkbench = "menu:workbench:view" // 座席工作台

	// ========== 工单与表单（规划占位） ==========
	PermMenuTicket = "menu:ticket:view" // 工单管理
	PermMenuForm   = "menu:form:view"   // 表单模板/表单管理
)

// rolePermissions 角色 -> 权限码 映射
// 注意：高权限角色自动包含低权限角色的菜单（通过 RoleHierarchy 实现）
var rolePermissions = map[string][]string{
	RoleAdmin: {
		// 仪表盘
		PermMenuDashboard,
		// 座席管理
		PermMenuAgentList,
		PermActionAgentAdd,
		PermActionAgentEdit,
		PermActionAgentDel,
		PermMenuAgentGroup,
		PermMenuAgentSkill,
		PermMenuAgentDn,
		// 队列管理
		PermMenuQueueList,
		PermMenuQueueMonitor,
		// 呼入接待
		PermMenuAdmit,
		// 通话记录
		PermMenuRecordInbound,
		PermMenuRecordOutbound,
		PermMenuRecordActive,
		// 线路资源
		PermMenuLines,
		// 统计报表
		PermMenuStatsAgent,
		PermMenuStatsQueue,
		PermMenuStatsLine,
		PermMenuStatsTrend,
		// 系统管理
		PermMenuSystemParam,
		PermMenuSystemLog,
		PermMenuSystemFs,
		PermMenuSystemTenant,
		PermMenuSystemLicense,
		// 工单与表单（规划占位）
		PermMenuTicket,
		PermMenuForm,
	},

	RoleSupervisor: {
		// 仪表盘
		PermMenuDashboard,
		// 座席管理（查看权限，没有删除）
		PermMenuAgentList,
		PermActionAgentEdit,
		PermMenuAgentGroup,
		PermMenuAgentSkill,
		// 队列管理
		PermMenuQueueList,
		PermMenuQueueMonitor,
		// 呼入接待
		PermMenuAdmit,
		// 通话记录
		PermMenuRecordInbound,
		PermMenuRecordOutbound,
		PermMenuRecordActive,
		// 统计报表
		PermMenuStatsAgent,
		PermMenuStatsQueue,
		PermMenuStatsTrend,
		// 系统日志
		PermMenuSystemLog,
	},

	RoleAgent: {
		// 座席菜单：工作台、通话记录、工单、表单（座席填报场景）
		PermMenuWorkbench,
		PermMenuRecordInbound,
		PermMenuRecordOutbound,
		PermMenuTicket,
		PermMenuForm,
	},
}

// roleHierarchy 角色继承关系：高级角色自动继承低级角色的所有权限
// admin > supervisor > agent
var roleHierarchy = map[string][]string{
	RoleAdmin:      {RoleSupervisor, RoleAgent},
	RoleSupervisor: {RoleAgent},
}

// GetPermissions 获取角色拥有的所有权限码（含继承）
func GetPermissions(role string) []string {
	perms := make(map[string]bool)

	// 收集当前角色和所有继承角色的权限
	collectPerms(role, perms)

	result := make([]string, 0, len(perms))
	for p := range perms {
		result = append(result, p)
	}
	return result
}

// HasPermission 判断角色是否拥有指定权限码
func HasPermission(role string, permission string) bool {
	perms := GetPermissions(role)
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission 判断角色是否拥有指定权限中的任意一个
func HasAnyPermission(role string, permissions ...string) bool {
	for _, p := range permissions {
		if HasPermission(role, p) {
			return true
		}
	}
	return false
}

// HasRole 判断角色是否匹配（含继承关系）
func HasRole(role string, targetRole string) bool {
	if role == targetRole {
		return true
	}
	// 递归检查继承关系
	if children, ok := roleHierarchy[role]; ok {
		for _, r := range children {
			if HasRole(r, targetRole) {
				return true
			}
		}
	}
	return false
}

// collectPerms 递归收集权限
func collectPerms(role string, perms map[string]bool) {
	if list, ok := rolePermissions[role]; ok {
		for _, p := range list {
			perms[p] = true
		}
	}
	if children, ok := roleHierarchy[role]; ok {
		for _, r := range children {
			collectPerms(r, perms)
		}
	}
}
