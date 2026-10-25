// Package handler HTTP 处理器层
// 处理 HTTP 请求，调用 service 层完成业务逻辑
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

// AuthHandler 认证处理器
type AuthHandler struct {
	agentSvc  *service.AgentService
	tenantSvc *service.TenantService
	cfg       *config.Config
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(cfg *config.Config, agentSvc *service.AgentService, tenantSvc *service.TenantService) *AuthHandler {
	return &AuthHandler{
		agentSvc:  agentSvc,
		tenantSvc: tenantSvc,
		cfg:       cfg,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TenantCode string `json:"tenant_code"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token    string   `json:"token"`
	UserInfo UserInfo `json:"user_info"`
}

// UserInfo 用户信息
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
	// HomePath 登录后跳转的首页路径
	HomePath string `json:"home_path"`
}

// Login 管理员登录
// 管理员可以用座席工号登录（admin/supervisor 角色）
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

	// 获取租户ID
	var tenantID uint64 = 1 // 默认租户
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

	// 座席登录（所有启用状态的座席都可以登录，权限通过 RBAC 控制）
	agent, err := h.agentSvc.Login(ctx, tenantID, req.Username, req.Password)
	if err != nil {
		response.Fail(c, errors.ErrAgentPasswordWrong)
		return
	}

	// 获取权限码列表
	permissions := rbac.GetPermissions(agent.Role)

	// 计算登录后跳转的首页
	homePath := "/dashboard"
	if agent.Role == rbac.RoleAgent {
		homePath = "/workbench"
	}

	// 生成 Token
	token, err := middleware.GenerateTokenWithAgentNum(
		uint64ToString(agent.ID),
		agent.AgentNumber,
		agent.Role,
		agent.AgentNumber,              // 座席工号，用于 push 服务按工号路由通话事件
		uint64ToString(agent.TenantID), // 租户 ID，push WS 握手路由与座席会话归属需要
		h.cfg.App.SecretKey,
	)
	if err != nil {
		response.Fail(c, errors.ErrSystemError)
		return
	}

	// 返回响应
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

// Logout 登出
func (h *AuthHandler) Logout(ctx context.Context, c *app.RequestContext) {
	// JWT 是无状态的，前端清除 token 即可
	response.OKNoContent(c)
}

// ChangePassword 修改密码
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

	// 获取租户ID
	tenantID := getTenantID(ctx, c)

	id := parseUint64(userID)
	if err := h.agentSvc.UpdatePassword(ctx, tenantID, id, req.OldPassword, req.NewPassword); err != nil {
		// 参数类错误（强度不足、新旧相同）保留原错误码与提示，旧密码不匹配走密码错误码
		if bizErr, ok := errors.IsBizError(err); ok {
			response.Fail(c, bizErr)
		} else {
			response.Fail(c, errors.WrapWithCode(err, errors.CodeAgentPasswordWrong, "密码修改失败"))
		}
		return
	}

	response.OKNoContent(c)
}

// GetUserInfo 获取当前用户信息
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

	// 获取租户信息
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

// UpdateProfileRequest 自助资料更新请求
type UpdateProfileRequest struct {
	AgentName string `json:"agent_name"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	Gender    int    `json:"gender"`
}

// UpdateProfile 更新当前登录座席的个人资料。
// 身份取自 JWT，角色、状态、分机等敏感字段不接受自助修改。
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

// homePathForRole 根据角色返回登录后首页路径
func homePathForRole(role string) string {
	if role == rbac.RoleAgent {
		return "/workbench"
	}
	return "/dashboard"
}

// ===== 工具函数 =====

// getTenantID 获取租户ID（从 token 或上下文中）
func getTenantID(ctx context.Context, c *app.RequestContext) uint64 {
	// 先从 Hertz 上下文取
	if tid, ok := c.Get("tenant_id"); ok {
		if id, ok := tid.(uint64); ok && id > 0 {
			return id
		}
	}
	// 默认租户 ID 为 1
	return 1
}

// parseUint64 解析 uint64
func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

// uint64ToString uint64 转字符串
func uint64ToString(v uint64) string {
	return strconv.FormatUint(v, 10)
}
