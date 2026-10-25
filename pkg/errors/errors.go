// Package errors 提供业务错误码封装
// 自定义业务错误类型，包含错误码、错误消息、HTTP 状态码
package errors

import (
	"errors"
	"fmt"
)

// BizError 业务错误类型
type BizError struct {
	Code       int    `json:"code"`              // 错误码
	Message    string `json:"message"`           // 错误消息
	Detail     string `json:"detail,omitempty"`  // 详细信息
	HTTPStatus int    `json:"-"`                 // HTTP 状态码
	cause      error  // 原始错误
}

// Error 实现 error 接口
func (e *BizError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Unwrap
func (e *BizError) Unwrap() error {
	return e.cause
}

// WithDetail 设置详细信息并返回自身
func (e *BizError) WithDetail(detail string) *BizError {
	e.Detail = detail
	return e
}

// WithDetailf 设置格式化的详细信息并返回自身
func (e *BizError) WithDetailf(format string, args ...interface{}) *BizError {
	e.Detail = fmt.Sprintf(format, args...)
	return e
}

// WithCause 设置原始错误并返回自身
func (e *BizError) WithCause(err error) *BizError {
	e.cause = err
	if e.Detail == "" && err != nil {
		e.Detail = err.Error()
	}
	return e
}

// NewBizError 创建新的业务错误
func NewBizError(code int, msg string) *BizError {
	return &BizError{
		Code:       code,
		Message:    msg,
		HTTPStatus: codeToHTTPStatus(code),
	}
}

// NewBizErrorWithStatus 创建带 HTTP 状态码的业务错误
func NewBizErrorWithStatus(code int, msg string, httpStatus int) *BizError {
	return &BizError{
		Code:       code,
		Message:    msg,
		HTTPStatus: httpStatus,
	}
}

// Wrap 将普通错误包装为业务错误（系统错误）
func Wrap(err error) *BizError {
	if err == nil {
		return nil
	}
	// 如果已经是 BizError，直接返回
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return &BizError{
		Code:       CodeSystemError,
		Message:    MsgSystemError,
		Detail:     err.Error(),
		HTTPStatus: 500,
		cause:      err,
	}
}

// WrapWithCode 将普通错误包装为指定错误码的业务错误
func WrapWithCode(err error, code int, msg string) *BizError {
	if err == nil {
		return nil
	}
	return &BizError{
		Code:       code,
		Message:    msg,
		Detail:     err.Error(),
		HTTPStatus: codeToHTTPStatus(code),
		cause:      err,
	}
}

// IsBizError 判断错误是否为业务错误并返回
func IsBizError(err error) (*BizError, bool) {
	if err == nil {
		return nil, false
	}
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr, true
	}
	return nil, false
}

// IsCode 判断错误码是否匹配
func (e *BizError) IsCode(code int) bool {
	return e.Code == code
}

// CodeToHTTPStatus 根据错误码推断 HTTP 状态码。
// 错误码规则：
//   0        -> 200 OK
//   40000-40099 -> 400 Bad Request
//   40100-40199 -> 401 Unauthorized
//   40300-40399 -> 403 Forbidden
//   40400-40499 -> 404 Not Found
//   40900-40999 -> 409 Conflict
//   42900-42999 -> 429 Too Many Requests
//   50000-59999 -> 500 Internal Server Error
//   70001-70003 -> 402 Payment Required
//   70004-70007 -> 403 Forbidden
//   其他      -> 200 OK
func CodeToHTTPStatus(code int) int {
	return codeToHTTPStatus(code)
}

func codeToHTTPStatus(code int) int {
	switch {
	case code == 0:
		return 200
	case code >= 40000 && code < 40100:
		return 400
	case code >= 40100 && code < 40200:
		return 401
	case code >= 40300 && code < 40400:
		return 403
	case code >= 40400 && code < 40500:
		return 404
	case code >= 40900 && code < 41000:
		return 409
	case code >= 42900 && code < 43000:
		return 429
	case code >= 50000 && code < 60000:
		return 500
	case code >= 70001 && code < 70004:
		return 402
	case code >= 70004 && code < 70008:
		return 403
	default:
		return 200
	}
}
