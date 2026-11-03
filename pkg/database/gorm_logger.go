// gorm_logger.go routes GORM logs through the unified zap logger.
// This prevents GORM's default logger from writing directly to stdout and creating a second set of log files through startup script output redirection.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"vocara/pkg/logger"
)

// slowSQLThreshold is the slow SQL threshold; slower statements are logged at Warn level.
const slowSQLThreshold = 200 * time.Millisecond

// zapGormLogger implements gormlogger.Interface and forwards GORM logs to the global zap logger.
type zapGormLogger struct {
	level gormlogger.LogLevel
}

// newZapGormLogger creates a GORM logger integrated with zap.
func newZapGormLogger(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{level: level}
}

// LogMode returns a new instance at the given level, as required by GORM.
func (l *zapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{level: level}
}

// Info logs a GORM message at Info level.
func (l *zapGormLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Info {
		logger.Info(fmt.Sprintf(msg, args...), zap.String("source", "gorm"))
	}
}

// Warn logs a GORM message at Warn level.
func (l *zapGormLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Warn {
		logger.Warn(fmt.Sprintf(msg, args...), zap.String("source", "gorm"))
	}
}

// Error logs a GORM message at Error level.
func (l *zapGormLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Error {
		logger.Error(fmt.Sprintf(msg, args...), zap.String("source", "gorm"))
	}
}

// Trace logs SQL execution details, prioritizing errors, then slow queries, and finally all SQL according to the log level.
func (l *zapGormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.String("source", "gorm"),
		zap.String("gorm_caller", utils.FileWithLineNum()),
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	switch {
	case err != nil && l.level >= gormlogger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		logger.Error("gorm sql error", append(fields, zap.Error(err))...)
	case elapsed > slowSQLThreshold && l.level >= gormlogger.Warn:
		logger.Warn("gorm slow sql", fields...)
	case l.level >= gormlogger.Info:
		logger.Info("gorm sql", fields...)
	}
}