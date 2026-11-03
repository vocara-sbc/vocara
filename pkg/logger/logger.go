// Package logger provides a Zap-based logging wrapper.
// It supports multiple log levels, structured logging, console and file output, and context propagation.
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

// Context key type
type ctxKey string

const (
	// TraceIDKey is the context key for trace_id.
	TraceIDKey ctxKey = "trace_id"
	// UserIDKey is the context key for user_id.
	UserIDKey ctxKey = "user_id"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

// Init initializes the global logger with a LogConfig.
func Init(cfg config.LogConfig) error {
	var initErr error
	once.Do(func() {
		level := parseLevel(cfg.Level)

		// Configure the encoder
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

		// Use a colorless encoder for file output to keep ANSI color escape codes out of log files
		fileEncoder := encoder
		if cfg.Format == "console" {
			fileEncoderConfig := encoderConfig
			fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
			fileEncoder = zapcore.NewConsoleEncoder(fileEncoderConfig)
		}

		// Configure outputs
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

		// Build the logger
		opts := []zap.Option{zap.AddCallerSkip(cfg.CallerSkip)}
		if cfg.Caller {
			opts = append(opts, zap.AddCaller())
		}
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))

		globalLogger = zap.New(core, opts...)
	})

	return initErr
}

// parseLevel parses a log level string.
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

// newFileWriter creates a file writer with log rotation.
func newFileWriter(cfg config.LogConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,    // MB
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,     // days
		Compress:   cfg.Compress,
	}
}

// getLogger returns the global logger, initializing a default logger if necessary.
func getLogger() *zap.Logger {
	if globalLogger == nil {
		// Initialize with the default configuration
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

// WithContext extracts information from the context and returns a logger with fields.
func WithContext(ctx context.Context) *zap.Logger {
	l := getLogger()
	if ctx == nil {
		return l
	}

	var fields []zap.Field

	// Extract trace_id
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// Extract user_id
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	if len(fields) > 0 {
		return l.With(fields...)
	}

	return l
}

// WithField returns a logger with a single field.
func WithField(key string, val interface{}) *zap.Logger {
	l := getLogger()
	return l.With(zap.Any(key, val))
}

// WithFields returns a logger with multiple fields.
func WithFields(fields map[string]interface{}) *zap.Logger {
	l := getLogger()
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return l.With(zapFields...)
}

// Sugar returns a sugared logger.
func Sugar() *zap.SugaredLogger {
	return getLogger().Sugar()
}

// Logger returns the raw zap.Logger.
func Logger() *zap.Logger {
	return getLogger()
}

// Sync flushes the log buffer.
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// ===== Global logging functions =====

// Debug logs at Debug level.
func Debug(msg string, fields ...zap.Field) {
	getLogger().Debug(msg, fields...)
}

// Info logs at Info level.
func Info(msg string, fields ...zap.Field) {
	getLogger().Info(msg, fields...)
}

// Warn logs at Warn level.
func Warn(msg string, fields ...zap.Field) {
	getLogger().Warn(msg, fields...)
}

// Error logs at Error level.
func Error(msg string, fields ...zap.Field) {
	getLogger().Error(msg, fields...)
}

// DPanic logs at DPanic level and panics in development mode.
func DPanic(msg string, fields ...zap.Field) {
	getLogger().DPanic(msg, fields...)
}

// Panic logs at Panic level and triggers a panic.
func Panic(msg string, fields ...zap.Field) {
	getLogger().Panic(msg, fields...)
}

// Fatal logs at Fatal level and triggers os.Exit(1).
func Fatal(msg string, fields ...zap.Field) {
	getLogger().Fatal(msg, fields...)
}

// ===== Formatted logging functions =====

// Debugf logs a formatted message at Debug level.
func Debugf(format string, args ...interface{}) {
	getLogger().Debug(fmt.Sprintf(format, args...))
}

// Infof logs a formatted message at Info level.
func Infof(format string, args ...interface{}) {
	getLogger().Info(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted message at Warn level.
func Warnf(format string, args ...interface{}) {
	getLogger().Warn(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted message at Error level.
func Errorf(format string, args ...interface{}) {
	getLogger().Error(fmt.Sprintf(format, args...))
}

// DPanicf logs a formatted message at DPanic level.
func DPanicf(format string, args ...interface{}) {
	getLogger().DPanic(fmt.Sprintf(format, args...))
}

// Panicf logs a formatted message at Panic level.
func Panicf(format string, args ...interface{}) {
	getLogger().Panic(fmt.Sprintf(format, args...))
}

// Fatalf logs a formatted message at Fatal level.
func Fatalf(format string, args ...interface{}) {
	getLogger().Fatal(fmt.Sprintf(format, args...))
}

// ===== Context logging functions =====

// DebugCtx extracts fields from the context and logs at Debug level.
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

// InfoCtx extracts fields from the context and logs at Info level.
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

// WarnCtx extracts fields from the context and logs at Warn level.
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

// ErrorCtx extracts fields from the context and logs at Error level.
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Error(msg, fields...)
}
