package middleware

import (
	"context"
	stderrors "errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"

	bizerrors "vocara/pkg/errors"
	"vocara/pkg/response"
)

// JWTConfig contains JWT middleware configuration.
type JWTConfig struct {
	Secret     string   // Signing secret
	ContextKey string   // Context key for user information
	SkipPaths  []string // Paths that skip validation
	TokenType  string   // Token type; defaults to Bearer
}

// DefaultJWTConfig is the default JWT configuration.
var DefaultJWTConfig = JWTConfig{
	Secret:     "default-secret-change-me",
	ContextKey: "user",
	TokenType:  "Bearer",
}

// UserClaims contains JWT user claims.
type UserClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	// AgentNum is the agent number for agent roles, used by the push service to route call events by agent number.
	AgentNum string `json:"agent_num,omitempty"`
	// TenantID is the tenant ID required for push WebSocket handshake routing and agent session establishment.
	TenantID string `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

// JWT returns JWT authentication middleware.
func JWT(secret string) app.HandlerFunc {
	cfg := DefaultJWTConfig
	cfg.Secret = secret
	return JWTWithConfig(cfg)
}

// JWTWithConfig creates JWT middleware with the given configuration.
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
		// Skip the specified paths
		if skipPathMap[string(ctx.Path())] {
			ctx.Next(c)
			return
		}

		// Get the token from the Authorization header
		authHeader := string(ctx.GetHeader("Authorization"))
		if authHeader == "" {
			// Also get it from a query parameter
			authHeader = ctx.Query("token")
		}

		if authHeader == "" {
			response.Fail(ctx, bizerrors.ErrTokenMissing)
			ctx.Abort()
			return
		}

		// Parse the Bearer token
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

		// Parse the token
		claims := &UserClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Secret), nil
		})

		if err != nil || !token.Valid {
			// Determine whether the token is expired or invalid
			if stderrors.Is(err, jwt.ErrTokenExpired) {
				response.Fail(ctx, bizerrors.ErrTokenExpired)
			} else {
				response.Fail(ctx, bizerrors.ErrTokenInvalid)
			}
			ctx.Abort()
			return
		}

		// Store user information in the Hertz context
		ctx.Set("user_claims", claims)
		ctx.Set("user_id", claims.UserID)
		ctx.Set("username", claims.Username)
		ctx.Set("user_role", claims.Role)
		ctx.Set("agent_num", claims.AgentNum)

		ctx.Next(c)
	}
}

// GenerateToken generates a JWT token.
func GenerateToken(userID, username, role, secret string) (string, error) {
	return GenerateTokenWithAgentNum(userID, username, role, "", "", secret)
}

// GenerateTokenWithAgentNum generates a JWT token with an agent number and tenant ID.
// agentNum applies only to agent roles and is used by the push service to route call events by agent number;
// tenantID determines tenant routing and agent session ownership during the push WebSocket handshake.
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

// GetUserClaims gets user information from the context.
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

// GetUserID gets the user ID from the context.
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

// GetUsername gets the username from the context.
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

// GetUserRole gets the user role from the context.
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

// GetAgentNum gets the agent number from the context for agent roles.
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

// InternalAuth returns internal service authentication middleware.
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
