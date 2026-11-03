package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// CORSConfig contains CORS middleware configuration.
type CORSConfig struct {
	AllowOrigins     string // Allowed origins; * allows all
	AllowMethods     string // Allowed methods
	AllowHeaders     string // Allowed request headers
	AllowCredentials bool   // Whether to allow credentials
	ExposeHeaders    string // Exposed response headers
	MaxAge           int    // Preflight request cache time in seconds
}

// DefaultCORSConfig is the default CORS configuration.
var DefaultCORSConfig = CORSConfig{
	AllowOrigins: "*",
	AllowMethods: "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD",
	AllowHeaders: "Content-Type, Authorization, X-Requested-With, X-Trace-ID, X-Internal-Token",
	MaxAge:       86400, // 24 hours
}

// CORS returns cross-origin middleware.
func CORS() app.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig)
}

// CORSWithConfig creates CORS middleware with the given configuration.
func CORSWithConfig(cfg CORSConfig) app.HandlerFunc {
	if cfg.AllowOrigins == "" {
		cfg.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	if cfg.AllowMethods == "" {
		cfg.AllowMethods = DefaultCORSConfig.AllowMethods
	}
	if cfg.AllowHeaders == "" {
		cfg.AllowHeaders = DefaultCORSConfig.AllowHeaders
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = DefaultCORSConfig.MaxAge
	}

	return func(c context.Context, ctx *app.RequestContext) {
		origin := string(ctx.GetHeader("Origin"))

		// A request without an Origin header is not cross-origin
		if origin == "" {
			ctx.Next(c)
			return
		}

		// Set allowed origins
		allowOrigin := cfg.AllowOrigins
		if allowOrigin == "*" && cfg.AllowCredentials {
			// Credentialed requests cannot use * and must return the actual origin
			allowOrigin = origin
		}
		ctx.Header("Access-Control-Allow-Origin", allowOrigin)

		// Allow credentials
		if cfg.AllowCredentials {
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		// Allowed methods
		ctx.Header("Access-Control-Allow-Methods", cfg.AllowMethods)

		// Allowed request headers
		if reqHeaders := string(ctx.GetHeader("Access-Control-Request-Headers")); reqHeaders != "" {
			// Merge configured and requested headers
			allHeaders := mergeHeaders(cfg.AllowHeaders, reqHeaders)
			ctx.Header("Access-Control-Allow-Headers", allHeaders)
		} else {
			ctx.Header("Access-Control-Allow-Headers", cfg.AllowHeaders)
		}

		// Exposed response headers
		if cfg.ExposeHeaders != "" {
			ctx.Header("Access-Control-Expose-Headers", cfg.ExposeHeaders)
		}

		// Preflight request cache time
		if cfg.MaxAge > 0 {
			ctx.Header("Access-Control-Max-Age", intToString(cfg.MaxAge))
		}

		// Handle preflight requests
		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(consts.StatusNoContent)
			ctx.Abort()
			return
		}

		ctx.Next(c)
	}
}

// mergeHeaders merges header strings.
func mergeHeaders(a, b string) string {
	set := make(map[string]bool)

	for _, h := range strings.Split(a, ",") {
		h = strings.TrimSpace(h)
		if h != "" {
			set[strings.ToLower(h)] = true
		}
	}

	var result []string
	for _, h := range strings.Split(a, ",") {
		h = strings.TrimSpace(h)
		if h != "" {
			result = append(result, h)
		}
	}

	for _, h := range strings.Split(b, ",") {
		h = strings.TrimSpace(h)
		if h != "" && !set[strings.ToLower(h)] {
			result = append(result, h)
			set[strings.ToLower(h)] = true
		}
	}

	return strings.Join(result, ", ")
}

// intToString performs a simple int-to-string conversion.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
