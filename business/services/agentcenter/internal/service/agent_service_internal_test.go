package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"vocara/services/agentcenter/internal/model"
)

// newTestAgentService 构造仅依赖 Redis 路径的 AgentService（repo 传 nil，测试用例不得触及 DB 路径）。
func newTestAgentService(t *testing.T) (*AgentService, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	sessionSvc := NewSessionService(rdb, 120)
	svc := NewAgentService(nil, nil, nil, nil, sessionSvc, "test-secret", "test", 24, nil, nil)
	return svc, mr
}

// TestInternalEstablishSession_RecoverExisting 会话存在时幂等恢复：状态保持、TTL 被续期。
func TestInternalEstablishSession_RecoverExisting(t *testing.T) {
	svc, mr := newTestAgentService(t)
	ctx := context.Background()

	existing := &model.AgentSession{
		AgentID:   "1001",
		AgentName: "测试座席",
		TenantID:  "1",
		Status:    int(model.StatusReady),
		DnNumber:  "1001",
	}
	if err := svc.sessionSvc.SetSession(ctx, existing); err != nil {
		t.Fatalf("set session: %v", err)
	}
	mr.FastForward(30 * time.Second)

	got, err := svc.InternalEstablishSession(ctx, "1", "1001", "1001")
	if err != nil {
		t.Fatalf("establish: %v", err)
	}
	if got.GetStatus() != model.StatusReady {
		t.Fatalf("status = %v, want READY（幂等恢复不得重置状态）", got.GetStatus())
	}
	// 续期生效：TTL 应接近 120s 而非 90s
	ttl := mr.TTL(svc.sessionSvc.SessionKey("1", "1001"))
	if ttl < 100*time.Second {
		t.Fatalf("ttl = %v, want >= 100s（应被续期）", ttl)
	}
}

// TestInternalRefreshSession_NotFound 会话不存在时续期应返回会话过期错误。
func TestInternalRefreshSession_NotFound(t *testing.T) {
	svc, _ := newTestAgentService(t)
	err := svc.InternalRefreshSession(context.Background(), "1", "9999")
	if err == nil {
		t.Fatal("expect error for missing session")
	}
}

// TestInternalDeleteSession_Idempotent 无会话时签出幂等成功。
func TestInternalDeleteSession_Idempotent(t *testing.T) {
	svc, _ := newTestAgentService(t)
	if err := svc.InternalDeleteSession(context.Background(), "1", "9999"); err != nil {
		t.Fatalf("delete should be idempotent: %v", err)
	}
}
