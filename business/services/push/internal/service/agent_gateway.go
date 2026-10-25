// Package service 提供推送服务的业务逻辑层。
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

// tenantStr 租户 ID 统一转字符串（conn 内为 int64）。
func tenantStr(tenantID int64) string { return strconv.FormatInt(tenantID, 10) }

// AgentGateway 座席命令网关：WS 连接生命周期 → agentcenter 内部接口。
// 职责：连接即签入（幂等恢复）、新连接踢旧连接、30s 保活续期、
// ready/busy/signout 命令透传与回执。状态机全在 agentcenter，本网关无状态。
type AgentGateway struct {
	client    AgentCenterAPI
	connMgr   *ConnectionManager
	keepalive time.Duration // 保活间隔，默认 30s
	maxMiss   int           // 续期连续失败上限，默认 3

	mu        sync.Mutex
	stopChans map[string]chan struct{} // connID → 保活停止信号
}

// NewAgentGateway 创建座席命令网关。client 为 nil 时网关禁用（未配置 agentcenter_base_url）。
func NewAgentGateway(client AgentCenterAPI, connMgr *ConnectionManager) *AgentGateway {
	return &AgentGateway{
		client:    client,
		connMgr:   connMgr,
		keepalive: 30 * time.Second,
		maxMiss:   3,
		stopChans: make(map[string]chan struct{}),
	}
}

// Enabled 网关是否可用。
func (g *AgentGateway) Enabled() bool { return g != nil && g.client != nil }

// OnConnect 连接即签入：幂等建立/恢复会话 → 踢旧连接 → 启动保活协程。
// 先建会话成功再踢旧：建会话失败时旧连接保持不动，前端重连即可幂等恢复。
// 返回错误表示签入失败（调用方负责告知前端并关闭连接）。
func (g *AgentGateway) OnConnect(ctx context.Context, conn *model.Connection, dn string) (*AgentSessionInfo, error) {
	if !g.Enabled() {
		return nil, fmt.Errorf("agent gateway disabled")
	}
	tenantID := conn.TenantID()
	agentID := conn.UserID() // 座席连接：userID 即工号（ws_handler 已换算）

	// 1. 幂等建立/恢复会话
	info, err := g.client.EstablishSession(ctx, tenantStr(tenantID), strconv.FormatInt(agentID, 10), dn)
	if err != nil {
		return nil, err
	}

	// 2. 新连接踢旧连接（仅 kick 仍存活的；断线重连场景旧连接已死，自然跳过）
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

	// 3. 启动保活协程
	g.startKeepalive(conn)
	return info, nil
}

// OnDisconnect 连接断开：仅停保活协程（D3：不主动变更座席状态，会话 120s 自然过期）。
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

// startKeepalive 每 30s 调 session/refresh；连续失败 maxMiss 次断开连接。
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

// doStatus 状态命令透传 + ack 回执。
func (g *AgentGateway) doStatus(conn *model.Connection, action string, status int, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	err := g.client.UpdateStatus(ctx, tenantStr(conn.TenantID()), strconv.FormatInt(conn.UserID(), 10), status, reason)
	g.sendAck(conn, action, err)
}

// HandleReady 示闲。agentStatusReady 与 agentcenter model.StatusReady 数值对齐。
func (g *AgentGateway) HandleReady(conn *model.Connection) {
	g.doStatus(conn, "ready", agentStatusReady, "")
}

// HandleBusy 示忙（reason 可选）。
func (g *AgentGateway) HandleBusy(conn *model.Connection, reason string) {
	g.doStatus(conn, "busy", agentStatusBusy, reason)
}

// HandleSignout 签出：清会话 → ack → 关闭连接。
func (g *AgentGateway) HandleSignout(conn *model.Connection) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	err := g.client.DeleteSession(ctx, tenantStr(conn.TenantID()), strconv.FormatInt(conn.UserID(), 10))
	if err != nil {
		g.sendAck(conn, "signout", err)
		return
	}
	// 成功：ack 帧经 sendAndClose 发出，留 flush 窗口后关闭连接
	g.sendAndClose(conn, ackMessage("signout", nil))
}

// ackMessage 构造命令回执帧。
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

// sendAck 命令回执。
func (g *AgentGateway) sendAck(conn *model.Connection, action string, err error) {
	g.sendJSON(conn, ackMessage(action, err))
}

// sendJSON 向连接发送 JSON 消息（经 conn.Send，写协程消费）。
func (g *AgentGateway) sendJSON(conn *model.Connection, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	conn.Send(data)
}

// sendAndClose 先发送下行帧，留出 flush 窗口后再关闭连接，
// 避免 Send 异步缓冲后立即 Close 导致写协程先命中关闭信号丢帧。
func (g *AgentGateway) sendAndClose(conn *model.Connection, v interface{}) {
	g.sendJSON(conn, v)
	time.Sleep(100 * time.Millisecond)
	conn.Close()
}

// agentcenter 状态枚举（与 services/agentcenter/internal/model/status.go 数值对齐，
// push 不 import agentcenter 包，避免跨服务依赖）
const (
	agentStatusReady = 3 // StatusReady
	agentStatusBusy  = 9 // StatusBusy
)
