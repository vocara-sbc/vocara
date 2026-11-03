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

// GetTenantID returns the tenant ID.
// It prefers the context value and defaults to 1.
func GetTenantID(ctx context.Context, c *app.RequestContext) uint64 {
	if tid, ok := c.Get("tenant_id"); ok {
		if id, ok := tid.(uint64); ok && id > 0 {
			return id
		}
	}
	return 1
}

// GetUserIDFromCtx returns the user ID from the context.
func GetUserIDFromCtx(c *app.RequestContext) uint64 {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return 0
	}
	id, _ := strconv.ParseUint(userID, 10, 64)
	return id
}

// GetUsernameFromCtx returns the username from the context.
func GetUsernameFromCtx(c *app.RequestContext) string {
	return middleware.GetUsername(c)
}

// ParseUintParam parses a uint64 parameter from the URL.
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

// ParseIntParam parses an int parameter from the URL.
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

// ParseIntPtrParam parses an optional *int URL parameter for status filtering.
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

// PageSuccess returns a successful paginated response.
func PageSuccess(c *app.RequestContext, list interface{}, total int64, page, pageSize int) {
	pages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		pages++
	}
	response.PageSuccess(c, list, total, page, pageSize, pages)
}

// ValidateRequired validates required parameters.
func ValidateRequired(c *app.RequestContext, fields map[string]string) bool {
	for k, v := range fields {
		if v == "" {
			response.FailParamError(c, k+"不能为空")
			return false
		}
	}
	return true
}

// BindError returns a parameter-binding error response.
func BindError(c *app.RequestContext, err error) {
	response.FailParamError(c, "参数绑定失败: "+err.Error())
}

// StringsToUint64s converts a string slice to a uint64 slice.
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

// CheckReportEnabled validates that the report export module is licensed and returns a module error otherwise.
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

// WrapErr wraps an error as a business error.
func WrapErr(err error, defaultMsg string) *errors.BizError {
	if bizErr, ok := errors.IsBizError(err); ok {
		return bizErr
	}
	return errors.WrapWithCode(err, errors.CodeSystemError, defaultMsg)
}
