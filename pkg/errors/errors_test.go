package errors

import (
	stderrors "errors"
	"testing"
)

func TestNewBizError(t *testing.T) {
	err := NewBizError(40001, "参数错误")
	if err == nil {
		t.Fatal("NewBizError should not return nil")
	}
	if err.Code != 40001 {
		t.Errorf("expected code 40001, got %d", err.Code)
	}
	if err.Message != "参数错误" {
		t.Errorf("expected message '参数错误', got '%s'", err.Message)
	}
	if err.HTTPStatus != 400 {
		t.Errorf("expected HTTP status 400, got %d", err.HTTPStatus)
	}
}

func TestNewBizErrorWithStatus(t *testing.T) {
	err := NewBizErrorWithStatus(10001, "自定义错误", 200)
	if err.HTTPStatus != 200 {
		t.Errorf("expected HTTP status 200, got %d", err.HTTPStatus)
	}
}

func TestBizError_Error(t *testing.T) {
	// 没有 detail
	err := NewBizError(40000, "参数错误")
	msg := err.Error()
	if msg != "[40000] 参数错误" {
		t.Errorf("unexpected error message: %s", msg)
	}

	// 有 detail
	err.WithDetail("具体的参数问题")
	msg = err.Error()
	if msg != "[40000] 参数错误: 具体的参数问题" {
		t.Errorf("unexpected error message with detail: %s", msg)
	}
}

func TestBizError_WithDetail(t *testing.T) {
	err := NewBizError(50000, "系统错误")
	result := err.WithDetail("database connection failed")
	if result != err {
		t.Error("WithDetail should return the same error")
	}
	if err.Detail != "database connection failed" {
		t.Errorf("unexpected detail: %s", err.Detail)
	}
}

func TestBizError_WithDetailf(t *testing.T) {
	err := NewBizError(50000, "系统错误")
	err.WithDetailf("connection failed: %s", "timeout")
	if err.Detail != "connection failed: timeout" {
		t.Errorf("unexpected detail: %s", err.Detail)
	}
}

func TestBizError_WithCause(t *testing.T) {
	original := stderrors.New("original error")
	err := NewBizError(50000, "系统错误").WithCause(original)

	if err.cause != original {
		t.Error("cause should be set")
	}
	if err.Detail != "original error" {
		t.Errorf("detail should be set from cause, got: %s", err.Detail)
	}

	// Unwrap
	unwrapped := err.Unwrap()
	if unwrapped != original {
		t.Error("Unwrap should return original error")
	}
}

func TestWrap(t *testing.T) {
	// nil error
	if Wrap(nil) != nil {
		t.Error("Wrap(nil) should return nil")
	}

	// 普通错误
	err := stderrors.New("some error")
	wrapped := Wrap(err)
	if wrapped == nil {
		t.Fatal("Wrap should not return nil")
	}
	if wrapped.Code != CodeSystemError {
		t.Errorf("expected code %d, got %d", CodeSystemError, wrapped.Code)
	}
	if wrapped.Message != MsgSystemError {
		t.Errorf("expected message '%s', got '%s'", MsgSystemError, wrapped.Message)
	}
	if wrapped.Detail != "some error" {
		t.Errorf("expected detail 'some error', got '%s'", wrapped.Detail)
	}

	// 已经是 BizError
	bizErr := NewBizError(40000, "参数错误")
	wrapped2 := Wrap(bizErr)
	if wrapped2 != bizErr {
		t.Error("Wrap on BizError should return the same error")
	}
}

func TestWrapWithCode(t *testing.T) {
	if WrapWithCode(nil, 40000, "参数错误") != nil {
		t.Error("WrapWithCode(nil) should return nil")
	}

	err := stderrors.New("some error")
	wrapped := WrapWithCode(err, 40001, "参数无效")
	if wrapped.Code != 40001 {
		t.Errorf("expected code 40001, got %d", wrapped.Code)
	}
	if wrapped.Message != "参数无效" {
		t.Errorf("expected message '参数无效', got '%s'", wrapped.Message)
	}
	if wrapped.Detail != "some error" {
		t.Errorf("expected detail 'some error', got '%s'", wrapped.Detail)
	}
}

func TestIsBizError(t *testing.T) {
	// nil
	_, ok := IsBizError(nil)
	if ok {
		t.Error("IsBizError(nil) should return false")
	}

	// 普通错误
	err := stderrors.New("normal error")
	_, ok = IsBizError(err)
	if ok {
		t.Error("IsBizError on normal error should return false")
	}

	// BizError
	bizErr := NewBizError(40000, "参数错误")
	result, ok := IsBizError(bizErr)
	if !ok {
		t.Error("IsBizError on BizError should return true")
	}
	if result != bizErr {
		t.Error("IsBizError should return the same BizError")
	}
}

func TestBizError_IsCode(t *testing.T) {
	err := NewBizError(40000, "参数错误")
	if !err.IsCode(40000) {
		t.Error("IsCode should return true for matching code")
	}
	if err.IsCode(50000) {
		t.Error("IsCode should return false for non-matching code")
	}
}

func TestCodeToHTTPStatus(t *testing.T) {
	tests := []struct {
		code     int
		expected int
	}{
		{0, 200},
		{40000, 400},
		{40099, 400},
		{40100, 401},
		{40199, 401},
		{40300, 403},
		{40399, 403},
		{40400, 404},
		{40499, 404},
		{40900, 409},
		{40999, 409},
		{42900, 429},
		{42999, 429},
		{50000, 500},
		{50001, 500},
		{59999, 500},
		{70001, 402}, // License 缺失/非法/到期
		{70003, 402},
		{70004, 403}, // 指纹/配额/模块/时钟
		{70007, 403},
		{70000, 200}, // 700xx 段外回落 200
		{70008, 200},
		{10001, 200}, // 业务错误默认 200
		{20001, 200},
		{30001, 200},
	}

	for _, tt := range tests {
		t.Run("code=%d", func(t *testing.T) {
			got := codeToHTTPStatus(tt.code)
			if got != tt.expected {
				t.Errorf("codeToHTTPStatus(%d) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	// 验证几个预定义错误
	if ErrSuccess.Code != CodeSuccess {
		t.Error("ErrSuccess code mismatch")
	}
	if ErrParamError.Code != CodeParamError {
		t.Error("ErrParamError code mismatch")
	}
	if ErrUnauthorized.Code != CodeUnauthorized {
		t.Error("ErrUnauthorized code mismatch")
	}
	if ErrForbidden.Code != CodeForbidden {
		t.Error("ErrForbidden code mismatch")
	}
	if ErrNotFound.Code != CodeNotFound {
		t.Error("ErrNotFound code mismatch")
	}
	if ErrSystemError.Code != CodeSystemError {
		t.Error("ErrSystemError code mismatch")
	}
	if ErrAgentNotCheckedIn.Code != CodeAgentNotCheckedIn {
		t.Error("ErrAgentNotCheckedIn code mismatch")
	}
	if ErrAgentBusy.Code != CodeAgentBusy {
		t.Error("ErrAgentBusy code mismatch")
	}
	if ErrQueueNotFound.Code != CodeQueueNotFound {
		t.Error("ErrQueueNotFound code mismatch")
	}
	if ErrCallNotFound.Code != CodeCallNotFound {
		t.Error("ErrCallNotFound code mismatch")
	}
}
