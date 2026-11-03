package model

import (
	"sync"
	"time"

	"github.com/hertz-contrib/websocket"
)

// Connection wraps a WebSocket connection.
// Each connection corresponds to an agent, administrator, or monitoring dashboard client.
type Connection struct {
	// Underlying WebSocket connection.
	wsConn *websocket.Conn

	// Basic connection information.
	connID   string // Unique connection identifier
	userType string // User type: agent/admin/monitor
	userID   int64  // User ID: agentID for agents and adminID for administrators
	tenantID int64  // Tenant ID
	username string // Username

	// Buffered outbound message channel.
	// All sends run asynchronously through this channel to avoid blocking.
	sendCh chan []byte

	// Connection state.
	connected bool      // Whether the connection is established
	lastPing  time.Time // Time the last heartbeat was received
	connectAt time.Time // Connection establishment time

	// Set of subscribed topics.
	// key: topic name, value: whether subscribed
	subscriptions map[string]bool
	subMu         sync.RWMutex

	// Write lock because WebSocket WriteMessage is not concurrency-safe.
	writeMu sync.Mutex

	// Close state.
	closeOnce sync.Once
	closeCh   chan struct{} // Close signal channel
}

// NewConnection creates a new connection.
func NewConnection(wsConn *websocket.Conn, connID, userType string, userID, tenantID int64, username string) *Connection {
	return &Connection{
		wsConn:        wsConn,
		connID:        connID,
		userType:      userType,
		userID:        userID,
		tenantID:      tenantID,
		username:      username,
		sendCh:        make(chan []byte, 256), // Buffer for 256 messages
		connected:     true,
		lastPing:      time.Now(),
		connectAt:     time.Now(),
		subscriptions: make(map[string]bool),
		closeCh:       make(chan struct{}),
	}
}

// ===== Getter methods =====

// ConnID returns the connection ID.
func (c *Connection) ConnID() string {
	return c.connID
}

// UserType returns the user type.
func (c *Connection) UserType() string {
	return c.userType
}

// UserID returns the user ID.
func (c *Connection) UserID() int64 {
	return c.userID
}

// TenantID returns the tenant ID.
func (c *Connection) TenantID() int64 {
	return c.tenantID
}

// Username returns the username.
func (c *Connection) Username() string {
	return c.username
}

// IsConnected reports whether the connection is established.
func (c *Connection) IsConnected() bool {
	return c.connected
}

// LastPing returns the time of the last heartbeat.
func (c *Connection) LastPing() time.Time {
	return c.lastPing
}

// ConnectAt returns the connection establishment time.
func (c *Connection) ConnectAt() time.Time {
	return c.connectAt
}

// CloseChan returns the close signal channel.
func (c *Connection) CloseChan() <-chan struct{} {
	return c.closeCh
}

// SendChan returns the send channel.
func (c *Connection) SendChan() chan<- []byte {
	return c.sendCh
}

// ReadSendChan returns the send channel for internal reads.
func (c *Connection) ReadSendChan() <-chan []byte {
	return c.sendCh
}

// WSConn returns the underlying WebSocket connection for careful internal use.
func (c *Connection) WSConn() *websocket.Conn {
	return c.wsConn
}

// ===== Send messages =====

// Send sends a message asynchronously without blocking.
// It returns false when the send buffer is full.
func (c *Connection) Send(data []byte) bool {
	if !c.connected {
		return false
	}
	select {
	case c.sendCh <- data:
		return true
	default:
		// The buffer is full, so the send fails.
		return false
	}
}

// WriteMessage writes directly to the WebSocket for internal use by the write goroutine.
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.wsConn.WriteMessage(messageType, data)
}

// ===== Heartbeat =====

// UpdatePing updates the heartbeat time.
func (c *Connection) UpdatePing() {
	c.lastPing = time.Now()
}

// IsExpired reports whether no heartbeat has been received within the specified timeout.
func (c *Connection) IsExpired(timeout time.Duration) bool {
	return time.Since(c.lastPing) > timeout
}

// ===== Subscription management =====

// Subscribe subscribes to a topic.
func (c *Connection) Subscribe(topic string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	c.subscriptions[topic] = true
}

// Unsubscribe unsubscribes from a topic.
func (c *Connection) Unsubscribe(topic string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	delete(c.subscriptions, topic)
}

// IsSubscribed reports whether the connection is subscribed to the specified topic.
func (c *Connection) IsSubscribed(topic string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return c.subscriptions[topic]
}

// GetSubscriptions returns all subscribed topics.
func (c *Connection) GetSubscriptions() []string {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	topics := make([]string, 0, len(c.subscriptions))
	for topic := range c.subscriptions {
		topics = append(topics, topic)
	}
	return topics
}

// SubscriptionCount returns the number of subscriptions.
func (c *Connection) SubscriptionCount() int {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return len(c.subscriptions)
}

// ===== Close connection =====

// Close closes the connection idempotently.
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.connected = false
		close(c.closeCh)
		close(c.sendCh)
		// Tests can use buffer-only connections with a nil wsConn.
		if c.wsConn != nil {
			_ = c.wsConn.Close()
		}
	})
}
