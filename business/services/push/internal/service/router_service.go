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
	// Redis 路由 key 前缀
	redisRoutePrefix = "vocara:ws:route:"

	// Redis 路由过期时间（心跳续期）
	routeTTL = 120 * time.Second

	// NATS 节点主题
	natsNodePrefix = "vocara.ws.node."

	// 本节点订阅的 NATS 主题
	natsLocalTopic = "vocara.ws.local"
)

// RouterService 多节点路由服务
// 负责在多节点部署环境下，将消息路由到正确的节点
//
// 工作原理：
// 1. 座席建立 WebSocket 连接时，在 Redis 中注册路由信息（agentId -> nodeId）
// 2. 需要给座席发消息时，先查 Redis 路由表
// 3. 如果座席在本节点，直接推送
// 4. 如果座席在其他节点，通过 NATS/Redis PubSub 转发到目标节点
// 5. 目标节点收到消息后，推送给本地连接
//
// 支持两种后端：
// - nats: 使用 NATS 做消息路由（推荐，性能更好）
// - redis: 使用 Redis Pub/Sub（备选，部署简单）
type RouterService struct {
	// 配置
	nodeID   string // 本节点ID
	nodeAddr string // 本节点地址
	mode     string // 路由模式: nats / redis / none

	// Redis 客户端（用于路由表和 Pub/Sub）
	redisClient *pkgredis.Client

	// NATS 配置
	natsCfg *config.NATSConfig

	// NATS 连接
	natsConn *nats.Conn

	// 连接管理器
	connManager *ConnectionManager

	// 订阅服务
	subService *SubscriptionService

	// 是否启用
	enabled bool

	// 停止信号
	stopCh chan struct{}
	once   sync.Once
}

// NewRouterService 创建路由服务
func NewRouterService(
	nodeID, nodeAddr string,
	cfg *config.Config,
	connManager *ConnectionManager,
	subService *SubscriptionService,
) *RouterService {
	mode := "redis" // 默认使用 redis 模式

	// 如果配置了 NATS，优先使用 NATS
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

// Start 启动路由服务
func (r *RouterService) Start(ctx context.Context) error {
	// 获取 Redis 客户端
	r.redisClient = pkgredis.Get()
	if r.redisClient == nil {
		logger.Warn("redis client not available, router service disabled")
		return nil
	}

	// 初始化对应模式的连接
	switch r.mode {
	case "nats":
		if err := r.initNATS(); err != nil {
			logger.Warn("init nats failed, fallback to redis mode", zap.Error(err))
			r.mode = "redis"
			// 继续用 redis 模式
		}

	case "redis":
		// redis 模式直接使用 redis client
	}

	r.enabled = true

	// 启动节点订阅
	go r.startLocalConsumer(ctx)

	// 启动路由续期心跳
	go r.heartbeatLoop(ctx)

	logger.Info("router service started",
		zap.String("node_id", r.nodeID),
		zap.String("node_addr", r.nodeAddr),
		zap.String("mode", r.mode),
	)

	return nil
}

// Stop 停止路由服务
func (r *RouterService) Stop() {
	r.once.Do(func() {
		close(r.stopCh)

		if r.natsConn != nil {
			r.natsConn.Close()
		}

		logger.Info("router service stopped")
	})
}

// IsEnabled 是否已启用
func (r *RouterService) IsEnabled() bool {
	return r.enabled
}

// NodeID 获取节点ID
func (r *RouterService) NodeID() string {
	return r.nodeID
}

// ===== NATS 初始化 =====

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

// ===== 路由注册 =====

// RegisterRoute 注册座席连接路由
// 座席建立连接时调用，记录 agentId 在哪个节点
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

// UnregisterRoute 注销座席连接路由
// 座席断开连接时调用；传入断开连接的 connID 做比对，
// 仅当路由表中记录的 connID 与之一致时才删除，
// 避免同工号重复登录时旧连接的清理误删新连接刚注册的路由。
func (r *RouterService) UnregisterRoute(agentID int64, connID string) error {
	if !r.enabled || r.redisClient == nil {
		return nil
	}

	key := r.agentRouteKey(agentID)
	ctx := context.Background()

	// 只有当路由指向本节点且 connID 匹配时才删除，避免误删其他节点/新连接的路由
	routeInfo, err := r.GetRoute(agentID)
	if err != nil {
		// 读取失败，直接删除（可能过期了）
		_, _ = r.redisClient.Del(ctx, key)
		return nil
	}

	if routeInfo.NodeID != r.nodeID {
		return nil
	}

	// connID 不匹配：说明路由已被该座席的新连接覆盖，跳过删除
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

// GetRoute 获取座席路由信息
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

// RenewRoute 续期路由（心跳时调用）
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

	// 如果 key 不存在，重新注册
	if !ok {
		// 重新注册的逻辑在调用方处理
		return fmt.Errorf("route key not exists")
	}

	return nil
}

// agentRouteKey 生成座席路由 key
func (r *RouterService) agentRouteKey(agentID int64) string {
	return fmt.Sprintf("%s%d", redisRoutePrefix, agentID)
}

// ===== 消息路由 =====

// PushToAgent 跨节点推送给座席
func (r *RouterService) PushToAgent(tenantID, agentID int64, msg *model.Message) (int, error) {
	if !r.enabled {
		return 0, nil
	}

	// 先查路由
	routeInfo, err := r.GetRoute(agentID)
	if err != nil {
		return 0, err
	}

	// 如果座席在本节点，不处理（本地推送已处理）
	if routeInfo.NodeID == r.nodeID {
		return 0, nil
	}

	// 构造路由消息
	routeMsg := &model.RouteMessage{
		TargetType: "agent",
		TargetID:   fmt.Sprintf("%d", agentID),
		TenantID:   tenantID,
		Event:      msg.Event,
		EventID:    msg.EventID,
		Timestamp:  msg.Timestamp,
		FromNode:   r.nodeID,
	}

	// 序列化 payload
	payload, err := json.Marshal(msg.Data)
	if err != nil {
		return 0, fmt.Errorf("marshal payload failed: %w", err)
	}
	routeMsg.Payload = payload

	// 发送到目标节点
	err = r.publishToNode(routeInfo.NodeID, routeMsg)
	if err != nil {
		return 0, err
	}

	return 1, nil
}

// PushToTopic 跨节点推送到主题
func (r *RouterService) PushToTopic(topic string, msg *model.Message) (int, error) {
	if !r.enabled {
		return 0, nil
	}

	// 构造路由消息
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

	// 发布到所有节点的本地主题（每个节点自己过滤订阅）
	err = r.publishToAll(routeMsg)
	if err != nil {
		return 0, err
	}

	// 跨节点推送无法精确计数，返回 0（实际数量在各节点统计）
	return 0, nil
}

// PushToAll 跨节点广播
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

// ===== 发布消息 =====

// publishToNode 发布消息到指定节点
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

// publishToAll 发布消息到所有节点
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

// ===== 本地消费者 =====

// startLocalConsumer 启动本地消息消费者
// 接收其他节点转发过来的消息，推送给本地连接
func (r *RouterService) startLocalConsumer(ctx context.Context) {
	switch r.mode {
	case "nats":
		r.startNATSConsumer(ctx)
	case "redis":
		r.startRedisConsumer(ctx)
	}
}

// startNATSConsumer 启动 NATS 消费者
func (r *RouterService) startNATSConsumer(ctx context.Context) {
	if r.natsConn == nil {
		return
	}

	// 订阅本节点专属主题
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

	// 订阅全局广播主题
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

	// 等待停止信号
	select {
	case <-ctx.Done():
	case <-r.stopCh:
	}
}

// startRedisConsumer 启动 Redis Pub/Sub 消费者
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

// handleRouteMessage 处理路由消息
func (r *RouterService) handleRouteMessage(data []byte) {
	var routeMsg model.RouteMessage
	if err := json.Unmarshal(data, &routeMsg); err != nil {
		logger.Error("unmarshal route message failed", zap.Error(err))
		return
	}

	// 忽略自己发的消息
	if routeMsg.FromNode == r.nodeID {
		return
	}

	// 构造推送消息
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
		// 推送给座席
		agentID := int64(0)
		fmt.Sscanf(routeMsg.TargetID, "%d", &agentID)
		conns := r.connManager.GetAgentConnections(routeMsg.TenantID, agentID)
		for _, conn := range conns {
			if conn.IsConnected() {
				conn.Send(msgBytes)
			}
		}

	case "topic":
		// 推送给主题订阅者
		conns := r.subService.GetTopicSubscribers(routeMsg.TargetID)
		for _, conn := range conns {
			if conn.IsConnected() {
				conn.Send(msgBytes)
			}
		}

	case "all":
		// 广播（通过广播主题）
		topic := r.subService.BroadcastTopic()
		conns := r.subService.GetTopicSubscribers(topic)
		for _, conn := range conns {
			if conn.IsConnected() {
				conn.Send(msgBytes)
			}
		}
	}
}

// ===== 心跳续期 =====

// heartbeatLoop 路由心跳续期循环
// 定期续期所有本节点座席的路由信息
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

// renewAllRoutes 续期所有本节点的路由
func (r *RouterService) renewAllRoutes() {
	if r.redisClient == nil {
		return
	}

	// 获取所有本节点的座席连接
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

	// 执行管道
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Warn("renew routes pipeline failed", zap.Error(err))
		return
	}

	logger.Debug("renew routes", zap.Int("count", count))
}
