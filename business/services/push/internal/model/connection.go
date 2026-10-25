package model

import (
	"sync"
	"time"

	"github.com/hertz-contrib/websocket"
)

// Connection WebSocket 连接封装
// 每个连接对应一个客户端（座席/管理员/监控大屏）
type Connection struct {
	// 底层 WebSocket 连接
	wsConn *websocket.Conn

	// 连接基本信息
	connID   string // 连接唯一标识
	userType string // 用户类型: agent/admin/monitor
	userID   int64  // 用户ID（座席为 agentID，管理员为 adminID）
	tenantID int64  // 租户ID
	username string // 用户名

	// 发送消息缓冲通道
	// 所有发送操作通过此通道异步执行，避免阻塞
	sendCh chan []byte

	// 连接状态
	connected bool      // 是否已连接
	lastPing  time.Time // 最后一次收到心跳的时间
	connectAt time.Time // 连接建立时间

	// 订阅的主题集合
	// key: 主题名, value: 是否订阅
	subscriptions map[string]bool
	subMu         sync.RWMutex

	// 写锁（WebSocket 的 WriteMessage 不是并发安全的）
	writeMu sync.Mutex

	// 关闭相关
	closeOnce sync.Once
	closeCh   chan struct{} // 关闭信号通道
}

// NewConnection 创建新的连接
func NewConnection(wsConn *websocket.Conn, connID, userType string, userID, tenantID int64, username string) *Connection {
	return &Connection{
		wsConn:        wsConn,
		connID:        connID,
		userType:      userType,
		userID:        userID,
		tenantID:      tenantID,
		username:      username,
		sendCh:        make(chan []byte, 256), // 256 条消息缓冲
		connected:     true,
		lastPing:      time.Now(),
		connectAt:     time.Now(),
		subscriptions: make(map[string]bool),
		closeCh:       make(chan struct{}),
	}
}

// ===== Getter 方法 =====

// ConnID 获取连接ID
func (c *Connection) ConnID() string {
	return c.connID
}

// UserType 获取用户类型
func (c *Connection) UserType() string {
	return c.userType
}

// UserID 获取用户ID
func (c *Connection) UserID() int64 {
	return c.userID
}

// TenantID 获取租户ID
func (c *Connection) TenantID() int64 {
	return c.tenantID
}

// Username 获取用户名
func (c *Connection) Username() string {
	return c.username
}

// IsConnected 是否处于连接状态
func (c *Connection) IsConnected() bool {
	return c.connected
}

// LastPing 最后心跳时间
func (c *Connection) LastPing() time.Time {
	return c.lastPing
}

// ConnectAt 连接建立时间
func (c *Connection) ConnectAt() time.Time {
	return c.connectAt
}

// CloseChan 返回关闭信号通道
func (c *Connection) CloseChan() <-chan struct{} {
	return c.closeCh
}

// SendChan 返回发送通道
func (c *Connection) SendChan() chan<- []byte {
	return c.sendCh
}

// ReadSendChan 读取发送通道（内部使用）
func (c *Connection) ReadSendChan() <-chan []byte {
	return c.sendCh
}

// WSConn 获取底层 WebSocket 连接（内部使用，谨慎操作）
func (c *Connection) WSConn() *websocket.Conn {
	return c.wsConn
}

// ===== 发送消息 =====

// Send 异步发送消息（非阻塞）
// 如果发送缓冲区满了，返回 false 表示发送失败
func (c *Connection) Send(data []byte) bool {
	if !c.connected {
		return false
	}
	select {
	case c.sendCh <- data:
		return true
	default:
		// 缓冲区满，发送失败
		return false
	}
}

// WriteMessage 直接写入 WebSocket 消息（写协程内部使用）
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.wsConn.WriteMessage(messageType, data)
}

// ===== 心跳 =====

// UpdatePing 更新心跳时间
func (c *Connection) UpdatePing() {
	c.lastPing = time.Now()
}

// IsExpired 判断连接是否过期（超过指定时间未收到心跳）
func (c *Connection) IsExpired(timeout time.Duration) bool {
	return time.Since(c.lastPing) > timeout
}

// ===== 订阅管理 =====

// Subscribe 订阅主题
func (c *Connection) Subscribe(topic string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	c.subscriptions[topic] = true
}

// Unsubscribe 取消订阅主题
func (c *Connection) Unsubscribe(topic string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	delete(c.subscriptions, topic)
}

// IsSubscribed 判断是否订阅了指定主题
func (c *Connection) IsSubscribed(topic string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return c.subscriptions[topic]
}

// GetSubscriptions 获取所有订阅的主题
func (c *Connection) GetSubscriptions() []string {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	topics := make([]string, 0, len(c.subscriptions))
	for topic := range c.subscriptions {
		topics = append(topics, topic)
	}
	return topics
}

// SubscriptionCount 获取订阅数量
func (c *Connection) SubscriptionCount() int {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return len(c.subscriptions)
}

// ===== 关闭连接 =====

// Close 关闭连接（幂等）
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.connected = false
		close(c.closeCh)
		close(c.sendCh)
		// 测试场景允许 wsConn 为 nil 的纯缓冲连接，此处最小防御
		if c.wsConn != nil {
			_ = c.wsConn.Close()
		}
	})
}
