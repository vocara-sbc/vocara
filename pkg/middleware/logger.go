package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"

	"vocara/pkg/logger"
)

// RequestLoggerConfig contains request logging middleware configuration.
type RequestLoggerConfig struct {
	SkipPaths       []string // Paths excluded from logging
	SlowThreshold   int      // Slow request threshold in milliseconds; 0 disables slow request logging
	EnableRequestBody  bool  // Whether to log request bodies
	EnableResponseBody bool  // Whether to log response bodies
	MaxBodySize     int      // Maximum body size to log in bytes
}

// DefaultRequestLoggerConfig is the default configuration.
var DefaultRequestLoggerConfig = RequestLoggerConfig{
	SlowThreshold:  1000, // 1 second
	MaxBodySize:    1024, // 1KB
}

// RequestLogger returns request logging middleware.
func RequestLogger() app.HandlerFunc {
	return RequestLoggerWithConfig(DefaultRequestLoggerConfig)
}

// RequestLoggerWithConfig creates request logging middleware with the given configuration.
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

		// Skip the specified paths
		if skipPathMap[path] {
			ctx.Next(c)
			return
		}

		// Process the request
		ctx.Next(c)

		// Calculate the elapsed time
		duration := time.Since(start)
		statusCode := ctx.Response.StatusCode()

		// Assemble fields
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("ip", ctx.ClientIP()),
		}

		// Record trace_id
		if traceID := GetTraceID(ctx); traceID != "" {
			fields = append(fields, zap.String("trace_id", traceID))
		}

		// Determine the log level from the status code and elapsed time
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
