// Package service provides the business logic layer for the push service.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"vocara/pkg/logger"
	"vocara/services/push/internal/model"

	"go.uber.org/zap"
)

// tenantStr consistently converts a tenant ID to a string; conn stores it as int64.
func tenantStr(tenantID int64) string { return strconv.FormatInt(tenantID, 10) }

// AgentGateway connects the WS connection lifecycle to agentcenter internal APIs.
// It signs in on connection with idempotent recovery, disconnects older connections for the same agent, renews sessions with a 30-second keepalive,
// and forwards ready/busy/signout commands with acknowledgments. The state machine resides in agentcenter; this gateway is stateless.
type AgentGateway struct {
	client    AgentCenterAPI
	connMgr   *ConnectionManager
	keepalive time.Duration // Keepalive interval; defaults to 30s
	maxMiss   int           // Maximum consecutive renewal failures; defaults to 3

	mu        sync.Mutex
	stopChans map[string]chan struct{} // connID -> keepalive stop signal
}

// NewAgentGateway creates an agent command gateway. A nil client disables the gateway when agentcenter_base_url is not configured.
func NewAgentGateway(client AgentCenterAPI, connMgr *ConnectionManager) *AgentGateway {
	return &AgentGateway{
		client:    client,
		connMgr:   connMgr,
		keepalive: 30 * time.Second,
		maxMiss:   3,
		stopChans: make(map[string]chan struct{}),
	}
}

// Enabled reports whether the gateway is available.
func (g *AgentGateway) Enabled() bool { return g != nil && g.client != nil }

// OnConnect signs in on connection by idempotently establishing or restoring a session, disconnecting older connections, and starting keepalive.
// The session is established before older connections are disconnected so they remain active if establishment fails; reconnects restore the session idempotently.
// An error indicates sign-in failure; the caller notifies the frontend and closes the connection.
func (g *AgentGateway) OnConnect(ctx context.Context, conn *model.Connection, dn string) (*AgentSessionInfo, error) {
	if !g.Enabled() {
		return nil, fmt.Errorf("agent gateway disabled")
	}
	tenantID := conn.TenantID()
	agentID := conn.UserID() // For agent connections, userID is the agent number converted by ws_handler.

	// 1. Idempotently establish or restore the session.
	info, err := g.client.EstablishSession(ctx, tenantStr(tenantID), strconv.FormatInt(agentID, 10), dn)
	if err != nil {
		return nil, err
	}

	// 2. Disconnect older connections that are still alive. Dead connections from reconnects are skipped.
	for _, old := range g.connMgr.GetAgentConnections(tenantID, agentID) {
		if old.ConnID() == conn.ConnID() {
			continue
		}
		g.sendAndClose(old, map[string]interface{}{
			"type": "kick",
			"data": map[string]interface{}{"reason": "duplicate_login"},
		})
		logger.Info("kick old agent connection",
			zap.String("old_conn_id", old.ConnID()),
			zap.String("new_conn_id", conn.ConnID()),
			zap.Int64("agent_id", agentID),
		)
	}

	// 3. Start the keepalive goroutine.
	g.startKeepalive(conn)
	return info, nil
}

// OnDisconnect stops only the keepalive goroutine. It does not change agent status; the session expires after 120 seconds.
func (g *AgentGateway) OnDisconnect(conn *model.Connection) {
	g.mu.Lock()
	stop, ok := g.stopChans[conn.ConnID()]
	if ok {
		delete(g.stopChans, conn.ConnID())
	}
	g.mu.Unlock()
	if ok {
		close(stop)
	}
}

// startKeepalive calls session/refresh every 30 seconds and disconnects after maxMiss consecutive failures.
func (g *AgentGateway) startKeepalive(conn *model.Connection) {
	stop := make(chan struct{})
	g.mu.Lock()
	g.stopChans[conn.ConnID()] = stop
	g.mu.Unlock()

	go func() {
		ticker := time.NewTicker(g.keepalive)
		defer ticker.Stop()
		miss := 0
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				err := g.client.RefreshSession(ctx, tenantStr(conn.TenantID()), strconv.FormatInt(conn.UserID(), 10))
				cancel()
				if err != nil {
					miss++
					logger.Warn("agent session refresh failed",
						zap.String("conn_id", conn.ConnID()),
						zap.Int("miss", miss),
						zap.Error(err),
					)
					if miss >= g.maxMiss {
						g.sendAndClose(conn, map[string]interface{}{
							"type":   "session",
							"action": "error",
							"data":   map[string]interface{}{"message": "会话续期失败，请重新连接"},
						})
						return
					}
				} else {
					miss = 0
				}
			}
		}
	}()
}

// doStatus forwards a status command and sends an acknowledgment.
func (g *AgentGateway) doStatus(conn *model.Connection, action string, status int, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	err := g.client.UpdateStatus(ctx, tenantStr(conn.TenantID()), strconv.FormatInt(conn.UserID(), 10), status, reason)
	g.sendAck(conn, action, err)
}

// HandleReady sets the agent to ready. agentStatusReady matches agentcenter model.StatusReady.
func (g *AgentGateway) HandleReady(conn *model.Connection) {
	g.doStatus(conn, "ready", agentStatusReady, "")
}

// HandleBusy sets the agent to busy with an optional reason.
func (g *AgentGateway) HandleBusy(conn *model.Connection, reason string) {
	g.doStatus(conn, "busy", agentStatusBusy, reason)
}

// HandleSignout clears the session, acknowledges sign-out, and closes the connection.
func (g *AgentGateway) HandleSignout(conn *model.Connection) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	err := g.client.DeleteSession(ctx, tenantStr(conn.TenantID()), strconv.FormatInt(conn.UserID(), 10))
	if err != nil {
		g.sendAck(conn, "signout", err)
		return
	}
	// On success, sendAndClose emits the ack frame and allows a flush window before closing the connection.
	g.sendAndClose(conn, ackMessage("signout", nil))
}

// ackMessage constructs a command acknowledgment frame.
func ackMessage(action string, err error) map[string]interface{} {
	msg := map[string]interface{}{
		"type":   "ack",
		"action": action,
		"ok":     err == nil,
	}
	if err != nil {
		msg["error"] = err.Error()
	}
	return msg
}

// sendAck sends a command acknowledgment.
func (g *AgentGateway) sendAck(conn *model.Connection, action string, err error) {
	g.sendJSON(conn, ackMessage(action, err))
}

// sendJSON sends a JSON message through conn.Send for consumption by the write goroutine.
func (g *AgentGateway) sendJSON(conn *model.Connection, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	conn.Send(data)
}

// sendAndClose sends an outbound frame and allows a flush window before closing the connection.
// This prevents the write goroutine from observing the close signal and dropping frames buffered asynchronously by Send.
func (g *AgentGateway) sendAndClose(conn *model.Connection, v interface{}) {
	g.sendJSON(conn, v)
	time.Sleep(100 * time.Millisecond)
	conn.Close()
}

// Agentcenter status values match services/agentcenter/internal/model/status.go.
// The push service does not import agentcenter, avoiding a cross-service dependency.
const (
	agentStatusReady = 3 // StatusReady
	agentStatusBusy  = 9 // StatusBusy
)
