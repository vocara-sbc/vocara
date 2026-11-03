// Package rbac implements lightweight role-based access control with extension points for permission frameworks such as Casbin.
//
// Design principles:
// - Permission checks use the RequirePermission and RequireRole middleware uniformly
// - Frontend menus are controlled by permission codes rather than hard-coded role names
// - Permission system implementations can change within the middleware without affecting business code
package rbac

// Role definitions
const (
	RoleAdmin      = "admin"      // Super administrator
	RoleSupervisor = "supervisor" // Supervisor
	RoleAgent      = "agent"      // Regular agent
)

// Permission code definitions (menu: menu permission, action: operation permission)
// Naming convention: module:submodule:operation
const (
	// ========== Dashboard ==========
	PermMenuDashboard = "menu:dashboard:view"

	// ========== Agent management ==========
	PermMenuAgentList   = "menu:agent:view"     // Agent list menu
	PermActionAgentAdd  = "action:agent:add"    // Add agent
	PermActionAgentEdit = "action:agent:edit"   // Edit agent
	PermActionAgentDel  = "action:agent:delete" // Delete agent

	PermMenuAgentGroup = "menu:agent_group:view" // Agent group menu
	PermMenuAgentSkill = "menu:agent_skill:view" // Skill group menu
	PermMenuAgentDn    = "menu:agent_dn:view"    // Extension management menu

	// ========== Queue management ==========
	PermMenuQueueList    = "menu:queue:view"         // Queue configuration menu
	PermMenuQueueMonitor = "menu:queue_monitor:view" // Real-time monitoring menu

	// ========== Inbound admission ==========
	PermMenuAdmit = "menu:admit:view"

	// ========== Call records ==========
	PermMenuRecordInbound  = "menu:record:inbound"  // Inbound records
	PermMenuRecordOutbound = "menu:record:outbound" // Outbound records
	PermMenuRecordActive   = "menu:record:active"   // Real-time calls

	// ========== Line resources ==========
	PermMenuLines = "menu:lines:view"

	// ========== Statistical reports ==========
	PermMenuStatsAgent = "menu:stats:agent" // Agent statistics
	PermMenuStatsQueue = "menu:stats:queue" // Queue statistics
	PermMenuStatsLine  = "menu:stats:line"  // Line statistics
	PermMenuStatsTrend = "menu:stats:trend" // Trend analysis

	// ========== System management ==========
	PermMenuSystemParam   = "menu:system:param"   // Parameter configuration
	PermMenuSystemLog     = "menu:system:log"     // Operation log
	PermMenuSystemFs      = "menu:system:fs"      // FS instance
	PermMenuSystemTenant  = "menu:system:tenant"  // Tenant management
	PermMenuSystemLicense = "menu:system:license" // License authorization

	// ========== Agent workbench ==========
	PermMenuWorkbench = "menu:workbench:view" // Agent workbench

	// ========== Tickets and forms (reserved) ==========
	PermMenuTicket = "menu:ticket:view" // Ticket management
	PermMenuForm   = "menu:form:view"   // Form templates and form management
)

// rolePermissions maps roles to permission codes.
// Higher-privilege roles automatically include lower-privilege roles' menus through roleHierarchy.
var rolePermissions = map[string][]string{
	RoleAdmin: {
		// Dashboard
		PermMenuDashboard,
		// Agent management
		PermMenuAgentList,
		PermActionAgentAdd,
		PermActionAgentEdit,
		PermActionAgentDel,
		PermMenuAgentGroup,
		PermMenuAgentSkill,
		PermMenuAgentDn,
		// Queue management
		PermMenuQueueList,
		PermMenuQueueMonitor,
		// Inbound admission
		PermMenuAdmit,
		// Call records
		PermMenuRecordInbound,
		PermMenuRecordOutbound,
		PermMenuRecordActive,
		// Line resources
		PermMenuLines,
		// Statistical reports
		PermMenuStatsAgent,
		PermMenuStatsQueue,
		PermMenuStatsLine,
		PermMenuStatsTrend,
		// System management
		PermMenuSystemParam,
		PermMenuSystemLog,
		PermMenuSystemFs,
		PermMenuSystemTenant,
		PermMenuSystemLicense,
		// Tickets and forms (reserved)
		PermMenuTicket,
		PermMenuForm,
	},

	RoleSupervisor: {
		// Dashboard
		PermMenuDashboard,
		// Agent management with view and edit permissions but no deletion
		PermMenuAgentList,
		PermActionAgentEdit,
		PermMenuAgentGroup,
		PermMenuAgentSkill,
		// Queue management
		PermMenuQueueList,
		PermMenuQueueMonitor,
		// Inbound admission
		PermMenuAdmit,
		// Call records
		PermMenuRecordInbound,
		PermMenuRecordOutbound,
		PermMenuRecordActive,
		// Statistical reports
		PermMenuStatsAgent,
		PermMenuStatsQueue,
		PermMenuStatsTrend,
		// System logs
		PermMenuSystemLog,
	},

	RoleAgent: {
		// Agent menus: workbench, call records, tickets, and forms for agent submissions
		PermMenuWorkbench,
		PermMenuRecordInbound,
		PermMenuRecordOutbound,
		PermMenuTicket,
		PermMenuForm,
	},
}

// roleHierarchy defines role inheritance: higher roles automatically inherit all permissions of lower roles.
// admin > supervisor > agent
var roleHierarchy = map[string][]string{
	RoleAdmin:      {RoleSupervisor, RoleAgent},
	RoleSupervisor: {RoleAgent},
}

// GetPermissions returns all permission codes for a role, including inherited permissions.
func GetPermissions(role string) []string {
	perms := make(map[string]bool)

	// Collect permissions for the current role and all inherited roles
	collectPerms(role, perms)

	result := make([]string, 0, len(perms))
	for p := range perms {
		result = append(result, p)
	}
	return result
}

// HasPermission reports whether a role has the specified permission code.
func HasPermission(role string, permission string) bool {
	perms := GetPermissions(role)
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission reports whether a role has any of the specified permissions.
func HasAnyPermission(role string, permissions ...string) bool {
	for _, p := range permissions {
		if HasPermission(role, p) {
			return true
		}
	}
	return false
}

// HasRole reports whether a role matches, including inheritance.
func HasRole(role string, targetRole string) bool {
	if role == targetRole {
		return true
	}
	// Recursively check inheritance
	if children, ok := roleHierarchy[role]; ok {
		for _, r := range children {
			if HasRole(r, targetRole) {
				return true
			}
		}
	}
	return false
}

// collectPerms recursively collects permissions.
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
