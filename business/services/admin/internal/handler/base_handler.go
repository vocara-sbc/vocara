package handler

import (
	"context"
	"strconv"
	"strings"

	"vocara/pkg/errors"
	"vocara/pkg/license"
	"vocara/pkg/middleware"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// GetTenantID 获取租户ID
// 优先从上下文获取，默认返回 1
func GetTenantID(ctx context.Context, c *app.RequestContext) uint64 {
	if tid, ok := c.Get("tenant_id"); ok {
		if id, ok := tid.(uint64); ok && id > 0 {
			return id
		}
	}
	return 1
}

// GetUserIDFromCtx 获取用户ID
func GetUserIDFromCtx(c *app.RequestContext) uint64 {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return 0
	}
	id, _ := strconv.ParseUint(userID, 10, 64)
	return id
}

// GetUsernameFromCtx 获取用户名
func GetUsernameFromCtx(c *app.RequestContext) string {
	return middleware.GetUsername(c)
}

// ParseUintParam 解析 URL 中的 uint64 参数
func ParseUintParam(c *app.RequestContext, key string) uint64 {
	val := c.Param(key)
	if val == "" {
		val = c.Query(key)
	}
	if val == "" {
		return 0
	}
	id, _ := strconv.ParseUint(val, 10, 64)
	return id
}

// ParseIntParam 解析 URL 中的 int 参数
func ParseIntParam(c *app.RequestContext, key string) int {
	val := c.Param(key)
	if val == "" {
		val = c.Query(key)
	}
	if val == "" {
		return 0
	}
	v, _ := strconv.Atoi(val)
	return v
}

// ParseIntPtrParam 解析 URL 中的 *int 参数（用于可选状态过滤）
func ParseIntPtrParam(c *app.RequestContext, key string) *int {
	val := c.Query(key)
	if val == "" {
		return nil
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return nil
	}
	return &v
}

// PageSuccess 分页成功响应
func PageSuccess(c *app.RequestContext, list interface{}, total int64, page, pageSize int) {
	pages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		pages++
	}
	response.PageSuccess(c, list, total, page, pageSize, pages)
}

// ValidateRequired 校验必填参数
func ValidateRequired(c *app.RequestContext, fields map[string]string) bool {
	for k, v := range fields {
		if v == "" {
			response.FailParamError(c, k+"不能为空")
			return false
		}
	}
	return true
}

// BindError 绑定参数错误响应
func BindError(c *app.RequestContext, err error) {
	response.FailParamError(c, "参数绑定失败: "+err.Error())
}

// StringsToUint64s 字符串数组转 uint64 数组
func StringsToUint64s(strs []string) []uint64 {
	var result []uint64
	for _, s := range strs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			continue
		}
		result = append(result, v)
	}
	return result
}

// CheckReportEnabled 校验报表导出模块是否已授权，未授权返回功能模块错误。
func CheckReportEnabled(licenseManager *license.Manager) error {
	if licenseManager == nil {
		return nil
	}
	if !licenseManager.ModuleEnabled(license.ModuleReport) {
		return errors.NewBizError(errors.CodeLicenseModuleDisabled, errors.MsgLicenseModuleDisabled).
			WithDetail("报表导出功能未授权")
	}
	return nil
}

// WrapErr 包装错误为业务错误
func WrapErr(err error, defaultMsg string) *errors.BizError {
	if bizErr, ok := errors.IsBizError(err); ok {
		return bizErr
	}
	return errors.WrapWithCode(err, errors.CodeSystemError, defaultMsg)
}
