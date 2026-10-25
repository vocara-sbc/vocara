// gorm_logger.go 将 GORM 日志接入统一 zap 日志
// 避免 GORM 默认 logger 直写 stdout，导致与启动脚本的输出重定向产生第二套日志文件
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

// slowSQLThreshold 慢 SQL 阈值，超过则以 Warn 级别输出
const slowSQLThreshold = 200 * time.Millisecond

// zapGormLogger 实现 gormlogger.Interface，把 GORM 日志转发到 zap 全局 logger
type zapGormLogger struct {
	level gormlogger.LogLevel
}

// newZapGormLogger 创建接入 zap 的 GORM logger
func newZapGormLogger(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{level: level}
}

// LogMode 按给定级别返回新实例（GORM 约定）
func (l *zapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{level: level}
}

// Info 输出 GORM Info 级别日志
func (l *zapGormLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Info {
		logger.Info(fmt.Sprintf(msg, args...), zap.String("source", "gorm"))
	}
}

// Warn 输出 GORM Warn 级别日志
func (l *zapGormLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Warn {
		logger.Warn(fmt.Sprintf(msg, args...), zap.String("source", "gorm"))
	}
}

// Error 输出 GORM Error 级别日志
func (l *zapGormLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Error {
		logger.Error(fmt.Sprintf(msg, args...), zap.String("source", "gorm"))
	}
}

// Trace 输出 SQL 执行明细：错误优先，其次慢查询，最后按级别输出全量 SQL
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