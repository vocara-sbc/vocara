package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"

	"vocara/pkg/logger"
)

// RequestLoggerConfig 请求日志中间件配置
type RequestLoggerConfig struct {
	SkipPaths       []string // 跳过日志的路径
	SlowThreshold   int      // 慢请求阈值（毫秒），0 表示不记录
	EnableRequestBody  bool  // 是否记录请求体
	EnableResponseBody bool  // 是否记录响应体
	MaxBodySize     int      // 记录的最大 body 大小（字节）
}

// DefaultRequestLoggerConfig 默认配置
var DefaultRequestLoggerConfig = RequestLoggerConfig{
	SlowThreshold:  1000, // 1秒
	MaxBodySize:    1024, // 1KB
}

// RequestLogger 请求日志中间件
func RequestLogger() app.HandlerFunc {
	return RequestLoggerWithConfig(DefaultRequestLoggerConfig)
}

// RequestLoggerWithConfig 使用配置创建请求日志中间件
func RequestLoggerWithConfig(cfg RequestLoggerConfig) app.HandlerFunc {
	skipPathMap := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipPathMap[path] = true
	}

	if cfg.SlowThreshold <= 0 {
		cfg.SlowThreshold = DefaultRequestLoggerConfig.SlowThreshold
	}
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = DefaultRequestLoggerConfig.MaxBodySize
	}

	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()
		path := string(ctx.Path())
		method := string(ctx.Method())

		// 跳过指定路径
		if skipPathMap[path] {
			ctx.Next(c)
			return
		}

		// 处理请求
		ctx.Next(c)

		// 计算耗时
		duration := time.Since(start)
		statusCode := ctx.Response.StatusCode()

		// 组装字段
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("ip", ctx.ClientIP()),
		}

		// 记录 trace_id
		if traceID := GetTraceID(ctx); traceID != "" {
			fields = append(fields, zap.String("trace_id", traceID))
		}

		// 根据状态码和耗时决定日志级别
		switch {
		case statusCode >= 500:
			logger.ErrorCtx(c, "request failed", fields...)
		case statusCode >= 400:
			logger.WarnCtx(c, "request warning", fields...)
		case duration > time.Duration(cfg.SlowThreshold)*time.Millisecond:
			fields = append(fields, zap.Bool("slow", true))
			logger.WarnCtx(c, "slow request", fields...)
		default:
			logger.InfoCtx(c, "request completed", fields...)
		}
	}
}
