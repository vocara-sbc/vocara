package rbac

import (
	"context"
	"strings"

	"vocara/pkg/errors"
	"vocara/pkg/middleware"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// RequireRole returns role validation middleware that supports role inheritance.
// Usage: r.GET("/api/agent/list", rbac.RequireRole(rbac.RoleAdmin), handler.AgentList)
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

// RequirePermission returns permission code validation middleware.
// Usage: r.GET("/api/agent/add", rbac.RequirePermission(rbac.PermActionAgentAdd), handler.AgentAdd)
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

// RequireAnyPermission requires at least one of the specified permissions.
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

// GetPermissionsFromCtx gets all permission codes for the current user from the request context.
// It supports direct use in handlers and returning permissions to the frontend.
func GetPermissionsFromCtx(ctx *app.RequestContext) []string {
	role := middleware.GetUserRole(ctx)
	if role == "" {
		return []string{}
	}
	return GetPermissions(role)
}

// IsAdminMenuPath reports whether a path is an admin page path for coarse-grained frontend routing.
// The path starts with /admin/ or belongs to an admin module.
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
