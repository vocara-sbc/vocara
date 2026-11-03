// Package handler implements the HTTP handler layer.
// It processes HTTP requests and calls the service layer to perform business logic.
package handler

import (
	"context"
	"strconv"

	"vocara/admin/internal/service"
	"vocara/pkg/config"
	"vocara/pkg/errors"
	"vocara/pkg/middleware"
	"vocara/pkg/middleware/rbac"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	agentSvc  *service.AgentService
	tenantSvc *service.TenantService
	cfg       *config.Config
}

// NewAuthHandler creates an authentication handler.
func NewAuthHandler(cfg *config.Config, agentSvc *service.AgentService, tenantSvc *service.TenantService) *AuthHandler {
	return &AuthHandler{
		agentSvc:  agentSvc,
		tenantSvc: tenantSvc,
		cfg:       cfg,
	}
}

// LoginRequest contains login credentials.
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TenantCode string `json:"tenant_code"`
}

// LoginResponse contains a login token and user information.
type LoginResponse struct {
	Token    string   `json:"token"`
	UserInfo UserInfo `json:"user_info"`
}

// UserInfo contains user details.
type UserInfo struct {
	UserID      uint64   `json:"user_id"`
	Username    string   `json:"username"`
	Nickname    string   `json:"nickname"`
	Role        string   `json:"role"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Avatar      string   `json:"avatar"`
	Mobile      string   `json:"mobile"`
	Email       string   `json:"email"`
	Gender      int      `json:"gender"`
	TenantID    uint64   `json:"tenant_id"`
	TenantName  string   `json:"tenant_name"`
	TenantCode  string   `json:"tenant_code"`
	// HomePath is the landing page after login.
	HomePath string `json:"home_path"`
}

// Login authenticates an administrator.
// Administrators log in with an agent number for the admin or supervisor role.
func (h *AuthHandler) Login(ctx context.Context, c *app.RequestContext) {
	var req LoginRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.FailParamError(c, "参数错误: "+err.Error())
		return
	}

	if req.Username == "" || req.Password == "" {
		response.FailParamError(c, "用户名和密码不能为空")
		return
	}

	// Resolve the tenant ID.
	var tenantID uint64 = 1 // Default tenant.
	var tenantName string = "默认租户"
	var tenantCode string = "default"

	if req.TenantCode != "" {
		tenant, err := h.tenantSvc.GetByCode(ctx, req.TenantCode)
		if err != nil {
			response.Fail(c, errors.ErrUserNotFound.WithDetail("租户不存在"))
			return
		}
		if tenant.Status != 1 {
			response.Fail(c, errors.ErrForbidden.WithDetail("租户已被禁用"))
			return
		}
		tenantID = tenant.ID
		tenantName = tenant.TenantName
		tenantCode = tenant.TenantCode
	}

	// Authenticate the agent; all enabled agents can log in, and RBAC controls permissions.
	agent, err := h.agentSvc.Login(ctx, tenantID, req.Username, req.Password)
	if err != nil {
		response.Fail(c, errors.ErrAgentPasswordWrong)
		return
	}

	// Get permission codes.
	permissions := rbac.GetPermissions(agent.Role)

	// Determine the landing page after login.
	homePath := "/dashboard"
	if agent.Role == rbac.RoleAgent {
		homePath = "/workbench"
	}

	// Generate a token.
	token, err := middleware.GenerateTokenWithAgentNum(
		uint64ToString(agent.ID),
		agent.AgentNumber,
		agent.Role,
		agent.AgentNumber,              // Agent number used by push to route call events.
		uint64ToString(agent.TenantID), // Tenant ID used for push WS handshake routing and agent session ownership.
		h.cfg.App.SecretKey,
	)
	if err != nil {
		response.Fail(c, errors.ErrSystemError)
		return
	}

	// Return the response.
	response.OK(c, LoginResponse{
		Token: token,
		UserInfo: UserInfo{
			UserID:      agent.ID,
			Username:    agent.AgentNumber,
			Nickname:    agent.AgentName,
			Role:        agent.Role,
			Roles:       []string{agent.Role},
			Permissions: permissions,
			Avatar:      agent.Avatar,
			Mobile:      agent.Mobile,
			Email:       agent.Email,
			Gender:      agent.Gender,
			TenantID:    agent.TenantID,
			TenantName:  tenantName,
			TenantCode:  tenantCode,
			HomePath:    homePath,
		},
	})
}

// Logout logs out the current user.
func (h *AuthHandler) Logout(ctx context.Context, c *app.RequestContext) {
	// JWT is stateless, so the frontend clears the token.
	response.OKNoContent(c)
}

// ChangePasswordRequest contains the current and new passwords.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(ctx context.Context, c *app.RequestContext) {
	var req ChangePasswordRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.FailParamError(c, "参数错误: "+err.Error())
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		response.FailParamError(c, "旧密码和新密码不能为空")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	// Get the tenant ID.
	tenantID := getTenantID(ctx, c)

	id := parseUint64(userID)
	if err := h.agentSvc.UpdatePassword(ctx, tenantID, id, req.OldPassword, req.NewPassword); err != nil {
		// Preserve error codes and messages for validation failures such as insufficient strength or matching passwords; use the password error code when the current password does not match.
		if bizErr, ok := errors.IsBizError(err); ok {
			response.Fail(c, bizErr)
		} else {
			response.Fail(c, errors.WrapWithCode(err, errors.CodeAgentPasswordWrong, "密码修改失败"))
		}
		return
	}

	response.OKNoContent(c)
}

// GetUserInfo returns the current user's information.
func (h *AuthHandler) GetUserInfo(ctx context.Context, c *app.RequestContext) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	tenantID := getTenantID(ctx, c)
	id := parseUint64(userID)

	agent, err := h.agentSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, errors.ErrUserNotFound)
		return
	}

	// Get tenant information.
	tenant, _ := h.tenantSvc.GetByID(ctx, tenantID)
	tenantName := ""
	tenantCode := ""
	if tenant != nil {
		tenantName = tenant.TenantName
		tenantCode = tenant.TenantCode
	}

	response.OK(c, UserInfo{
		UserID:      agent.ID,
		Username:    agent.AgentNumber,
		Nickname:    agent.AgentName,
		Role:        agent.Role,
		Roles:       []string{agent.Role},
		Permissions: rbac.GetPermissions(agent.Role),
		Avatar:      agent.Avatar,
		Mobile:      agent.Mobile,
		Email:       agent.Email,
		Gender:      agent.Gender,
		TenantID:    agent.TenantID,
		TenantName:  tenantName,
		TenantCode:  tenantCode,
		HomePath:    homePathForRole(agent.Role),
	})
}

// UpdateProfileRequest contains self-service profile fields.
type UpdateProfileRequest struct {
	AgentName string `json:"agent_name"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	Gender    int    `json:"gender"`
}

// UpdateProfile updates the current agent's profile.
// Identity comes from the JWT; sensitive fields such as role, status, and extension cannot be changed through self-service updates.
func (h *AuthHandler) UpdateProfile(ctx context.Context, c *app.RequestContext) {
	var req UpdateProfileRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.FailParamError(c, "参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}
	tenantID := getTenantID(ctx, c)
	id := parseUint64(userID)

	dto := &service.ProfileUpdateDTO{
		AgentName: req.AgentName,
		Mobile:    req.Mobile,
		Email:     req.Email,
		Gender:    req.Gender,
	}
	if err := h.agentSvc.UpdateProfile(ctx, tenantID, id, dto); err != nil {
		response.Fail(c, WrapErr(err, "个人资料更新失败"))
		return
	}
	response.OKNoContent(c)
}

// homePathForRole returns the landing page for a role.
func homePathForRole(role string) string {
	if role == rbac.RoleAgent {
		return "/workbench"
	}
	return "/dashboard"
}

// ===== Utility Functions =====

// getTenantID returns the tenant ID from the token or context.
func getTenantID(ctx context.Context, c *app.RequestContext) uint64 {
	// Read the Hertz context first.
	if tid, ok := c.Get("tenant_id"); ok {
		if id, ok := tid.(uint64); ok && id > 0 {
			return id
		}
	}
	// Default the tenant ID to 1.
	return 1
}

// parseUint64 parses a uint64.
func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

// uint64ToString converts a uint64 to a string.
func uint64ToString(v uint64) string {
	return strconv.FormatUint(v, 10)
}
