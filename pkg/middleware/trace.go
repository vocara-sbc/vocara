package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"vocara/pkg/logger"
	"vocara/pkg/utils"
)

const (
	// TraceIDHeader TraceID 请求头
	TraceIDHeader = "X-Trace-ID"
	// TraceIDKey TraceID 上下文中的 key
	TraceIDKey = "trace_id"
)

// TraceIDConfig TraceID 中间件配置
type TraceIDConfig struct {
	HeaderKey string // 请求头中的 key
	ContextKey string // 上下文中的 key
	GenFunc   func() string // 生成 TraceID 的函数
}

// DefaultTraceIDConfig 默认配置
var DefaultTraceIDConfig = TraceIDConfig{
	HeaderKey:  TraceIDHeader,
	ContextKey: TraceIDKey,
	GenFunc:    utils.UUID,
}

// TraceID TraceID 中间件
// 从请求头获取 TraceID，如果没有则生成一个
func TraceID() app.HandlerFunc {
	return TraceIDWithConfig(DefaultTraceIDConfig)
}

// TraceIDWithConfig 使用配置创建 TraceID 中间件
func TraceIDWithConfig(cfg TraceIDConfig) app.HandlerFunc {
	if cfg.HeaderKey == "" {
		cfg.HeaderKey = DefaultTraceIDConfig.HeaderKey
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = DefaultTraceIDConfig.ContextKey
	}
	if cfg.GenFunc == nil {
		cfg.GenFunc = DefaultTraceIDConfig.GenFunc
	}

	return func(c context.Context, ctx *app.RequestContext) {
		// 从请求头获取 trace_id
		traceID := string(ctx.GetHeader(cfg.HeaderKey))
		if traceID == "" {
			// 生成新的 trace_id
			traceID = cfg.GenFunc()
		}

		// 设置到响应头
		ctx.Header(cfg.HeaderKey, traceID)

		// 设置到 Hertz 上下文
		ctx.Set("trace_id", traceID)

		// 设置到 Go context
		c = context.WithValue(c, logger.TraceIDKey, traceID)

		ctx.Next(c)
	}
}

// GetTraceID 从上下文中获取 TraceID
func GetTraceID(ctx *app.RequestContext) string {
	val, exists := ctx.Get("trace_id")
	if !exists {
		return ""
	}
	id, ok := val.(string)
	if !ok {
		return ""
	}
	return id
}
