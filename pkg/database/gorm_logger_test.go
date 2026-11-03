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

// TestZapGormLogger verifies that GORM logs are forwarded to the unified zap log file.
// It covers slow SQL (Warn), SQL errors (Error), level filtering, and ignoring ErrRecordNotFound.
func TestZapGormLogger(t *testing.T) {
	// Do not use t.TempDir: lumberjack holds the log file handle on Windows, causing TempDir cleanup to fail because the file is in use
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

	// Slow SQL (>200ms) should be logged at Warn level
	gl.Trace(ctx, time.Now().Add(-300*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM slow_table", 1
	}, nil)

	// Normal SQL should be filtered at Warn level
	gl.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM fast_table", 1
	}, nil)

	// SQL errors should be logged at Error level
	gl.Trace(ctx, time.Now(), func() (string, int64) {
		return "UPDATE t SET a=1", 0
	}, errors.New("connection refused"))

	// ErrRecordNotFound should be ignored
	gl.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM t WHERE id=1", 0
	}, gorm.ErrRecordNotFound)

	// Silent level should not output anything
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

// TestZapGormLoggerLogMode verifies that LogMode returns a new instance with the correct level.
func TestZapGormLoggerLogMode(t *testing.T) {
	gl := newZapGormLogger(gormlogger.Warn)
	if l, ok := gl.LogMode(gormlogger.Error).(*zapGormLogger); !ok || l.level != gormlogger.Error {
		t.Error("LogMode should return new instance with given level")
	}
	if l := gl.(*zapGormLogger); l.level != gormlogger.Warn {
		t.Error("LogMode should not mutate original instance")
	}
}
