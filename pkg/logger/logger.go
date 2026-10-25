// Package logger 提供基于 Zap 的日志封装
// 支持不同日志级别、结构化日志、控制台+文件输出、上下文透传
package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"vocara/pkg/config"
)

// 上下文 key 类型
type ctxKey string

const (
	// TraceIDKey trace_id 在上下文中的 key
	TraceIDKey ctxKey = "trace_id"
	// UserIDKey user_id 在上下文中的 key
	UserIDKey ctxKey = "user_id"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

// Init 初始化全局日志器
// 传入 LogConfig 配置
func Init(cfg config.LogConfig) error {
	var initErr error
	once.Do(func() {
		level := parseLevel(cfg.Level)

		// 设置编码器
		var encoder zapcore.Encoder
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoderConfig.TimeKey = "time"
		encoderConfig.LevelKey = "level"
		encoderConfig.MessageKey = "msg"
		encoderConfig.CallerKey = "caller"

		if cfg.Format == "console" {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		} else {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		}

		// 文件输出使用无颜色的 encoder，避免 ANSI 颜色转义码写入日志文件
		fileEncoder := encoder
		if cfg.Format == "console" {
			fileEncoderConfig := encoderConfig
			fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
			fileEncoder = zapcore.NewConsoleEncoder(fileEncoderConfig)
		}

		// 设置输出
		var cores []zapcore.Core

		switch cfg.Output {
		case "stdout":
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))
		case "file":
			cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(newFileWriter(cfg)), level))
		case "both":
			fallthrough
		default:
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))
			cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(newFileWriter(cfg)), level))
		}

		core := zapcore.NewTee(cores...)

		// 构建 logger
		opts := []zap.Option{zap.AddCallerSkip(cfg.CallerSkip)}
		if cfg.Caller {
			opts = append(opts, zap.AddCaller())
		}
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))

		globalLogger = zap.New(core, opts...)
	})

	return initErr
}

// parseLevel 解析日志级别字符串
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// newFileWriter 创建文件写入器（带日志切割）
func newFileWriter(cfg config.LogConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,    // MB
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,     // days
		Compress:   cfg.Compress,
	}
}

// getLogger 获取全局 logger，如果未初始化则返回默认 logger
func getLogger() *zap.Logger {
	if globalLogger == nil {
		// 使用默认配置初始化
		cfg := config.LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			FilePath:   "./logs/app.log",
			MaxSize:    100,
			MaxAge:     30,
			MaxBackups: 10,
			Compress:   true,
			Caller:     true,
			CallerSkip: 1,
		}
		_ = Init(cfg)
	}
	return globalLogger
}

// WithContext 从上下文中提取信息并返回带字段的 logger
func WithContext(ctx context.Context) *zap.Logger {
	l := getLogger()
	if ctx == nil {
		return l
	}

	var fields []zap.Field

	// 提取 trace_id
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// 提取 user_id
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	if len(fields) > 0 {
		return l.With(fields...)
	}

	return l
}

// WithField 返回带单个字段的 logger
func WithField(key string, val interface{}) *zap.Logger {
	l := getLogger()
	return l.With(zap.Any(key, val))
}

// WithFields 返回带多个字段的 logger
func WithFields(fields map[string]interface{}) *zap.Logger {
	l := getLogger()
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return l.With(zapFields...)
}

// Sugar 获取 sugared logger
func Sugar() *zap.SugaredLogger {
	return getLogger().Sugar()
}

// Logger 获取原始 zap.Logger
func Logger() *zap.Logger {
	return getLogger()
}

// Sync 刷新日志缓冲区
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// ===== 全局日志函数 =====

// Debug 输出 Debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	getLogger().Debug(msg, fields...)
}

// Info 输出 Info 级别日志
func Info(msg string, fields ...zap.Field) {
	getLogger().Info(msg, fields...)
}

// Warn 输出 Warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	getLogger().Warn(msg, fields...)
}

// Error 输出 Error 级别日志
func Error(msg string, fields ...zap.Field) {
	getLogger().Error(msg, fields...)
}

// DPanic 输出 DPanic 级别日志（开发模式下 panic）
func DPanic(msg string, fields ...zap.Field) {
	getLogger().DPanic(msg, fields...)
}

// Panic 输出 Panic 级别日志（会触发 panic）
func Panic(msg string, fields ...zap.Field) {
	getLogger().Panic(msg, fields...)
}

// Fatal 输出 Fatal 级别日志（会触发 os.Exit(1)）
func Fatal(msg string, fields ...zap.Field) {
	getLogger().Fatal(msg, fields...)
}

// ===== 格式化日志函数 =====

// Debugf 输出格式化的 Debug 级别日志
func Debugf(format string, args ...interface{}) {
	getLogger().Debug(fmt.Sprintf(format, args...))
}

// Infof 输出格式化的 Info 级别日志
func Infof(format string, args ...interface{}) {
	getLogger().Info(fmt.Sprintf(format, args...))
}

// Warnf 输出格式化的 Warn 级别日志
func Warnf(format string, args ...interface{}) {
	getLogger().Warn(fmt.Sprintf(format, args...))
}

// Errorf 输出格式化的 Error 级别日志
func Errorf(format string, args ...interface{}) {
	getLogger().Error(fmt.Sprintf(format, args...))
}

// DPanicf 输出格式化的 DPanic 级别日志
func DPanicf(format string, args ...interface{}) {
	getLogger().DPanic(fmt.Sprintf(format, args...))
}

// Panicf 输出格式化的 Panic 级别日志
func Panicf(format string, args ...interface{}) {
	getLogger().Panic(fmt.Sprintf(format, args...))
}

// Fatalf 输出格式化的 Fatal 级别日志
func Fatalf(format string, args ...interface{}) {
	getLogger().Fatal(fmt.Sprintf(format, args...))
}

// ===== 上下文日志函数 =====

// DebugCtx 从上下文中提取字段并输出 Debug 级别日志
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

// InfoCtx 从上下文中提取字段并输出 Info 级别日志
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

// WarnCtx 从上下文中提取字段并输出 Warn 级别日志
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

// ErrorCtx 从上下文中提取字段并输出 Error 级别日志
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Error(msg, fields...)
}
