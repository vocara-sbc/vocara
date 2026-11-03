// Package response provides unified HTTP response wrappers.
// Based on the Hertz framework, it supports success, failure, paginated, and other responses.
package response

import (
	"vocara/pkg/errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Response is the unified response structure.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ===== Hertz style (recommended) =====

// OK writes a success response.
func OK(c *app.RequestContext, data interface{}) {
	c.JSON(consts.StatusOK, Response{
		Code:    errors.CodeSuccess,
		Message: errors.MsgSuccess,
		Data:    data,
	})
}

// Success writes a success response and is an alias for OK.
func Success(c *app.RequestContext, data interface{}) {
	OK(c, data)
}

// Fail writes a business error response.
func Fail(c *app.RequestContext, err *errors.BizError) {
	if err != nil {
		status := err.HTTPStatus
		if status == 0 {
			status = consts.StatusOK
		}
		c.JSON(status, Response{
			Code:    err.Code,
			Message: err.Message,
			Data:    nil,
		})
		return
	}
	c.JSON(consts.StatusOK, Response{
		Code:    errors.CodeSystemError,
		Message: errors.MsgSystemError,
	})
}

// FailWithMsg writes a failure response with a custom message.
func FailWithMsg(c *app.RequestContext, code int, msg string) {
	status := errors.CodeToHTTPStatus(code)
	c.JSON(status, Response{
		Code:    code,
		Message: msg,
	})
}

// FailWithDetail writes a failure response with details.
func FailWithDetail(c *app.RequestContext, err *errors.BizError, detail string) {
	if err != nil {
		status := err.HTTPStatus
		if status == 0 {
			status = consts.StatusOK
		}
		c.JSON(status, Response{
			Code:    err.Code,
			Message: err.Message,
			Data:    detail,
		})
		return
	}
	c.JSON(consts.StatusOK, Response{
		Code:    errors.CodeSystemError,
		Message: errors.MsgSystemError,
	})
}

// Error writes a generic error response and automatically determines whether the error is a business error.
func Error(c *app.RequestContext, err error) {
	if bizErr, ok := errors.IsBizError(err); ok {
		Fail(c, bizErr)
		return
	}
	// Wrap it as a system error
	bizErr := errors.Wrap(err)
	Fail(c, bizErr)
}

// ===== context.Context style =====
//
// Note: hertz v0.10's *app.RequestContext does not implement context.Context
// because it lacks a Deadline method, so the following compatibility methods accept any
// and write a response only when the value is itself a *app.RequestContext.
// Business code should prefer OK, Fail, and other methods that accept *app.RequestContext directly.

// OKCtx writes a success response in compatibility mode; c must be a *app.RequestContext.
func OKCtx(c any, data interface{}) {
	if rc, ok := c.(*app.RequestContext); ok {
		OK(rc, data)
	}
}

// FailCtx writes a failure response in compatibility mode; c must be a *app.RequestContext.
func FailCtx(c any, err *errors.BizError) {
	if rc, ok := c.(*app.RequestContext); ok {
		Fail(rc, err)
	}
}

// FailWithMsgCtx writes a failure response with a message in compatibility mode; c must be a *app.RequestContext.
func FailWithMsgCtx(c any, code int, msg string) {
	if rc, ok := c.(*app.RequestContext); ok {
		FailWithMsg(rc, code, msg)
	}
}

// ===== Paginated responses =====

// PageData contains paginated data.
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Pages    int         `json:"pages"`
}

// PageSuccess writes a paginated success response.
func PageSuccess(c *app.RequestContext, list interface{}, total int64, page, pageSize, pages int) {
	OK(c, PageData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	})
}

// PageOK writes a paginated success response and is an alias for PageSuccess.
func PageOK(c *app.RequestContext, list interface{}, total int64, page, pageSize, pages int) {
	PageSuccess(c, list, total, page, pageSize, pages)
}

// ===== Common shortcuts =====

// OKNoContent writes a success response without data.
func OKNoContent(c *app.RequestContext) {
	OK(c, nil)
}

// FailParamError writes a parameter error response.
func FailParamError(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgParamError
	}
	FailWithMsg(c, errors.CodeParamError, msg)
}

// FailUnauthorized writes an unauthorized response.
func FailUnauthorized(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgUnauthorized
	}
	FailWithMsg(c, errors.CodeUnauthorized, msg)
}

// FailForbidden writes a forbidden response.
func FailForbidden(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgForbidden
	}
	FailWithMsg(c, errors.CodeForbidden, msg)
}

// FailNotFound writes a resource-not-found response.
func FailNotFound(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgNotFound
	}
	FailWithMsg(c, errors.CodeNotFound, msg)
}

// FailSystemError writes a system error response.
func FailSystemError(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgSystemError
	}
	FailWithMsg(c, errors.CodeSystemError, msg)
}
