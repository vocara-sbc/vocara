package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"vocara/pkg/logger"
	"vocara/pkg/utils"
)

const (
	// TraceIDHeader is the TraceID request header.
	TraceIDHeader = "X-Trace-ID"
	// TraceIDKey is the TraceID context key.
	TraceIDKey = "trace_id"
)

// TraceIDConfig contains TraceID middleware configuration.
type TraceIDConfig struct {
	HeaderKey string // Key in the request header
	ContextKey string // Key in the context
	GenFunc   func() string // Function that generates a TraceID
}

// DefaultTraceIDConfig is the default configuration.
var DefaultTraceIDConfig = TraceIDConfig{
	HeaderKey:  TraceIDHeader,
	ContextKey: TraceIDKey,
	GenFunc:    utils.UUID,
}

// TraceID returns TraceID middleware.
// It gets the TraceID from the request header or generates one if absent.
func TraceID() app.HandlerFunc {
	return TraceIDWithConfig(DefaultTraceIDConfig)
}

// TraceIDWithConfig creates TraceID middleware with the given configuration.
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
		// Get trace_id from the request header
		traceID := string(ctx.GetHeader(cfg.HeaderKey))
		if traceID == "" {
			// Generate a new trace_id
			traceID = cfg.GenFunc()
		}

		// Set it in the response header
		ctx.Header(cfg.HeaderKey, traceID)

		// Set it in the Hertz context
		ctx.Set("trace_id", traceID)

		// Set it in the Go context
		c = context.WithValue(c, logger.TraceIDKey, traceID)

		ctx.Next(c)
	}
}

// GetTraceID gets the TraceID from the context.
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
