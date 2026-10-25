// Package consumer 提供消息消费者
// 消费来自 NATS 或 Redis Pub/Sub 的推送消息，转发给本地 WebSocket 连接
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"vocara/pkg/config"
	"vocara/pkg/logger"
	"vocara/services/push/internal/model"
	"vocara/services/push/internal/service"
)

// NATSConsumer NATS 消息消费者
// 从 NATS 订阅推送相关主题，收到消息后推送给本地 WebSocket 连接
//
// 订阅的主题：
// - vocara.ws.push.agent.{agentId}   推送给指定座席
// - vocara.ws.push.tenant.{tenantId} 推送给指定租户
// - vocara.ws.push.topic.{topic}     推送到指定主题
// - vocara.ws.push.broadcast         全局广播
//
// 注意：此消费者与 RouterService 中的消费者功能不同
// RouterService 的消费者用于多节点之间的消息路由
// 此消费者用于消费业务服务（如座席服务、通话服务）发来的推送事件
type NATSConsumer struct {
	connManager *service.ConnectionManager
	subService  *service.SubscriptionService
	pushService *service.PushService
	natsConfig  config.NATSConfig

	conn *nats.Conn

	// 订阅列表（用于优雅关闭）
	subs []*nats.Subscription

	// 停止相关
	stopCh chan struct{}
	once   sync.Once
}

// NewNATSConsumer 创建 NATS 消费者
func NewNATSConsumer(
	connManager *service.ConnectionManager,
	subService *service.SubscriptionService,
	pushService *service.PushService,
	natsConfig config.NATSConfig,
) *NATSConsumer {
	return &NATSConsumer{
		connManager: connManager,
		subService:  subService,
		pushService: pushService,
		natsConfig:  natsConfig,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动 NATS 消费者
func (n *NATSConsumer) Start(ctx context.Context) error {
	// 连接 NATS
	if err := n.connect(); err != nil {
		return fmt.Errorf("connect nats failed: %w", err)
	}

	// 订阅各主题
	if err := n.subscribeAll(); err != nil {
		// 关闭连接
		n.conn.Close()
		return fmt.Errorf("subscribe nats topics failed: %w", err)
	}

	logger.Info("nats consumer started",
		zap.Strings("urls", n.natsConfig.URLList()),
	)

	// 监听停止信号
	go func() {
		select {
		case <-ctx.Done():
		case <-n.stopCh:
		}
		n.stop()
	}()

	return nil
}

// Stop 停止 NATS 消费者
func (n *NATSConsumer) Stop() {
	n.once.Do(func() {
		close(n.stopCh)
	})
}

// stop 实际停止逻辑
func (n *NATSConsumer) stop() {
	// 取消所有订阅
	for _, sub := range n.subs {
		_ = sub.Unsubscribe()
	}
	n.subs = nil

	// 关闭连接
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}

	logger.Info("nats consumer stopped")
}

// connect 连接 NATS
func (n *NATSConsumer) connect() error {
	urls := n.natsConfig.URLs
	if urls == "" {
		urls = nats.DefaultURL
	}

	opts := []nats.Option{
		nats.Name("vocara-push-consumer"),
		nats.MaxReconnects(n.natsConfig.MaxReconnect),
		nats.ReconnectWait(time.Duration(n.natsConfig.ReconnectWait) * time.Second),
		nats.Timeout(time.Duration(n.natsConfig.Timeout) * time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			logger.Warn("nats disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			logger.Info("nats reconnected", zap.String("url", conn.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			logger.Info("nats connection closed")
		}),
	}

	// 用户认证
	if n.natsConfig.User != "" {
		opts = append(opts, nats.UserInfo(n.natsConfig.User, n.natsConfig.Password))
	}
	if n.natsConfig.Token != "" {
		opts = append(opts, nats.Token(n.natsConfig.Token))
	}

	var err error
	n.conn, err = nats.Connect(urls, opts...)
	if err != nil {
		return fmt.Errorf("nats connect failed: %w", err)
	}

	return nil
}

// subscribeAll 订阅所有主题
func (n *NATSConsumer) subscribeAll() error {
	// 座席推送主题（使用通配符）
	sub, err := n.conn.Subscribe("vocara.ws.push.agent.*", n.handleAgentPush)
	if err != nil {
		return fmt.Errorf("subscribe agent push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	// 租户推送主题
	sub, err = n.conn.Subscribe("vocara.ws.push.tenant.*", n.handleTenantPush)
	if err != nil {
		return fmt.Errorf("subscribe tenant push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	// 主题推送
	sub, err = n.conn.Subscribe("vocara.ws.push.topic.>", n.handleTopicPush)
	if err != nil {
		return fmt.Errorf("subscribe topic push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	// 广播
	sub, err = n.conn.Subscribe("vocara.ws.push.broadcast", n.handleBroadcastPush)
	if err != nil {
		return fmt.Errorf("subscribe broadcast push failed: %w", err)
	}
	n.subs = append(n.subs, sub)

	return nil
}

// ===== 消息处理 =====

// handleAgentPush 处理座席推送消息
func (n *NATSConsumer) handleAgentPush(msg *nats.Msg) {
	// 从主题中提取 agentId
	agentIDStr := extractLastPart(msg.Subject)
	agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
	if err != nil || agentID <= 0 {
		logger.Warn("invalid agent push subject",
			zap.String("subject", msg.Subject),
			zap.String("agent_id", agentIDStr),
		)
		return
	}

	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal agent push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received agent push from nats",
		zap.Int64("agent_id", agentID),
		zap.String("event", pushMsg.Event),
	)

	// 通过推送服务推送（会自动处理本地+跨节点）
	// 从消息 data 中尝试提取 tenantId
	tenantID := extractTenantIDFromData(pushMsg.Data)

	_, err = n.pushService.PushToAgent(tenantID, agentID, pushMsg.Event, pushMsg.Data)
	if err != nil {
		logger.Error("push to agent failed",
			zap.Int64("agent_id", agentID),
			zap.Error(err),
		)
	}
}

// handleTenantPush 处理租户推送消息
func (n *NATSConsumer) handleTenantPush(msg *nats.Msg) {
	tenantIDStr := extractLastPart(msg.Subject)
	tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64)
	if err != nil || tenantID <= 0 {
		logger.Warn("invalid tenant push subject",
			zap.String("subject", msg.Subject),
			zap.String("tenant_id", tenantIDStr),
		)
		return
	}

	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal tenant push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received tenant push from nats",
		zap.Int64("tenant_id", tenantID),
		zap.String("event", pushMsg.Event),
	)

	// 通过推送服务推送给租户
	_, err = n.pushService.PushToTenant(tenantID, pushMsg.Event, pushMsg.Data, nil)
	if err != nil {
		logger.Error("push to tenant failed",
			zap.Int64("tenant_id", tenantID),
			zap.Error(err),
		)
	}
}

// handleTopicPush 处理主题推送消息
func (n *NATSConsumer) handleTopicPush(msg *nats.Msg) {
	// 主题可能包含多级路径，提取 vocara.ws.push.topic. 之后的部分
	topic := strings.TrimPrefix(msg.Subject, "vocara.ws.push.topic.")
	if topic == "" {
		logger.Warn("invalid topic push subject", zap.String("subject", msg.Subject))
		return
	}

	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal topic push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received topic push from nats",
		zap.String("topic", topic),
		zap.String("event", pushMsg.Event),
	)

	_, err := n.pushService.PushToTopic(topic, pushMsg.Event, pushMsg.Data)
	if err != nil {
		logger.Error("push to topic failed",
			zap.String("topic", topic),
			zap.Error(err),
		)
	}
}

// handleBroadcastPush 处理广播消息
func (n *NATSConsumer) handleBroadcastPush(msg *nats.Msg) {
	var pushMsg model.Message
	if err := json.Unmarshal(msg.Data, &pushMsg); err != nil {
		logger.Error("unmarshal broadcast push message failed",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return
	}

	logger.Debug("received broadcast push from nats",
		zap.String("event", pushMsg.Event),
	)

	_, err := n.pushService.PushToAll(pushMsg.Event, pushMsg.Data)
	if err != nil {
		logger.Error("push to all failed", zap.Error(err))
	}
}

// ===== 辅助方法 =====

// extractLastPart 提取主题的最后一部分
func extractLastPart(subject string) string {
	idx := strings.LastIndex(subject, ".")
	if idx < 0 || idx == len(subject)-1 {
		return ""
	}
	return subject[idx+1:]
}

// extractTenantIDFromData 从消息数据中提取 tenantId
func extractTenantIDFromData(data interface{}) int64 {
	if data == nil {
		return 0
	}

	// 尝试从 map 中获取
	if m, ok := data.(map[string]interface{}); ok {
		if tid, exists := m["tenant_id"]; exists {
			switch v := tid.(type) {
			case float64:
				return int64(v)
			case int64:
				return v
			case int:
				return int64(v)
			case string:
				if id, err := strconv.ParseInt(v, 10, 64); err == nil {
					return id
				}
			}
		}
	}

	return 0
}
