package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// CORSConfig CORS 中间件配置
type CORSConfig struct {
	AllowOrigins     string // 允许的源，* 表示所有
	AllowMethods     string // 允许的方法
	AllowHeaders     string // 允许的请求头
	AllowCredentials bool   // 是否允许凭证
	ExposeHeaders    string // 暴露的响应头
	MaxAge           int    // 预检请求缓存时间（秒）
}

// DefaultCORSConfig 默认 CORS 配置
var DefaultCORSConfig = CORSConfig{
	AllowOrigins: "*",
	AllowMethods: "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD",
	AllowHeaders: "Content-Type, Authorization, X-Requested-With, X-Trace-ID, X-Internal-Token",
	MaxAge:       86400, // 24小时
}

// CORS 跨域中间件
func CORS() app.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig)
}

// CORSWithConfig 使用配置创建 CORS 中间件
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

		// 没有 Origin 头，不是跨域请求
		if origin == "" {
			ctx.Next(c)
			return
		}

		// 设置允许的源
		allowOrigin := cfg.AllowOrigins
		if allowOrigin == "*" && cfg.AllowCredentials {
			// 带凭证的请求不能用 *，需要返回实际的 origin
			allowOrigin = origin
		}
		ctx.Header("Access-Control-Allow-Origin", allowOrigin)

		// 允许凭证
		if cfg.AllowCredentials {
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		// 允许的方法
		ctx.Header("Access-Control-Allow-Methods", cfg.AllowMethods)

		// 允许的请求头
		if reqHeaders := string(ctx.GetHeader("Access-Control-Request-Headers")); reqHeaders != "" {
			// 合并配置的和请求的
			allHeaders := mergeHeaders(cfg.AllowHeaders, reqHeaders)
			ctx.Header("Access-Control-Allow-Headers", allHeaders)
		} else {
			ctx.Header("Access-Control-Allow-Headers", cfg.AllowHeaders)
		}

		// 暴露的响应头
		if cfg.ExposeHeaders != "" {
			ctx.Header("Access-Control-Expose-Headers", cfg.ExposeHeaders)
		}

		// 预检请求缓存时间
		if cfg.MaxAge > 0 {
			ctx.Header("Access-Control-Max-Age", intToString(cfg.MaxAge))
		}

		// 处理预检请求
		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(consts.StatusNoContent)
			ctx.Abort()
			return
		}

		ctx.Next(c)
	}
}

// mergeHeaders 合并 header 字符串
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

// intToString 简单的 int 转 string
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
