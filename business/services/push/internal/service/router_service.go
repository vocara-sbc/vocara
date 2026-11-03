package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	pkgredis "vocara/pkg/redis"
	"vocara/pkg/config"
	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
)

const (
	// Redis route key prefix.
	redisRoutePrefix = "vocara:ws:route:"

	// Redis route expiration time, renewed by heartbeats.
	routeTTL = 120 * time.Second

	// NATS node topic.
	natsNodePrefix = "vocara.ws.node."

	// NATS topic subscribed to by the local node.
	natsLocalTopic = "vocara.ws.local"
)

// RouterService provides multi-node routing.
// It routes messages to the correct node in multi-node deployments.
//
// How it works:
// 1. When an agent establishes a WebSocket connection, register route information in Redis (agentId -> nodeId).
// 2. Before sending a message to an agent, query the Redis routing table.
// 3. Push directly when the agent is on the local node.
// 4. Forward through NATS/Redis PubSub when the agent is on another node.
// 5. The target node receives the message and pushes it to local connections.
//
// Supported backends:
// - nats: use NATS for message routing (recommended for better performance)
// - redis: use Redis Pub/Sub as a simpler deployment option
type RouterService struct {
	// Configuration.
	nodeID   string // Local node ID
	nodeAddr string // Local node address
	mode     string // Routing mode: nats / redis / none

	// Redis client for the routing table and Pub/Sub.
	redisClient *pkgredis.Client

	// NATS configuration.
	natsCfg *config.NATSConfig

	// NATS connection.
	natsConn *nats.Conn

	// Connection manager.
	connManager *ConnectionManager

	// Subscription service.
	subService *SubscriptionService

	// Whether routing is enabled.
	enabled bool

	// Stop signal.
	stopCh chan struct{}
	once   sync.Once
}

// NewRouterService creates a router service.
func NewRouterService(
	nodeID, nodeAddr string,
	cfg *config.Config,
	connManager *ConnectionManager,
	subService *SubscriptionService,
) *RouterService {
	mode := "redis" // Use Redis mode by default.

	// Prefer NATS when configured.
	if cfg.NATS.URLs != "" {
		mode = "nats"
	}

	rs := &RouterService{
		nodeID:      nodeID,
		nodeAddr:    nodeAddr,
		mode:        mode,
		natsCfg:     &cfg.NATS,
		connManager: connManager,
		subService:  subService,
		stopCh:      make(chan struct{}),
	}

	return rs
}

// Start starts the router service.
func (r *RouterService) Start(ctx context.Context) error {
	// Get the Redis client.
	r.redisClient = pkgredis.Get()
	if r.redisClient == nil {
		logger.Warn("redis client not available, router service disabled")
		return nil
	}

	// Initialize the connection for the selected mode.
	switch r.mode {
	case "nats":
		if err := r.initNATS(); err != nil {
			logger.Warn("init nats failed, fallback to redis mode", zap.Error(err))
			r.mode = "redis"
			// Continue in Redis mode.
		}

	case "redis":
		// Redis mode uses the Redis client directly.
	}

	r.enabled = true

	// Start the node subscription.
	go r.startLocalConsumer(ctx)

	// Start route renewal heartbeats.
	go r.heartbeatLoop(ctx)

	logger.Info("router service started",
		zap.String("node_id", r.nodeID),
		zap.String("node_addr", r.nodeAddr),
		zap.String("mode", r.mode),
	)

	return nil
}

// Stop stops the router service.
func (r *RouterService) Stop() {
	r.once.Do(func() {
		close(r.stopCh)

		if r.natsConn != nil {
			r.natsConn.Close()
		}

		logger.Info("router service stopped")
	})
}

// IsEnabled reports whether the router service is enabled.
func (r *RouterService) IsEnabled() bool {
	return r.enabled
}

// NodeID returns the node ID.
func (r *RouterService) NodeID() string {
	return r.nodeID
}

// ===== NATS initialization =====

func (r *RouterService) initNATS() error {
	natsCfg := r.natsCfg
	urls := natsCfg.URLs
	if urls == "" {
		urls = nats.DefaultURL
	}

	opts := []nats.Option{
		nats.Name("vocara-push-router"),
		nats.MaxReconnects(natsCfg.MaxReconnect),
		nats.ReconnectWait(time.Duration(natsCfg.ReconnectWait) * time.Second),
		nats.Timeout(time.Duration(natsCfg.Timeout) * time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			logger.Warn("router nats disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			logger.Info("router nats reconnected", zap.String("url", conn.ConnectedUrl()))
		}),
	}
	if natsCfg.ReconnectBuf > 0 {
		opts = append(opts, nats.ReconnectBufSize(natsCfg.ReconnectBuf))
	}
	if natsCfg.User != "" {
		opts = append(opts, nats.UserInfo(natsCfg.User, natsCfg.Password))
	}
	if natsCfg.Token != "" {
		opts = append(opts, nats.Token(natsCfg.Token))
	}

	var err error
	r.natsConn, err = nats.Connect(urls, opts...)
	if err != nil {
		return fmt.Errorf("connect nats failed: %w", err)
	}
	return nil
}

// ===== Route registration =====

// RegisterRoute registers an agent connection route.
// It is called when an agent establishes a connection and records which node hosts the agentId.
func (r *RouterService) RegisterRoute(tenantID, agentID int64, connID string) error {
	if !r.enabled || r.redisClient == nil {
		return nil
	}

	routeInfo := &model.RouteInfo{
		AgentID:    agentID,
		TenantID:   tenantID,
		NodeID:     r.nodeID,
		NodeAddr:   r.nodeAddr,
		ConnID:     connID,
		ServerTime: time.Now().Unix(),
	}

	data, err := json.Marshal(routeInfo)
	if err != nil {
		return fmt.Errorf("marshal route info failed: %w", err)
	}

	key := r.agentRouteKey(agentID)
	ctx := context.Background()
	if err := r.redisClient.Set(ctx, key, string(data), routeTTL); err != nil {
		return fmt.Errorf("set route info failed: %w", err)
	}

	logger.Debug("register agent route",
		zap.Int64("agent_id", agentID),
		zap.String("node_id", r.nodeID),
	)

	return nil
}

// UnregisterRoute unregisters an agent connection route.
// It is called when the agent disconnects and compares the disconnecting connection's connID.
// It deletes the route only when the connID recorded in the routing table matches,
// preventing cleanup of an older connection from deleting a route registered by another connection for the same agent.
func (r *RouterService) UnregisterRoute(agentID int64, connID string) error {
	if !r.enabled || r.redisClient == nil {
		return nil
	}

	key := r.agentRouteKey(agentID)
	ctx := context.Background()

	// Delete only when the route points to this node and connID matches, avoiding deletion of another node's or connection's route.
	routeInfo, err := r.GetRoute(agentID)
	if err != nil {
		// Delete directly when reading fails because the route may have expired.
		_, _ = r.redisClient.Del(ctx, key)
		return nil
	}

	if routeInfo.NodeID != r.nodeID {
		return nil
	}

	// A connID mismatch means another connection for the agent owns the route, so skip deletion.
	if routeInfo.ConnID != connID {
		logger.Debug("skip unregister route, conn id mismatch",
			zap.Int64("agent_id", agentID),
			zap.String("route_conn_id", routeInfo.ConnID),
			zap.String("conn_id", connID),
		)
		return nil
	}

	if _, err = r.redisClient.Del(ctx, key); err != nil {
		return fmt.Errorf("delete route info failed: %w", err)
	}

	logger.Debug("unregister agent route",
		zap.Int64("agent_id", agentID),
		zap.String("node_id", r.nodeID),
		zap.String("conn_id", connID),
	)

	return nil
}

// GetRoute returns routing information for an agent.
func (r *RouterService) GetRoute(agentID int64) (*model.RouteInfo, error) {
	if !r.enabled || r.redisClient == nil {
		return nil, fmt.Errorf("router service not enabled")
	}

	key := r.agentRouteKey(agentID)
	ctx := context.Background()
	data, err := r.redisClient.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("route not found for agent %d", agentID)
		}
		return nil, fmt.Errorf("get route info failed: %w", err)
	}

	var routeInfo model.RouteInfo
	if err := json.Unmarshal([]byte(data), &routeInfo); err != nil {
		return nil, fmt.Errorf("unmarshal route info failed: %w", err)
	}

	return &routeInfo, nil
}

// RenewRoute renews a route when a heartbeat is received.
func (r *RouterService) RenewRoute(agentID int64) error {
	if !r.enabled || r.redisClient == nil {
		return nil
	}

	key := r.agentRouteKey(agentID)
	ctx := context.Background()
	ok, err := r.redisClient.Expire(ctx, key, routeTTL)
	if err != nil {
		return fmt.Errorf("renew route failed: %w", err)
	}

	// Register the route again if the key does not exist.
	if !ok {
		// The caller handles registration.
		return fmt.Errorf("route key not exists")
	}

	return nil
}

// agentRouteKey generates an agent route key.
func (r *RouterService) agentRouteKey(agentID int64) string {
	return fmt.Sprintf("%s%d", redisRoutePrefix, agentID)
}

// ===== Message routing =====

// PushToAgent pushes to an agent across nodes.
func (r *RouterService) PushToAgent(tenantID, agentID int64, msg *model.Message) (int, error) {
	if !r.enabled {
		return 0, nil
	}

	// Look up the route first.
	routeInfo, err := r.GetRoute(agentID)
	if err != nil {
		return 0, err
	}

	// Skip agents on the local node because local delivery handles them.
	if routeInfo.NodeID == r.nodeID {
		return 0, nil
	}

	// Construct the route message.
	routeMsg := &model.RouteMessage{
		TargetType: "agent",
		TargetID:   fmt.Sprintf("%d", agentID),
		TenantID:   tenantID,
		Event:      msg.Event,
		EventID:    msg.EventID,
		Timestamp:  msg.Timestamp,
		FromNode:   r.nodeID,
	}

	// Serialize the payload.
	payload, err := json.Marshal(msg.Data)
	if err != nil {
		return 0, fmt.Errorf("marshal payload failed: %w", err)
	}
	routeMsg.Payload = payload

	// Send to the target node.
	err = r.publishToNode(routeInfo.NodeID, routeMsg)
	if err != nil {
		return 0, err
	}

	return 1, nil
}

// PushToTopic pushes to a topic across nodes.
func (r *RouterService) PushToTopic(topic string, msg *model.Message) (int, error) {
	if !r.enabled {
		return 0, nil
	}

	// Construct the route message.
	routeMsg := &model.RouteMessage{
		TargetType: "topic",
		TargetID:   topic,
		Event:      msg.Event,
		EventID:    msg.EventID,
		Timestamp:  msg.Timestamp,
		FromNode:   r.nodeID,
	}

	payload, err := json.Marshal(msg.Data)
	if err != nil {
		return 0, fmt.Errorf("marshal payload failed: %w", err)
	}
	routeMsg.Payload = payload

	// Publish to every node's local topic; each node filters its own subscriptions.
	err = r.publishToAll(routeMsg)
	if err != nil {
		return 0, err
	}

	// Cross-node delivery cannot be counted precisely, so return 0; each node tracks its own count.
	return 0, nil
}

// PushToAll broadcasts across nodes.
func (r *RouterService) PushToAll(msg *model.Message) (int, error) {
	if !r.enabled {
		return 0, nil
	}

	routeMsg := &model.RouteMessage{
		TargetType: "all",
		TargetID:   "all",
		Event:      msg.Event,
		EventID:    msg.EventID,
		Timestamp:  msg.Timestamp,
		FromNode:   r.nodeID,
	}

	payload, err := json.Marshal(msg.Data)
	if err != nil {
		return 0, fmt.Errorf("marshal payload failed: %w", err)
	}
	routeMsg.Payload = payload

	err = r.publishToAll(routeMsg)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

// ===== Publish messages =====

// publishToNode publishes a message to a specific node.
func (r *RouterService) publishToNode(nodeID string, routeMsg *model.RouteMessage) error {
	data, err := json.Marshal(routeMsg)
	if err != nil {
		return fmt.Errorf("marshal route message failed: %w", err)
	}

	switch r.mode {
	case "nats":
		topic := natsNodePrefix + nodeID
		return r.natsConn.Publish(topic, data)

	case "redis":
		topic := natsNodePrefix + nodeID
		ctx := context.Background()
		return r.redisClient.Raw().Publish(ctx, topic, data).Err()

	default:
		return fmt.Errorf("unknown router mode: %s", r.mode)
	}
}

// publishToAll publishes a message to all nodes.
func (r *RouterService) publishToAll(routeMsg *model.RouteMessage) error {
	data, err := json.Marshal(routeMsg)
	if err != nil {
		return fmt.Errorf("marshal route message failed: %w", err)
	}

	switch r.mode {
	case "nats":
		return r.natsConn.Publish(natsLocalTopic, data)

	case "redis":
		ctx := context.Background()
		return r.redisClient.Raw().Publish(ctx, natsLocalTopic, data).Err()

	default:
		return fmt.Errorf("unknown router mode: %s", r.mode)
	}
}

// ===== Local consumer =====

// startLocalConsumer starts the local message consumer.
// It receives messages forwarded by other nodes and pushes them to local connections.
func (r *RouterService) startLocalConsumer(ctx context.Context) {
	switch r.mode {
	case "nats":
		r.startNATSConsumer(ctx)
	case "redis":
		r.startRedisConsumer(ctx)
	}
}

// startNATSConsumer starts the NATS consumer.
func (r *RouterService) startNATSConsumer(ctx context.Context) {
	if r.natsConn == nil {
		return
	}

	// Subscribe to the node-specific topic.
	nodeTopic := natsNodePrefix + r.nodeID
	_, err := r.natsConn.Subscribe(nodeTopic, func(m *nats.Msg) {
		r.handleRouteMessage(m.Data)
	})
	if err != nil {
		logger.Error("nats subscribe node topic failed",
			zap.String("topic", nodeTopic),
			zap.Error(err),
		)
		return
	}

	// Subscribe to the global broadcast topic.
	_, err = r.natsConn.Subscribe(natsLocalTopic, func(m *nats.Msg) {
		r.handleRouteMessage(m.Data)
	})
	if err != nil {
		logger.Error("nats subscribe local topic failed",
			zap.String("topic", natsLocalTopic),
			zap.Error(err),
		)
		return
	}

	logger.Info("nats consumer started",
		zap.String("node_topic", nodeTopic),
		zap.String("local_topic", natsLocalTopic),
	)

	// Wait for the stop signal.
	select {
	case <-ctx.Done():
	case <-r.stopCh:
	}
}

// startRedisConsumer starts the Redis Pub/Sub consumer.
func (r *RouterService) startRedisConsumer(ctx context.Context) {
	if r.redisClient == nil {
		return
	}

	nodeTopic := natsNodePrefix + r.nodeID
	topics := []string{nodeTopic, natsLocalTopic}

	pubsub := r.redisClient.Raw().Subscribe(ctx, topics...)
	defer pubsub.Close()

	logger.Info("redis pubsub consumer started",
		zap.Strings("topics", topics),
	)

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			logger.Info("redis consumer stopped by context")
			return
		case <-r.stopCh:
			logger.Info("redis consumer stopped")
			return
		case msg, ok := <-ch:
			if !ok {
				logger.Warn("redis pubsub channel closed")
				return
			}
			r.handleRouteMessage([]byte(msg.Payload))
		}
	}
}

// handleRouteMessage processes a route message.
func (r *RouterService) handleRouteMessage(data []byte) {
	var routeMsg model.RouteMessage
	if err := json.Unmarshal(data, &routeMsg); err != nil {
		logger.Error("unmarshal route message failed", zap.Error(err))
		return
	}

	// Ignore messages sent by this node.
	if routeMsg.FromNode == r.nodeID {
		return
	}

	// Construct the push message.
	msg := model.NewMessageWithID(
		routeMsg.Event,
		routeMsg.EventID,
		routeMsg.Timestamp,
		json.RawMessage(routeMsg.Payload),
	)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Error("marshal message failed", zap.Error(err))
		return
	}

	switch routeMsg.TargetType {
	case "agent":
		// Push to the agent.
		agentID := int64(0)
		fmt.Sscanf(routeMsg.TargetID, "%d", &agentID)
		conns := r.connManager.GetAgentConnections(routeMsg.TenantID, agentID)
		for _, conn := range conns {
			if conn.IsConnected() {
				conn.Send(msgBytes)
			}
		}

	case "topic":
		// Push to topic subscribers.
		conns := r.subService.GetTopicSubscribers(routeMsg.TargetID)
		for _, conn := range conns {
			if conn.IsConnected() {
				conn.Send(msgBytes)
			}
		}

	case "all":
		// Broadcast through the broadcast topic.
		topic := r.subService.BroadcastTopic()
		conns := r.subService.GetTopicSubscribers(topic)
		for _, conn := range conns {
			if conn.IsConnected() {
				conn.Send(msgBytes)
			}
		}
	}
}

// ===== Heartbeat renewal =====

// heartbeatLoop runs the route heartbeat renewal loop.
// It periodically renews route information for all agents on the local node.
func (r *RouterService) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.renewAllRoutes()
		}
	}
}

// renewAllRoutes renews all routes on the local node.
func (r *RouterService) renewAllRoutes() {
	if r.redisClient == nil {
		return
	}

	// Get all agent connections on the local node.
	agentConns := r.connManager.GetAllAgentConnIDs()
	if len(agentConns) == 0 {
		return
	}

	ctx := context.Background()
	pipe := r.redisClient.Pipeline()

	count := 0
	for agentID := range agentConns {
		key := r.agentRouteKey(agentID)
		pipe.Expire(ctx, key, routeTTL)
		count++
	}

	// Execute the pipeline.
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Warn("renew routes pipeline failed", zap.Error(err))
		return
	}

	logger.Debug("renew routes", zap.Int("count", count))
}
