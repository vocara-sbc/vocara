package middleware

import (
	"context"
	stderrors "errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"

	bizerrors "vocara/pkg/errors"
	"vocara/pkg/response"
)

// JWTConfig JWT 中间件配置
type JWTConfig struct {
	Secret     string   // 签名密钥
	ContextKey string   // 用户信息在上下文中的 key
	SkipPaths  []string // 跳过验证的路径
	TokenType  string   // Token 类型，默认 Bearer
}

// DefaultJWTConfig 默认 JWT 配置
var DefaultJWTConfig = JWTConfig{
	Secret:     "default-secret-change-me",
	ContextKey: "user",
	TokenType:  "Bearer",
}

// UserClaims JWT 用户声明
type UserClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	// AgentNum 座席工号（座席角色时填写，用于 push 服务按工号路由通话事件）
	AgentNum string `json:"agent_num,omitempty"`
	// TenantID 租户 ID（push WS 握手路由与座席会话建立需要）
	TenantID string `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

// JWT JWT 鉴权中间件
func JWT(secret string) app.HandlerFunc {
	cfg := DefaultJWTConfig
	cfg.Secret = secret
	return JWTWithConfig(cfg)
}

// JWTWithConfig 使用配置创建 JWT 中间件
func JWTWithConfig(cfg JWTConfig) app.HandlerFunc {
	if cfg.Secret == "" {
		cfg.Secret = DefaultJWTConfig.Secret
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = DefaultJWTConfig.ContextKey
	}
	if cfg.TokenType == "" {
		cfg.TokenType = DefaultJWTConfig.TokenType
	}

	skipPathMap := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipPathMap[path] = true
	}

	return func(c context.Context, ctx *app.RequestContext) {
		// 跳过指定路径
		if skipPathMap[string(ctx.Path())] {
			ctx.Next(c)
			return
		}

		// 从 Authorization header 获取 token
		authHeader := string(ctx.GetHeader("Authorization"))
		if authHeader == "" {
			// 也从 query 参数获取
			authHeader = ctx.Query("token")
		}

		if authHeader == "" {
			response.Fail(ctx, bizerrors.ErrTokenMissing)
			ctx.Abort()
			return
		}

		// 解析 Bearer token
		tokenString := authHeader
		prefixLen := len(cfg.TokenType) + 1
		if len(authHeader) > prefixLen &&
			authHeader[:len(cfg.TokenType)] == cfg.TokenType {
			tokenString = authHeader[prefixLen:]
		}

		if tokenString == "" {
			response.Fail(ctx, bizerrors.ErrTokenMissing)
			ctx.Abort()
			return
		}

		// 解析 token
		claims := &UserClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Secret), nil
		})

		if err != nil || !token.Valid {
			// 判断是过期还是无效
			if stderrors.Is(err, jwt.ErrTokenExpired) {
				response.Fail(ctx, bizerrors.ErrTokenExpired)
			} else {
				response.Fail(ctx, bizerrors.ErrTokenInvalid)
			}
			ctx.Abort()
			return
		}

		// 将用户信息存入 Hertz 上下文
		ctx.Set("user_claims", claims)
		ctx.Set("user_id", claims.UserID)
		ctx.Set("username", claims.Username)
		ctx.Set("user_role", claims.Role)
		ctx.Set("agent_num", claims.AgentNum)

		ctx.Next(c)
	}
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID, username, role, secret string) (string, error) {
	return GenerateTokenWithAgentNum(userID, username, role, "", "", secret)
}

// GenerateTokenWithAgentNum 生成带座席工号与租户 ID 的 JWT Token
// agentNum 仅座席角色有效，用于 push 服务按工号路由通话事件；
// tenantID 用于 push WS 握手时确定租户路由与座席会话归属。
func GenerateTokenWithAgentNum(userID, username, role, agentNum, tenantID, secret string) (string, error) {
	claims := UserClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		AgentNum: agentNum,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "vocara",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GetUserClaims 从上下文获取用户信息
func GetUserClaims(ctx *app.RequestContext) *UserClaims {
	val, exists := ctx.Get("user_claims")
	if !exists {
		return nil
	}
	claims, ok := val.(*UserClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetUserID 从上下文获取用户 ID
func GetUserID(ctx *app.RequestContext) string {
	val, exists := ctx.Get("user_id")
	if !exists {
		return ""
	}
	id, ok := val.(string)
	if !ok {
		return ""
	}
	return id
}

// GetUsername 从上下文获取用户名
func GetUsername(ctx *app.RequestContext) string {
	val, exists := ctx.Get("username")
	if !exists {
		return ""
	}
	name, ok := val.(string)
	if !ok {
		return ""
	}
	return name
}

// GetUserRole 从上下文获取用户角色
func GetUserRole(ctx *app.RequestContext) string {
	val, exists := ctx.Get("user_role")
	if !exists {
		return ""
	}
	role, ok := val.(string)
	if !ok {
		return ""
	}
	return role
}

// GetAgentNum 从上下文获取座席工号（座席角色时有效）
func GetAgentNum(ctx *app.RequestContext) string {
	val, exists := ctx.Get("agent_num")
	if !exists {
		return ""
	}
	num, ok := val.(string)
	if !ok {
		return ""
	}
	return num
}

// InternalAuth 内部服务鉴权中间件
func InternalAuth(token string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		internalToken := string(ctx.GetHeader("X-Internal-Token"))
		if internalToken == "" || internalToken != token {
			response.Fail(ctx, bizerrors.ErrForbidden)
			ctx.Abort()
			return
		}
		ctx.Next(c)
	}
}
