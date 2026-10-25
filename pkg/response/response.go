// Package response 提供统一的 HTTP 响应封装
// 基于 Hertz 框架，支持成功/失败响应、分页响应等
package response

import (
	"vocara/pkg/errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ===== Hertz 方式（推荐）=====

// OK 成功响应
func OK(c *app.RequestContext, data interface{}) {
	c.JSON(consts.StatusOK, Response{
		Code:    errors.CodeSuccess,
		Message: errors.MsgSuccess,
		Data:    data,
	})
}

// Success 成功响应（OK 的别名）
func Success(c *app.RequestContext, data interface{}) {
	OK(c, data)
}

// Fail 失败响应（业务错误）
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

// FailWithMsg 失败响应，带自定义消息
func FailWithMsg(c *app.RequestContext, code int, msg string) {
	status := errors.CodeToHTTPStatus(code)
	c.JSON(status, Response{
		Code:    code,
		Message: msg,
	})
}

// FailWithDetail 失败响应，带详情
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

// Error 通用错误响应（自动判断是否业务错误）
func Error(c *app.RequestContext, err error) {
	if bizErr, ok := errors.IsBizError(err); ok {
		Fail(c, bizErr)
		return
	}
	// 包装为系统错误
	bizErr := errors.Wrap(err)
	Fail(c, bizErr)
}

// ===== context.Context 方式 =====
//
// 注意：hertz v0.10 的 *app.RequestContext 并未实现 context.Context
// （缺少 Deadline 方法），因此下列兼容方法接收 any，
// 仅当传入值本身是 *app.RequestContext 时才输出响应。
// 业务代码应优先使用 OK / Fail 等直接接收 *app.RequestContext 的方法。

// OKCtx 成功响应（兼容方式，c 需为 *app.RequestContext）
func OKCtx(c any, data interface{}) {
	if rc, ok := c.(*app.RequestContext); ok {
		OK(rc, data)
	}
}

// FailCtx 失败响应（兼容方式，c 需为 *app.RequestContext）
func FailCtx(c any, err *errors.BizError) {
	if rc, ok := c.(*app.RequestContext); ok {
		Fail(rc, err)
	}
}

// FailWithMsgCtx 失败响应带消息（兼容方式，c 需为 *app.RequestContext）
func FailWithMsgCtx(c any, code int, msg string) {
	if rc, ok := c.(*app.RequestContext); ok {
		FailWithMsg(rc, code, msg)
	}
}

// ===== 分页响应 =====

// PageData 分页数据
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Pages    int         `json:"pages"`
}

// PageSuccess 分页成功响应
func PageSuccess(c *app.RequestContext, list interface{}, total int64, page, pageSize, pages int) {
	OK(c, PageData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	})
}

// PageOK 分页成功响应（PageSuccess 的别名）
func PageOK(c *app.RequestContext, list interface{}, total int64, page, pageSize, pages int) {
	PageSuccess(c, list, total, page, pageSize, pages)
}

// ===== 常用快捷方法 =====

// OKNoContent 成功响应（无数据）
func OKNoContent(c *app.RequestContext) {
	OK(c, nil)
}

// FailParamError 参数错误响应
func FailParamError(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgParamError
	}
	FailWithMsg(c, errors.CodeParamError, msg)
}

// FailUnauthorized 未授权响应
func FailUnauthorized(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgUnauthorized
	}
	FailWithMsg(c, errors.CodeUnauthorized, msg)
}

// FailForbidden 禁止访问响应
func FailForbidden(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgForbidden
	}
	FailWithMsg(c, errors.CodeForbidden, msg)
}

// FailNotFound 资源不存在响应
func FailNotFound(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgNotFound
	}
	FailWithMsg(c, errors.CodeNotFound, msg)
}

// FailSystemError 系统错误响应
func FailSystemError(c *app.RequestContext, msg string) {
	if msg == "" {
		msg = errors.MsgSystemError
	}
	FailWithMsg(c, errors.CodeSystemError, msg)
}
