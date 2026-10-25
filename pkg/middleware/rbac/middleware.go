package rbac

import (
	"context"
	"strings"

	"vocara/pkg/errors"
	"vocara/pkg/middleware"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// RequireRole 角色校验中间件（含继承关系）
// 用法：r.GET("/api/agent/list", rbac.RequireRole(rbac.RoleAdmin), handler.AgentList)
func RequireRole(role string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		userRole := middleware.GetUserRole(ctx)
		if userRole == "" {
			response.Fail(ctx, errors.ErrUnauthorized)
			ctx.Abort()
			return
		}
		if !HasRole(userRole, role) {
			response.Fail(ctx, errors.ErrForbidden)
			ctx.Abort()
			return
		}
		ctx.Next(c)
	}
}

// RequirePermission 权限码校验中间件
// 用法：r.GET("/api/agent/add", rbac.RequirePermission(rbac.PermActionAgentAdd), handler.AgentAdd)
func RequirePermission(permission string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		userRole := middleware.GetUserRole(ctx)
		if userRole == "" {
			response.Fail(ctx, errors.ErrUnauthorized)
			ctx.Abort()
			return
		}
		if !HasPermission(userRole, permission) {
			response.Fail(ctx, errors.ErrForbidden)
			ctx.Abort()
			return
		}
		ctx.Next(c)
	}
}

// RequireAnyPermission 任意一个权限满足即可
func RequireAnyPermission(permissions ...string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		userRole := middleware.GetUserRole(ctx)
		if userRole == "" {
			response.Fail(ctx, errors.ErrUnauthorized)
			ctx.Abort()
			return
		}
		if !HasAnyPermission(userRole, permissions...) {
			response.Fail(ctx, errors.ErrForbidden)
			ctx.Abort()
			return
		}
		ctx.Next(c)
	}
}

// GetPermissionsFromCtx 从请求上下文中获取当前用户的所有权限码
// （方便在 handler 里直接用，也用于返回给前端）
func GetPermissionsFromCtx(ctx *app.RequestContext) []string {
	role := middleware.GetUserRole(ctx)
	if role == "" {
		return []string{}
	}
	return GetPermissions(role)
}

// IsAdminPath 判断是否为管理后台页面路径（粗粒度，前端路由用）
// 路径以 /admin/ 开头或属于管理端模块
func IsAdminMenuPath(path string) bool {
	adminPrefixes := []string{
		"/dashboard",
		"/agent",
		"/queue",
		"/admit",
		"/records",
		"/lines",
		"/stats",
		"/system",
	}
	for _, p := range adminPrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}
