package handler

import (
	"context"
	"strings"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// AgentAuth provides JWT authentication middleware for agents.
func AgentAuth(agentSvc *service.AgentService) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		authHeader := string(ctx.GetHeader("Authorization"))
		if authHeader == "" {
			// Also accept the token as a query parameter.
			authHeader = ctx.Query("token")
		}

		if authHeader == "" {
			response.Fail(ctx, errors.ErrTokenMissing)
			ctx.Abort()
			return
		}

		// Parse the Bearer token.
		tokenString := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		}

		if tokenString == "" {
			response.Fail(ctx, errors.ErrTokenMissing)
			ctx.Abort()
			return
		}

		// Parse the token.
		claims, err := agentSvc.ParseToken(tokenString)
		if err != nil {
			response.Fail(ctx, errors.ErrTokenExpired)
			ctx.Abort()
			return
		}

		// Store agent information in the context.
		ctx.Set("agent_id", claims.AgentID)
		ctx.Set("agent_name", claims.AgentName)
		ctx.Set("tenant_id", claims.TenantID)
		ctx.Set("agent_claims", claims)

		ctx.Next(c)
	}
}

// GetAgentID returns the agent ID from the context.
func GetAgentID(c *app.RequestContext) string {
	val, ok := c.Get("agent_id")
	if !ok {
		return ""
	}
	if id, ok := val.(string); ok {
		return id
	}
	return ""
}

// GetTenantID returns the tenant ID from the context.
func GetTenantID(c *app.RequestContext) string {
	val, ok := c.Get("tenant_id")
	if !ok {
		return ""
	}
	if id, ok := val.(string); ok {
		return id
	}
	return ""
}

// GetAgentName returns the agent name from the context.
func GetAgentName(c *app.RequestContext) string {
	val, ok := c.Get("agent_name")
	if !ok {
		return ""
	}
	if name, ok := val.(string); ok {
		return name
	}
	return ""
}
