package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"vocara/services/push/internal/model"
)

// fakeAgentCenter records calls and fails on demand.
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

// newTestConn constructs a test connection with a nil wsConn; Send only writes to the buffered channel and is safe with a nil connection.
func newTestConn(t *testing.T, tenantID, userID int64) *model.Connection {
	t.Helper()
	return model.NewConnection(nil, "conn_test", model.UserTypeAgent, userID, tenantID, "tester")
}

// TestOnConnectEstablishFailure verifies that keepalive does not start when sign-in fails.
func TestOnConnectEstablishFailure(t *testing.T) {
	fake := &fakeAgentCenter{estabErr: errors.New("dn not found")}
	g := NewAgentGateway(fake, NewConnectionManager())
	conn := newTestConn(t, 1, 1001)
	_, err := g.OnConnect(context.Background(), conn, "1001")
	if err == nil {
		t.Fatal("expect establish error")
	}
}

// TestKeepaliveRefresh verifies that the keepalive goroutine renews the session at each interval.
func TestKeepaliveRefresh(t *testing.T) {
	fake := &fakeAgentCenter{estabInfo: &AgentSessionInfo{Status: 2, StatusName: "CHECKED"}}
	g := NewAgentGateway(fake, NewConnectionManager())
	g.keepalive = 50 * time.Millisecond // Speed up the test.
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

// TestHandleReadyBusy verifies that status commands forward numeric values and the reason.
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

// TestOnConnectKickOldAfterEstablish verifies that a new connection disconnects the older connection for the same agent after session establishment and remains online.
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

// TestHandleSignout verifies that successful sign-out clears the session, acknowledges it, and closes the connection through sendAndClose without panicking on a nil wsConn.
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
