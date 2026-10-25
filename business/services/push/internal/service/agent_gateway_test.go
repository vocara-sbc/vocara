package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"vocara/services/push/internal/model"
)

// fakeAgentCenter 记录调用并按需失败。
type fakeAgentCenter struct {
	estabInfo  *AgentSessionInfo
	estabErr   error
	refreshErr error
	refreshN   int
	updateN    int
	deleteN    int
	lastStatus int
	lastReason string
}

func (f *fakeAgentCenter) EstablishSession(ctx context.Context, tenantID, agentID, dn string) (*AgentSessionInfo, error) {
	return f.estabInfo, f.estabErr
}
func (f *fakeAgentCenter) RefreshSession(ctx context.Context, tenantID, agentID string) error {
	f.refreshN++
	return f.refreshErr
}
func (f *fakeAgentCenter) DeleteSession(ctx context.Context, tenantID, agentID string) error {
	f.deleteN++
	return nil
}
func (f *fakeAgentCenter) UpdateStatus(ctx context.Context, tenantID, agentID string, status int, reason string) error {
	f.updateN++
	f.lastStatus = status
	f.lastReason = reason
	return nil
}

// newTestConn 构造测试连接（wsConn 为 nil；Send 只写缓冲 channel，对 nil 连接安全）。
func newTestConn(t *testing.T, tenantID, userID int64) *model.Connection {
	t.Helper()
	return model.NewConnection(nil, "conn_test", model.UserTypeAgent, userID, tenantID, "tester")
}

// TestOnConnectEstablishFailure 签入失败不启动保活。
func TestOnConnectEstablishFailure(t *testing.T) {
	fake := &fakeAgentCenter{estabErr: errors.New("dn not found")}
	g := NewAgentGateway(fake, NewConnectionManager())
	conn := newTestConn(t, 1, 1001)
	_, err := g.OnConnect(context.Background(), conn, "1001")
	if err == nil {
		t.Fatal("expect establish error")
	}
}

// TestKeepaliveRefresh 保活协程按间隔续期。
func TestKeepaliveRefresh(t *testing.T) {
	fake := &fakeAgentCenter{estabInfo: &AgentSessionInfo{Status: 2, StatusName: "CHECKED"}}
	g := NewAgentGateway(fake, NewConnectionManager())
	g.keepalive = 50 * time.Millisecond // 测试加速
	conn := newTestConn(t, 1, 1001)
	if _, err := g.OnConnect(context.Background(), conn, "1001"); err != nil {
		t.Fatalf("on connect: %v", err)
	}
	time.Sleep(180 * time.Millisecond)
	g.OnDisconnect(conn)
	if fake.refreshN < 2 {
		t.Fatalf("refreshN = %d, want >= 2", fake.refreshN)
	}
}

// TestHandleReadyBusy 状态命令透传数值与 reason。
func TestHandleReadyBusy(t *testing.T) {
	fake := &fakeAgentCenter{}
	g := NewAgentGateway(fake, NewConnectionManager())
	conn := newTestConn(t, 1, 1001)
	g.HandleReady(conn)
	g.HandleBusy(conn, "rest")
	if fake.lastStatus != agentStatusBusy || fake.lastReason != "rest" {
		t.Fatalf("lastStatus=%d reason=%q", fake.lastStatus, fake.lastReason)
	}
	if fake.updateN != 2 {
		t.Fatalf("updateN = %d, want 2", fake.updateN)
	}
}

// TestOnConnectKickOldAfterEstablish 建会话成功后踢掉同工号旧连接，新连接保持在线。
func TestOnConnectKickOldAfterEstablish(t *testing.T) {
	fake := &fakeAgentCenter{estabInfo: &AgentSessionInfo{Status: 2, StatusName: "CHECKED"}}
	cm := NewConnectionManager()
	g := NewAgentGateway(fake, cm)
	old := model.NewConnection(nil, "conn_old", model.UserTypeAgent, 1001, 1, "old")
	cm.AddConnection(old)
	newConn := model.NewConnection(nil, "conn_new", model.UserTypeAgent, 1001, 1, "new")
	cm.AddConnection(newConn)

	if _, err := g.OnConnect(context.Background(), newConn, "1001"); err != nil {
		t.Fatalf("on connect: %v", err)
	}
	if old.IsConnected() {
		t.Fatal("old connection should be kicked (closed)")
	}
	if !newConn.IsConnected() {
		t.Fatal("new connection should stay connected")
	}
	g.OnDisconnect(newConn)
}

// TestHandleSignout 签出成功：清会话 + ack 后关闭连接（sendAndClose 路径，nil wsConn 不 panic）。
func TestHandleSignout(t *testing.T) {
	fake := &fakeAgentCenter{}
	g := NewAgentGateway(fake, NewConnectionManager())
	conn := newTestConn(t, 1, 1001)
	g.HandleSignout(conn)
	if fake.deleteN != 1 {
		t.Fatalf("deleteN = %d, want 1", fake.deleteN)
	}
	if conn.IsConnected() {
		t.Fatal("connection should be closed after signout")
	}
}
