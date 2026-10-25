package model

import "encoding/json"

// PushRequest 推送请求（HTTP 接口使用）
// 其他服务通过 HTTP 调用推送服务时使用此结构

// PushToAgentRequest 推送给指定座席的请求
type PushToAgentRequest struct {
	TenantID int64       `json:"tenant_id" binding:"required"` // 租户ID
	AgentID  int64       `json:"agent_id" binding:"required"`  // 座席ID
	Event    string      `json:"event" binding:"required"`     // 事件类型
	Data     interface{} `json:"data"`                          // 事件数据
}

// PushToTenantRequest 推送给整个租户的请求
type PushToTenantRequest struct {
	TenantID int64       `json:"tenant_id" binding:"required"` // 租户ID
	Event    string      `json:"event" binding:"required"`     // 事件类型
	Data     interface{} `json:"data"`                          // 事件数据
	// 可选：推送给特定用户类型
	UserTypes []string `json:"user_types,omitempty"` // 用户类型过滤
}

// PushToAllRequest 广播给所有人的请求
type PushToAllRequest struct {
	Event string      `json:"event" binding:"required"` // 事件类型
	Data  interface{} `json:"data"`                      // 事件数据
}

// PushToTopicRequest 推送到指定主题的请求
type PushToTopicRequest struct {
	Topic string      `json:"topic" binding:"required"` // 主题
	Event string      `json:"event" binding:"required"` // 事件类型
	Data  interface{} `json:"data"`                      // 事件数据
}

// PushBatchRequest 批量推送请求
type PushBatchRequest struct {
	Items []BatchPushItem `json:"items" binding:"required"` // 推送项列表
}

// BatchPushItem 批量推送项
type BatchPushItem struct {
	TargetType string      `json:"target_type"` // 目标类型: agent/tenant/topic/all
	TargetID   string      `json:"target_id"`   // 目标ID
	TenantID   int64       `json:"tenant_id"`   // 租户ID（当 target_type 为 agent 时）
	Event      string      `json:"event"`       // 事件类型
	Data       interface{} `json:"data"`        // 事件数据
}

// PushResponse 推送响应
type PushResponse struct {
	Success bool   `json:"success"` // 是否成功
	Count   int    `json:"count"`   // 推送的连接数
	Message string `json:"message"` // 消息
}

// ===== 多节点路由消息 =====

// RouteMessage 跨节点路由消息
// 当节点需要给不在本节点的座席发消息时，通过 NATS/Redis PubSub 转发
type RouteMessage struct {
	TargetType string          `json:"target_type"` // 目标类型: agent/tenant/topic
	TargetID   string          `json:"target_id"`   // 目标标识
	TenantID   int64           `json:"tenant_id"`   // 租户ID
	Event      string          `json:"event"`       // 事件类型
	EventID    string          `json:"event_id"`    // 事件ID
	Timestamp  int64           `json:"timestamp"`   // 时间戳
	Payload    json.RawMessage `json:"payload"`     // 原始消息数据
	FromNode   string          `json:"from_node"`   // 来源节点ID
}

// ===== 连接路由信息（存储在 Redis 中）=====

// RouteInfo 连接路由信息
type RouteInfo struct {
	AgentID    int64  `json:"agent_id"`    // 座席ID
	TenantID   int64  `json:"tenant_id"`   // 租户ID
	NodeID     string `json:"node_id"`     // 节点ID
	NodeAddr   string `json:"node_addr"`   // 节点地址
	ConnID     string `json:"conn_id"`     // 连接ID
	ServerTime int64  `json:"server_time"` // 注册时间
}

// ===== 统计信息 =====

// Stats 推送服务统计信息
type Stats struct {
	TotalConns     int            `json:"total_conns"`      // 总连接数
	AgentConns     int            `json:"agent_conns"`      // 座席连接数
	AdminConns     int            `json:"admin_conns"`      // 管理员连接数
	MonitorConns   int            `json:"monitor_conns"`    // 监控连接数
	TenantCount    int            `json:"tenant_count"`     // 租户数
	TotalMessages  int64          `json:"total_messages"`   // 总消息数
	FailedMessages int64          `json:"failed_messages"`  // 失败消息数
	Topics         map[string]int `json:"topics"`           // 各主题订阅数
	Uptime         int64          `json:"uptime"`           // 运行时间（秒）
}
