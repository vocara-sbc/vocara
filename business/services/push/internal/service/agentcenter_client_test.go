package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newFakeAgentCenter 返回一个校验 token 并按路径应答的假座席中心。
func newFakeAgentCenter(t *testing.T, wantToken string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Token") != wantToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/internal/agent/session":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0, "message": "success",
				"data": map[string]interface{}{"status": 2, "status_name": "CHECKED", "dn_number": "1001", "skills": []string{"skill1"}},
			})
		case "/api/v1/internal/agent/session/refresh", "/api/v1/internal/agent/session/delete", "/api/v1/internal/agent/update-status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "message": "success"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestEstablishSession(t *testing.T) {
	srv := newFakeAgentCenter(t, "tok")
	defer srv.Close()
	c := NewAgentCenterClient(srv.URL, "tok")
	info, err := c.EstablishSession(context.Background(), "1", "1001", "1001")
	if err != nil {
		t.Fatalf("establish: %v", err)
	}
	if info.StatusName != "CHECKED" || info.DnNumber != "1001" {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestRefreshDeleteUpdateStatus(t *testing.T) {
	srv := newFakeAgentCenter(t, "tok")
	defer srv.Close()
	c := NewAgentCenterClient(srv.URL, "tok")
	ctx := context.Background()
	if err := c.RefreshSession(ctx, "1", "1001"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := c.DeleteSession(ctx, "1", "1001"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// status=9 对应 agentcenter model.StatusBusy
	if err := c.UpdateStatus(ctx, "1", "1001", 9, "rest"); err != nil {
		t.Fatalf("update status: %v", err)
	}
}

func TestWrongTokenRejected(t *testing.T) {
	srv := newFakeAgentCenter(t, "tok")
	defer srv.Close()
	c := NewAgentCenterClient(srv.URL, "bad")
	if _, err := c.EstablishSession(context.Background(), "1", "1001", "1001"); err == nil {
		t.Fatal("expect error for wrong token")
	}
}
