package logger

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"vocara/pkg/config"
)

func TestInit(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	cfg := config.LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "both",
		FilePath:   logPath,
		MaxSize:    10,
		MaxAge:     1,
		MaxBackups: 3,
		Compress:   false,
		Caller:     true,
		CallerSkip: 1,
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if globalLogger == nil {
		t.Fatal("globalLogger should not be nil after Init")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"dpanic", zapcore.DPanicLevel},
		{"panic", zapcore.PanicLevel},
		{"fatal", zapcore.FatalLevel},
		{"unknown", zapcore.InfoLevel},
		{"", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := config.LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "stdout",
		Caller:     false,
		CallerSkip: 1,
	}
	_ = Init(cfg)

	// Test a nil context
	//lint:ignore SA1012 This test specifically validates fallback handling for a nil context
	l := WithContext(nil)
	if l == nil {
		t.Error("WithContext(nil) should not return nil")
	}

	// Test a context with trace_id
	ctx := context.WithValue(context.Background(), TraceIDKey, "test-trace-123")
	l = WithContext(ctx)
	if l == nil {
		t.Error("WithContext with trace_id should not return nil")
	}

	// Test a context with user_id
	ctx = context.WithValue(context.Background(), UserIDKey, "user-456")
	l = WithContext(ctx)
	if l == nil {
		t.Error("WithContext with user_id should not return nil")
	}

	// Test a context with both trace_id and user_id
	ctx = context.WithValue(context.Background(), TraceIDKey, "trace-789")
	ctx = context.WithValue(ctx, UserIDKey, "user-789")
	l = WithContext(ctx)
	if l == nil {
		t.Error("WithContext with both fields should not return nil")
	}
}

func TestWithField(t *testing.T) {
	globalLogger = nil
	once = sync.Once{}

	cfg := config.LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "stdout",
		Caller:     false,
		CallerSkip: 1,
	}
	_ = Init(cfg)

	l := WithField("key", "value")
	if l == nil {
		t.Error("WithField should not return nil")
	}
}

func TestWithFields(t *testing.T) {
	globalLogger = nil
	once = sync.Once{}

	cfg := config.LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "stdout",
		Caller:     false,
		CallerSkip: 1,
	}
	_ = Init(cfg)

	l := WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	})
	if l == nil {
		t.Error("WithFields should not return nil")
	}
}

func TestGlobalLogFunctions(t *testing.T) {
	globalLogger = nil
	once = sync.Once{}

	// Temporarily replace os.Stdout to avoid test output
	cfg := config.LogConfig{
		Level:      "debug",
		Format:     "console",
		Output:     "stdout",
		Caller:     false,
		CallerSkip: 1,
	}
	_ = Init(cfg)

	// A simple smoke test only needs to verify that no panic occurs
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	Debugf("debug %s", "formatted")
	Infof("info %s", "formatted")
	Warnf("warn %s", "formatted")
	Errorf("error %s", "formatted")

	// With fields
	Info("with fields", zap.String("foo", "bar"), zap.Int("num", 42))
}

func TestContextLogFunctions(t *testing.T) {
	globalLogger = nil
	once = sync.Once{}

	cfg := config.LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "stdout",
		Caller:     false,
		CallerSkip: 1,
	}
	_ = Init(cfg)

	ctx := context.WithValue(context.Background(), TraceIDKey, "ctx-trace-123")
	ctx = context.WithValue(ctx, UserIDKey, "ctx-user-456")

	// The test only needs to verify that no panic occurs
	DebugCtx(ctx, "debug ctx message")
	InfoCtx(ctx, "info ctx message")
	WarnCtx(ctx, "warn ctx message")
	ErrorCtx(ctx, "error ctx message")
}

func TestSugarAndLogger(t *testing.T) {
	globalLogger = nil
	once = sync.Once{}

	cfg := config.LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		Caller:     false,
		CallerSkip: 1,
	}
	_ = Init(cfg)

	if Logger() == nil {
		t.Error("Logger() should not return nil")
	}
	if Sugar() == nil {
		t.Error("Sugar() should not return nil")
	}
}

func TestNewFileWriter(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	cfg := config.LogConfig{
		FilePath:   logPath,
		MaxSize:    10,
		MaxAge:     1,
		MaxBackups: 3,
		Compress:   false,
	}

	w := newFileWriter(cfg)
	if w == nil {
		t.Fatal("newFileWriter should not return nil")
	}
	// On Windows, leaving file handles open causes TempDir cleanup to fail
	if c, ok := w.(io.Closer); ok {
		t.Cleanup(func() { _ = c.Close() })
	}

	// Test writing
	data := []byte("test log message\n")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}

	// Verify that the file exists and has the correct content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("file content = %q, want %q", string(content), string(data))
	}
}

// TestConsoleFormatFileWithoutANSI verifies that console format output written to a file contains no ANSI color codes.
// Colors belong only in stdout; writing them to a file produces unreadable escape characters.
func TestConsoleFormatFileWithoutANSI(t *testing.T) {
	globalLogger = nil
	once = sync.Once{}

	// Do not use t.TempDir: lumberjack holds handles on Windows, causing TempDir cleanup to report that files are in use
	dir, err := os.MkdirTemp("", "logger_console_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	logPath := filepath.Join(dir, "console.log")

	cfg := config.LogConfig{
		Level:      "debug",
		Format:     "console",
		Output:     "file",
		FilePath:   logPath,
		MaxSize:    10,
		MaxAge:     1,
		MaxBackups: 3,
		CallerSkip: 1,
	}

	if err := Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	Info("ansi check message", zap.String("key", "value"))
	if err := Sync(); err != nil {
		t.Logf("Sync: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "ansi check message") {
		t.Error("log message should be written to file")
	}
	if strings.Contains(string(content), "\x1b[") {
		t.Error("file output should not contain ANSI escape codes")
	}
}
