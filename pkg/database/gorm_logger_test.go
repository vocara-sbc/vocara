package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"vocara/pkg/config"
	"vocara/pkg/logger"
)

// TestZapGormLogger 验证 GORM 日志被转发到统一 zap 日志文件
// 覆盖：慢 SQL（Warn）、SQL 错误（Error）、级别过滤、ErrRecordNotFound 忽略
func TestZapGormLogger(t *testing.T) {
	// 不用 t.TempDir：Windows 上 lumberjack 持有日志文件句柄，TempDir 清理会报占用错误
	tmpDir, err := os.MkdirTemp("", "gorm_logger_test")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	logFile := filepath.Join(tmpDir, "gorm.log")
	if err := logger.Init(config.LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "file",
		FilePath:   logFile,
		MaxSize:    10,
		MaxBackups: 1,
		MaxAge:     1,
		CallerSkip: 1,
	}); err != nil {
		t.Fatalf("init logger failed: %v", err)
	}

	gl := newZapGormLogger(gormlogger.Warn)
	ctx := context.Background()

	// 慢 SQL（>200ms）应输出 Warn
	gl.Trace(ctx, time.Now().Add(-300*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM slow_table", 1
	}, nil)

	// 普通 SQL 在 Warn 级别下应被过滤
	gl.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM fast_table", 1
	}, nil)

	// SQL 错误应输出 Error
	gl.Trace(ctx, time.Now(), func() (string, int64) {
		return "UPDATE t SET a=1", 0
	}, errors.New("connection refused"))

	// ErrRecordNotFound 应被忽略
	gl.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM t WHERE id=1", 0
	}, gorm.ErrRecordNotFound)

	// Silent 级别什么都不输出
	silent := gl.LogMode(gormlogger.Silent)
	silent.Trace(ctx, time.Now().Add(-time.Second), func() (string, int64) {
		return "SELECT silent", 1
	}, errors.New("should not appear"))

	if err := logger.Sync(); err != nil {
		t.Logf("sync logger: %v", err)
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	s := string(content)

	if !strings.Contains(s, "gorm slow sql") {
		t.Error("expected slow sql warn log")
	}
	if !strings.Contains(s, "gorm sql error") {
		t.Error("expected sql error log")
	}
	if strings.Contains(s, "fast_table") {
		t.Error("normal sql should be filtered at warn level")
	}
	if strings.Contains(s, "WHERE id=1") {
		t.Error("ErrRecordNotFound should be ignored")
	}
	if strings.Contains(s, "SELECT silent") {
		t.Error("silent level should not output anything")
	}
}

// TestZapGormLoggerLogMode 验证 LogMode 返回正确级别的新实例
func TestZapGormLoggerLogMode(t *testing.T) {
	gl := newZapGormLogger(gormlogger.Warn)
	if l, ok := gl.LogMode(gormlogger.Error).(*zapGormLogger); !ok || l.level != gormlogger.Error {
		t.Error("LogMode should return new instance with given level")
	}
	if l := gl.(*zapGormLogger); l.level != gormlogger.Warn {
		t.Error("LogMode should not mutate original instance")
	}
}
