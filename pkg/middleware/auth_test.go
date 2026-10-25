package middleware

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// TestGenerateTokenWithTenantID 生成的 token 应携带 tenant_id 与 agent_num。
func TestGenerateTokenWithTenantID(t *testing.T) {
	token, err := GenerateTokenWithAgentNum("3", "张三", "agent", "1001", "1", "test-secret")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("parse token: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if got := claims["tenant_id"]; got != "1" {
		t.Fatalf("tenant_id = %v, want %q", got, "1")
	}
	if got := claims["agent_num"]; got != "1001" {
		t.Fatalf("agent_num = %v, want %q", got, "1001")
	}
}
