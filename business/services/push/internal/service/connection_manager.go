// Package service 提供推送服务的核心业务逻辑
// 包括连接管理、消息推送、订阅管理、多节点路由等功能
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
	// 默认心跳超时时间
	defaultHeartbeatTimeout = 60 * time.Second
	// 默认连接清理检查间隔
	defaultCleanupInterval = 30 * time.Second
	// 写超时时间
	writeWait = 10 * time.Second
	// pong 等待时间
	pongWait = 60 * time.Second
	// ping 发送周期（应小于 pongWait）
	pingPeriod = (pongWait * 9) / 10
)

// ConnectionManager 连接管理器
// 负责管理所有 WebSocket 连接，包括座席连接、管理员连接、监控连接
// 提供连接的注册、注销、查询、消息发送等功能
//
// 连接组织结构：
// - agentConns: tenantId -> agentId -> Connection（座席连接按租户和座席分组）
// - adminConns: adminId -> Connection（管理员连接）
// - monitorConns: monitorId -> Connection（监控大屏连接）
type ConnectionManager struct {
	// 座席连接：tenantId -> agentId -> Connection
	// 座席可能同时有多个连接（多端登录），所以是 slice
	agentConns map[int64]map[int64][]*model.Connection

	// 管理员连接：connID -> Connection
	adminConns map[string]*model.Connection

	// 监控大屏连接：connID -> Connection
	monitorConns map[string]*model.Connection

	// 全局连接索引：connID -> Connection
	connIndex map[string]*model.Connection

	// 读写锁
	mu sync.RWMutex

	// 统计信息
	totalMessages  atomic.Int64 // 总发送消息数
	failedMessages atomic.Int64 // 失败消息数

	// 心跳超时时间
	heartbeatTimeout time.Duration

	// 清理检查间隔
	cleanupInterval time.Duration

	// 停止信号
	stopCh chan struct{}
}

// NewConnectionManager 创建新的连接管理器
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

// SetHeartbeatTimeout 设置心跳超时时间
func (m *ConnectionManager) SetHeartbeatTimeout(d time.Duration) {
	if d > 0 {
		m.heartbeatTimeout = d
	}
}

// SetCleanupInterval 设置清理检查间隔
func (m *ConnectionManager) SetCleanupInterval(d time.Duration) {
	if d > 0 {
		m.cleanupInterval = d
	}
}

// Start 启动连接管理器（启动后台清理协程）
func (m *ConnectionManager) Start(ctx context.Context) {
	logger.Info("connection manager started")

	// 启动连接清理协程
	go m.cleanupLoop(ctx)
}

// Stop 停止连接管理器
func (m *ConnectionManager) Stop() {
	close(m.stopCh)

	// 关闭所有连接
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connIndex {
		conn.Close()
	}

	logger.Info("connection manager stopped",
		zap.Int("total_conns", len(m.connIndex)),
	)
}

// ===== 连接注册/注销 =====

// AddConnection 添加连接
func (m *ConnectionManager) AddConnection(conn *model.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 添加到全局索引
	m.connIndex[conn.ConnID()] = conn

	// 按用户类型分类存储
	switch conn.UserType() {
	case model.UserTypeAgent:
		// 座席连接：按 tenant -> agent 组织
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

// RemoveConnection 移除连接
func (m *ConnectionManager) RemoveConnection(conn *model.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从全局索引移除
	delete(m.connIndex, conn.ConnID())

	// 从分类存储中移除
	switch conn.UserType() {
	case model.UserTypeAgent:
		if tenantMap, ok := m.agentConns[conn.TenantID()]; ok {
			if conns, ok := tenantMap[conn.UserID()]; ok {
				// 从 slice 中移除
				for i, c := range conns {
					if c.ConnID() == conn.ConnID() {
						tenantMap[conn.UserID()] = append(conns[:i], conns[i+1:]...)
						break
					}
				}
				// 如果该座席没有连接了，清理 map
				if len(tenantMap[conn.UserID()]) == 0 {
					delete(tenantMap, conn.UserID())
				}
			}
			// 如果该租户没有座席连接了，清理 map
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

// GetConnection 根据 connID 获取连接
func (m *ConnectionManager) GetConnection(connID string) (*model.Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connIndex[connID]
	return conn, ok
}

// ===== 座席连接查询 =====

// GetAgentConnections 获取指定座席的所有连接
func (m *ConnectionManager) GetAgentConnections(tenantID, agentID int64) []*model.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if tenantMap, ok := m.agentConns[tenantID]; ok {
		if conns, ok := tenantMap[agentID]; ok {
			// 返回副本，避免并发问题
			result := make([]*model.Connection, len(conns))
			copy(result, conns)
			return result
		}
	}
	return nil
}

// IsAgentOnline 判断座席是否在线
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

// GetTenantAgentIDs 获取指定租户的所有在线座席ID
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

// GetTenantAgentConns 获取指定租户的所有座席连接
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

// GetAllAgentConnIDs 获取所有座席连接的 agentID 集合（用于路由续期）
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

// ===== 管理员/监控连接查询 =====

// GetAllAdminConnections 获取所有管理员连接
func (m *ConnectionManager) GetAllAdminConnections() []*model.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*model.Connection, 0, len(m.adminConns))
	for _, conn := range m.adminConns {
		result = append(result, conn)
	}
	return result
}

// GetAllMonitorConnections 获取所有监控连接
func (m *ConnectionManager) GetAllMonitorConnections() []*model.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*model.Connection, 0, len(m.monitorConns))
	for _, conn := range m.monitorConns {
		result = append(result, conn)
	}
	return result
}

// ===== 统计信息 =====

// GetStats 获取统计信息
func (m *ConnectionManager) GetStats() *model.Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agentCount := 0
	for _, tenantMap := range m.agentConns {
		for _, conns := range tenantMap {
			agentCount += len(conns)
		}
	}

	// 统计各主题订阅数
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

// IncrementTotalMessages 增加总消息计数
func (m *ConnectionManager) IncrementTotalMessages() {
	m.totalMessages.Add(1)
}

// IncrementFailedMessages 增加失败消息计数
func (m *ConnectionManager) IncrementFailedMessages() {
	m.failedMessages.Add(1)
}

// ===== 后台清理 =====

// cleanupLoop 连接清理循环
// 定期检查并清理超时的连接
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

// cleanupExpiredConnections 清理过期连接
func (m *ConnectionManager) cleanupExpiredConnections() {
	m.mu.RLock()
	// 先收集所有可能过期的连接
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

	// 逐个关闭并移除过期连接
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

// ===== 连接读写处理 =====

// StartWritePump 启动连接的写协程
// 负责从 sendCh 读取消息并发送到 WebSocket
// 同时负责发送心跳 ping
func (m *ConnectionManager) StartWritePump(conn *model.Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case <-conn.CloseChan():
			// 连接已关闭
			return

		case message, ok := <-conn.ReadSendChan():
			// 从发送通道读取消息
			_ = conn.WSConn().SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道已关闭
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
			// 发送心跳 ping
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
