// Package service provides the core business logic for the push service,
// including connection management, message push, subscription management, and multi-node routing.
package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hertz-contrib/websocket"
	"go.uber.org/zap"

	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
)

const (
	// Default heartbeat timeout.
	defaultHeartbeatTimeout = 60 * time.Second
	// Default connection cleanup check interval.
	defaultCleanupInterval = 30 * time.Second
	// Write timeout.
	writeWait = 10 * time.Second
	// Pong wait time.
	pongWait = 60 * time.Second
	// Ping interval, which must be shorter than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

// ConnectionManager manages WebSocket connections.
// It manages all connections for agents, administrators, and monitoring dashboards.
// It provides connection registration, unregistration, lookup, and message delivery.
//
// Connection organization:
// - agentConns: tenantId -> agentId -> Connection, grouped by tenant and agent
// - adminConns: adminId -> Connection
// - monitorConns: monitorId -> Connection
type ConnectionManager struct {
	// Agent connections: tenantId -> agentId -> Connection.
	// An agent can have multiple concurrent connections from different clients, so each entry is a slice.
	agentConns map[int64]map[int64][]*model.Connection

	// Admin connections: connID -> Connection.
	adminConns map[string]*model.Connection

	// Monitoring dashboard connections: connID -> Connection.
	monitorConns map[string]*model.Connection

	// Global connection index: connID -> Connection.
	connIndex map[string]*model.Connection

	// Read-write lock.
	mu sync.RWMutex

	// Statistics.
	totalMessages  atomic.Int64 // Total messages sent
	failedMessages atomic.Int64 // Failed messages

	// Heartbeat timeout.
	heartbeatTimeout time.Duration

	// Cleanup check interval.
	cleanupInterval time.Duration

	// Stop signal.
	stopCh chan struct{}
}

// NewConnectionManager creates a new connection manager.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		agentConns:       make(map[int64]map[int64][]*model.Connection),
		adminConns:       make(map[string]*model.Connection),
		monitorConns:     make(map[string]*model.Connection),
		connIndex:        make(map[string]*model.Connection),
		heartbeatTimeout: defaultHeartbeatTimeout,
		cleanupInterval:  defaultCleanupInterval,
		stopCh:           make(chan struct{}),
	}
}

// SetHeartbeatTimeout sets the heartbeat timeout.
func (m *ConnectionManager) SetHeartbeatTimeout(d time.Duration) {
	if d > 0 {
		m.heartbeatTimeout = d
	}
}

// SetCleanupInterval sets the cleanup check interval.
func (m *ConnectionManager) SetCleanupInterval(d time.Duration) {
	if d > 0 {
		m.cleanupInterval = d
	}
}

// Start starts the connection manager and its background cleanup goroutine.
func (m *ConnectionManager) Start(ctx context.Context) {
	logger.Info("connection manager started")

	// Start the connection cleanup goroutine.
	go m.cleanupLoop(ctx)
}

// Stop stops the connection manager.
func (m *ConnectionManager) Stop() {
	close(m.stopCh)

	// Close all connections.
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connIndex {
		conn.Close()
	}

	logger.Info("connection manager stopped",
		zap.Int("total_conns", len(m.connIndex)),
	)
}

// ===== Connection registration and unregistration =====

// AddConnection adds a connection.
func (m *ConnectionManager) AddConnection(conn *model.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to the global index.
	m.connIndex[conn.ConnID()] = conn

	// Store by user type.
	switch conn.UserType() {
	case model.UserTypeAgent:
		// Organize agent connections by tenant -> agent.
		tenantMap, ok := m.agentConns[conn.TenantID()]
		if !ok {
			tenantMap = make(map[int64][]*model.Connection)
			m.agentConns[conn.TenantID()] = tenantMap
		}
		tenantMap[conn.UserID()] = append(tenantMap[conn.UserID()], conn)

	case model.UserTypeAdmin:
		m.adminConns[conn.ConnID()] = conn

	case model.UserTypeMonitor:
		m.monitorConns[conn.ConnID()] = conn
	}

	logger.Info("connection added",
		zap.String("conn_id", conn.ConnID()),
		zap.String("user_type", conn.UserType()),
		zap.Int64("user_id", conn.UserID()),
		zap.Int64("tenant_id", conn.TenantID()),
		zap.Int("total_conns", len(m.connIndex)),
	)
}

// RemoveConnection removes a connection.
func (m *ConnectionManager) RemoveConnection(conn *model.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove from the global index.
	delete(m.connIndex, conn.ConnID())

	// Remove from user-type storage.
	switch conn.UserType() {
	case model.UserTypeAgent:
		if tenantMap, ok := m.agentConns[conn.TenantID()]; ok {
			if conns, ok := tenantMap[conn.UserID()]; ok {
				// Remove from the slice.
				for i, c := range conns {
					if c.ConnID() == conn.ConnID() {
						tenantMap[conn.UserID()] = append(conns[:i], conns[i+1:]...)
						break
					}
				}
				// Clean up the map when the agent has no connections.
				if len(tenantMap[conn.UserID()]) == 0 {
					delete(tenantMap, conn.UserID())
				}
			}
			// Clean up the map when the tenant has no agent connections.
			if len(tenantMap) == 0 {
				delete(m.agentConns, conn.TenantID())
			}
		}

	case model.UserTypeAdmin:
		delete(m.adminConns, conn.ConnID())

	case model.UserTypeMonitor:
		delete(m.monitorConns, conn.ConnID())
	}

	logger.Info("connection removed",
		zap.String("conn_id", conn.ConnID()),
		zap.String("user_type", conn.UserType()),
		zap.Int64("user_id", conn.UserID()),
		zap.Int64("tenant_id", conn.TenantID()),
		zap.Int("total_conns", len(m.connIndex)),
	)
}

// GetConnection returns the connection with the specified connID.
func (m *ConnectionManager) GetConnection(connID string) (*model.Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connIndex[connID]
	return conn, ok
}

// ===== Agent connection queries =====

// GetAgentConnections returns all connections for the specified agent.
func (m *ConnectionManager) GetAgentConnections(tenantID, agentID int64) []*model.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if tenantMap, ok := m.agentConns[tenantID]; ok {
		if conns, ok := tenantMap[agentID]; ok {
			// Return a copy to avoid concurrency issues.
			result := make([]*model.Connection, len(conns))
			copy(result, conns)
			return result
		}
	}
	return nil
}

// IsAgentOnline reports whether the agent is online.
func (m *ConnectionManager) IsAgentOnline(tenantID, agentID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if tenantMap, ok := m.agentConns[tenantID]; ok {
		if conns, ok := tenantMap[agentID]; ok {
			for _, conn := range conns {
				if conn.IsConnected() {
					return true
				}
			}
		}
	}
	return false
}

// GetTenantAgentIDs returns the IDs of all online agents for the specified tenant.
func (m *ConnectionManager) GetTenantAgentIDs(tenantID int64) []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantMap, ok := m.agentConns[tenantID]
	if !ok {
		return nil
	}

	agentIDs := make([]int64, 0, len(tenantMap))
	for agentID := range tenantMap {
		agentIDs = append(agentIDs, agentID)
	}
	return agentIDs
}

// GetTenantAgentConns returns all agent connections for the specified tenant.
func (m *ConnectionManager) GetTenantAgentConns(tenantID int64) []*model.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantMap, ok := m.agentConns[tenantID]
	if !ok {
		return nil
	}

	var result []*model.Connection
	for _, conns := range tenantMap {
		result = append(result, conns...)
	}
	return result
}

// GetAllAgentConnIDs returns the set of agentIDs with agent connections for route renewal.
func (m *ConnectionManager) GetAllAgentConnIDs() map[int64]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[int64]bool)
	for _, tenantMap := range m.agentConns {
		for agentID, conns := range tenantMap {
			if len(conns) > 0 {
				result[agentID] = true
			}
		}
	}
	return result
}

// ===== Admin and monitor connection queries =====

// GetAllAdminConnections returns all administrator connections.
func (m *ConnectionManager) GetAllAdminConnections() []*model.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*model.Connection, 0, len(m.adminConns))
	for _, conn := range m.adminConns {
		result = append(result, conn)
	}
	return result
}

// GetAllMonitorConnections returns all monitoring connections.
func (m *ConnectionManager) GetAllMonitorConnections() []*model.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*model.Connection, 0, len(m.monitorConns))
	for _, conn := range m.monitorConns {
		result = append(result, conn)
	}
	return result
}

// ===== Statistics =====

// GetStats returns statistics.
func (m *ConnectionManager) GetStats() *model.Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agentCount := 0
	for _, tenantMap := range m.agentConns {
		for _, conns := range tenantMap {
			agentCount += len(conns)
		}
	}

	// Count subscriptions for each topic.
	topicCount := make(map[string]int)
	for _, conn := range m.connIndex {
		for _, topic := range conn.GetSubscriptions() {
			topicCount[topic]++
		}
	}

	return &model.Stats{
		TotalConns:     len(m.connIndex),
		AgentConns:     agentCount,
		AdminConns:     len(m.adminConns),
		MonitorConns:   len(m.monitorConns),
		TenantCount:    len(m.agentConns),
		TotalMessages:  m.totalMessages.Load(),
		FailedMessages: m.failedMessages.Load(),
		Topics:         topicCount,
	}
}

// IncrementTotalMessages increments the total message count.
func (m *ConnectionManager) IncrementTotalMessages() {
	m.totalMessages.Add(1)
}

// IncrementFailedMessages increments the failed message count.
func (m *ConnectionManager) IncrementFailedMessages() {
	m.failedMessages.Add(1)
}

// ===== Background cleanup =====

// cleanupLoop runs the connection cleanup loop.
// It periodically checks for and cleans up expired connections.
func (m *ConnectionManager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("connection cleanup loop stopped by context")
			return
		case <-m.stopCh:
			logger.Info("connection cleanup loop stopped")
			return
		case <-ticker.C:
			m.cleanupExpiredConnections()
		}
	}
}

// cleanupExpiredConnections cleans up expired connections.
func (m *ConnectionManager) cleanupExpiredConnections() {
	m.mu.RLock()
	// Collect all potentially expired connections first.
	var expired []*model.Connection
	for _, conn := range m.connIndex {
		if conn.IsExpired(m.heartbeatTimeout) {
			expired = append(expired, conn)
		}
	}
	m.mu.RUnlock()

	if len(expired) == 0 {
		return
	}

	logger.Warn("cleaning up expired connections",
		zap.Int("count", len(expired)),
	)

	// Close and remove each expired connection.
	for _, conn := range expired {
		logger.Warn("connection expired, closing",
			zap.String("conn_id", conn.ConnID()),
			zap.String("user_type", conn.UserType()),
			zap.Int64("user_id", conn.UserID()),
		)
		conn.Close()
		m.RemoveConnection(conn)
	}
}

// ===== Connection read/write handling =====

// StartWritePump starts the connection's write goroutine.
// It reads messages from sendCh and sends them to the WebSocket.
// It also sends heartbeat pings.
func (m *ConnectionManager) StartWritePump(conn *model.Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case <-conn.CloseChan():
			// The connection is closed.
			return

		case message, ok := <-conn.ReadSendChan():
			// Read a message from the send channel.
			_ = conn.WSConn().SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The channel is closed.
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Warn("write message failed",
					zap.String("conn_id", conn.ConnID()),
					zap.Error(err),
				)
				m.failedMessages.Add(1)
				return
			}
			m.totalMessages.Add(1)

		case <-ticker.C:
			// Send a heartbeat ping.
			_ = conn.WSConn().SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Debug("ping failed, connection may be closed",
					zap.String("conn_id", conn.ConnID()),
					zap.Error(err),
				)
				return
			}
		}
	}
}
