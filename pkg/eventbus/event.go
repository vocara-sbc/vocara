// Package eventbus 提供统一的事件总线封装
// 支持 NATS JetStream 和 Redis Pub/Sub 两种后端，NATS 优先，Redis 降级
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ===== 事件类型常量 =====
// 所有业务事件的类型定义，保持全局唯一

const (
	// ===== 座席相关事件 =====
	EventAgentStatusChange = "agent.status_change" // 座席状态变更
	EventAgentCheckIn      = "agent.checkin"       // 座席签入
	EventAgentCheckOut     = "agent.checkout"      // 座席签出

	// ===== 通话相关事件 =====
	EventCallRinging    = "call.ringing"    // 来电振铃
	EventCallAnswered   = "call.answered"   // 通话接通
	EventCallBridged    = "call.bridged"    // 通话桥接
	EventCallHeld       = "call.held"       // 通话保持
	EventCallUnheld     = "call.unheld"     // 通话恢复
	EventCallHangup     = "call.hangup"     // 通话结束
	EventCallTransferred = "call.transferred" // 通话转接

	// ===== 排队相关事件 =====
	EventQueueEnqueue    = "queue.enqueue"    // 通话入队
	EventQueueDequeue    = "queue.dequeue"    // 通话出队（分配成功）
	EventQueueAbandoned  = "queue.abandoned"  // 排队放弃
	EventQueueTimeout    = "queue.timeout"    // 排队超时
	EventQueuePosition   = "queue.position"   // 排队位置更新

	// ===== CDR 相关事件 =====
	EventCDRCreated = "cdr.created" // CDR 生成

	// ===== 系统相关事件 =====
	EventSystemNotice = "system.notice" // 系统通知

	// ===== 呼入路由事件 =====
	// admin 在呼入路由增/删/改/启停后发布，callctrl 订阅后立即刷新路由缓存（即建即用）
	EventInboundRouteChange = "inbound_route.change" // 呼入路由变更
)

// ===== Topic 命名 =====
// NATS / Redis 统一使用点分隔的主题命名
// 完整 topic = prefix + event_type
// 例如：vocara.event.agent.status_change

const (
	// TopicPrefix 事件主题前缀
	TopicPrefix = "vocara.event."
)

// GetTopic 根据事件类型获取完整主题
func GetTopic(eventType string) string {
	return TopicPrefix + eventType
}

// ===== 事件信封格式 =====

// Event 事件信封，所有业务事件都包装在此结构中
type Event struct {
	EventID   string      `json:"event_id"`   // 事件唯一ID
	EventType string      `json:"event_type"` // 事件类型
	Timestamp int64       `json:"timestamp"`  // 事件时间戳（毫秒）
	TenantID  uint64      `json:"tenant_id"`  // 租户ID
	Source    string      `json:"source"`     // 事件来源服务名
	Data      interface{} `json:"data"`       // 事件数据（不同事件类型结构不同）
}

// NewEvent 创建新事件
func NewEvent(eventType string, tenantID uint64, source string, data interface{}) *Event {
	return &Event{
		EventID:   "evt_" + uuid.New().String()[:16],
		EventType: eventType,
		Timestamp: time.Now().UnixMilli(),
		TenantID:  tenantID,
		Source:    source,
		Data:      data,
	}
}

// Topic 获取事件对应的主题
func (e *Event) Topic() string {
	return GetTopic(e.EventType)
}

// ToJSON 序列化为 JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ParseEvent 从 JSON 解析事件
func ParseEvent(data []byte) (*Event, error) {
	var evt Event
	if err := json.Unmarshal(data, &evt); err != nil {
		return nil, fmt.Errorf("parse event failed: %w", err)
	}
	return &evt, nil
}

// ===== 发布者接口 =====

// Publisher 事件发布者接口
type Publisher interface {
	// Publish 发布事件
	Publish(ctx context.Context, event *Event) error
	// Close 关闭连接
	Close() error
}

// ===== 订阅者接口 =====

// Handler 事件处理函数
type Handler func(ctx context.Context, event *Event) error

// Subscriber 事件订阅者接口
type Subscriber interface {
	// Subscribe 订阅指定事件类型
	Subscribe(ctx context.Context, eventType string, handler Handler) (string, error)
	// Unsubscribe 取消订阅
	Unsubscribe(ctx context.Context, subscriptionID string) error
	// Close 关闭连接
	Close() error
}

// ===== 事件总线接口 =====

// EventBus 事件总线接口（发布+订阅）
type EventBus interface {
	Publisher
	Subscriber
}
