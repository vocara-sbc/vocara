package response

import (
	"testing"

	"vocara/pkg/errors"
)

func TestResponse(t *testing.T) {
	// 测试 Response 结构体
	resp := Response{
		Code:    0,
		Message: "success",
		Data:    map[string]string{"key": "value"},
	}

	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("expected message 'success', got '%s'", resp.Message)
	}
}

func TestPageData(t *testing.T) {
	data := PageData{
		List:     []string{"a", "b"},
		Total:    100,
		Page:     1,
		PageSize: 10,
		Pages:    10,
	}

	if data.Total != 100 {
		t.Errorf("expected total 100, got %d", data.Total)
	}
	if data.Page != 1 {
		t.Errorf("expected page 1, got %d", data.Page)
	}
}

func TestBizErrorIntegration(t *testing.T) {
	// 测试与 errors 包的集成
	err := errors.NewBizError(errors.CodeParamError, errors.MsgParamError)
	if err.Code != errors.CodeParamError {
		t.Errorf("expected code %d, got %d", errors.CodeParamError, err.Code)
	}

	// 测试包装
	wrapped := errors.Wrap(err)
	if wrapped.Code != errors.CodeParamError {
		t.Errorf("wrapped should preserve BizError, got code %d", wrapped.Code)
	}
}
