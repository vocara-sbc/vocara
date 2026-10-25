package handler

import (
	"context"
	"strings"

	"vocara/pkg/errors"
	"vocara/pkg/response"
	"vocara/services/agentcenter/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// AgentAuth 座席 JWT 鉴权中间件
func AgentAuth(agentSvc *service.AgentService) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		authHeader := string(ctx.GetHeader("Authorization"))
		if authHeader == "" {
			// 也从 query 参数获取
			authHeader = ctx.Query("token")
		}

		if authHeader == "" {
			response.Fail(ctx, errors.ErrTokenMissing)
			ctx.Abort()
			return
		}

		// 解析 Bearer token
		tokenString := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		}

		if tokenString == "" {
			response.Fail(ctx, errors.ErrTokenMissing)
			ctx.Abort()
			return
		}

		// 解析 token
		claims, err := agentSvc.ParseToken(tokenString)
		if err != nil {
			response.Fail(ctx, errors.ErrTokenExpired)
			ctx.Abort()
			return
		}

		// 将座席信息写入上下文
		ctx.Set("agent_id", claims.AgentID)
		ctx.Set("agent_name", claims.AgentName)
		ctx.Set("tenant_id", claims.TenantID)
		ctx.Set("agent_claims", claims)

		ctx.Next(c)
	}
}

// GetAgentID 从上下文获取座席ID
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

// GetTenantID 从上下文获取租户ID
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

// GetAgentName 从上下文获取座席名称
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
