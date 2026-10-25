// Package handler 提供 HTTP/WebSocket 请求处理
// 包括 WebSocket 连接升级、消息处理、HTTP 推送接口等
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hertz-contrib/websocket"
	"go.uber.org/zap"

	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/pkg/response"
	"vocara/services/push/internal/model"
	"vocara/services/push/internal/service"
)

const (
	// pongWait WebSocket pong 等待时间
	pongWait = 60 * time.Second
)

// WebSocketHandler WebSocket 处理器
// 负责 WebSocket 连接的建立、消息收发、心跳检测等
type WebSocketHandler struct {
	connManager   *service.ConnectionManager
	subService    *service.SubscriptionService
	pushService   *service.PushService
	routerService *service.RouterService

	// agentGateway 座席命令网关（nil 表示未启用）
	agentGateway *service.AgentGateway

	// Hertz WebSocket 升级器
	upgrader websocket.HertzUpgrader

	// JWT 密钥
	jwtSecret string
}

// NewWebSocketHandler 创建 WebSocket 处理器
func NewWebSocketHandler(
	connManager *service.ConnectionManager,
	subService *service.SubscriptionService,
	pushService *service.PushService,
	routerService *service.RouterService,
	jwtSecret string,
	agentGateway *service.AgentGateway,
) *WebSocketHandler {
	return &WebSocketHandler{
		connManager:   connManager,
		subService:    subService,
		pushService:   pushService,
		routerService: routerService,
		agentGateway:  agentGateway,
		jwtSecret:     jwtSecret,
		upgrader: websocket.HertzUpgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// 允许所有来源（生产环境应配置正确的 Origin 检查）
			CheckOrigin: func(ctx *app.RequestContext) bool {
				return true
			},
		},
	}
}

// ===== 连接入口 =====

// AgentWS 座席端 WebSocket 连接入口
// GET /ws/agent?token=xxx
func (h *WebSocketHandler) AgentWS(c context.Context, ctx *app.RequestContext) {
	h.handleWSConnection(c, ctx, model.UserTypeAgent)
}

// AdminWS 管理端 WebSocket 连接入口
// GET /ws/admin?token=xxx
func (h *WebSocketHandler) AdminWS(c context.Context, ctx *app.RequestContext) {
	h.handleWSConnection(c, ctx, model.UserTypeAdmin)
}

// MonitorWS 监控大屏 WebSocket 连接入口
// GET /ws/monitor?token=xxx
func (h *WebSocketHandler) MonitorWS(c context.Context, ctx *app.RequestContext) {
	h.handleWSConnection(c, ctx, model.UserTypeMonitor)
}

// handleWSConnection 处理 WebSocket 连接的通用逻辑
func (h *WebSocketHandler) handleWSConnection(
	c context.Context,
	ctx *app.RequestContext,
	userType string,
) {
	// 1. 鉴权验证
	userID, tenantID, username, agentNum, err := h.authenticate(ctx)
	if err != nil {
		logger.Warn("websocket auth failed",
			zap.String("user_type", userType),
			zap.Error(err),
		)
		response.Fail(ctx, errors.ErrTokenInvalid)
		return
	}

	// 座席类型：使用 agent_num（座席工号）作为连接的 userID，
	// 与通话事件中的 agent_id（工号）对齐，确保 call.ringing 等事件能正确路由到座席连接
	connUserID := userID
	if userType == model.UserTypeAgent && agentNum != "" {
		if id, err := strconv.ParseInt(agentNum, 10, 64); err == nil && id > 0 {
			connUserID = id
		} else {
			// 工号可能不是纯数字（如 "A1001"），此时用 hash 方式不现实，
			// 连接管理器的 userID 是 int64，需要保留字符串兼容性。
			// 这里先 fallback 到原 userID，后续如果工号非纯数字需要重构连接管理器为 string 键。
			logger.Warn("agent_num is not numeric, fallback to user_id",
				zap.String("agent_num", agentNum),
			)
		}
	}

	// 2. 升级 WebSocket 连接（使用 Hertz WebSocket 适配器）
	err = h.upgrader.Upgrade(ctx, func(wsConn *websocket.Conn) {
		// 3. 创建连接对象
		connID := generateConnID()
		conn := model.NewConnection(wsConn, connID, userType, connUserID, tenantID, username)

		// 4. 注册连接
		h.connManager.AddConnection(conn)

		// 5. 自动订阅默认主题
		h.autoSubscribe(conn)

		// 6. 注册路由（座席类型）
		// 注意：必须使用 connUserID（座席工号），与通话事件中的 agent_id 对齐，
		// 否则 call.ringing 等事件按 agent_id=1001 寻址会找不到路由。
		if userType == model.UserTypeAgent && h.routerService != nil && h.routerService.IsEnabled() {
			if err := h.routerService.RegisterRoute(tenantID, connUserID, connID); err != nil {
				logger.Warn("register route failed",
					zap.String("conn_id", connID),
					zap.Int64("agent_id", connUserID),
					zap.Error(err),
				)
			}
		}

		// 7. 启动写协程（先于网关签入与欢迎消息，确保下行缓冲帧有人消费）
		go h.connManager.StartWritePump(conn)

		// 7.5 座席命令网关：dn 参数存在时执行"连接即签入"
		// 失败时不提前 return：发 error 帧 → 留 flush 窗口 → 关闭连接，
		// 让流程自然进入 readPump（读已关闭连接立即报错退出）→ cleanupConnection 走正常清理
		sessionFailed := false
		if userType == model.UserTypeAgent && h.agentGateway != nil && h.agentGateway.Enabled() {
			dn := ctx.Query("dn")
			if dn != "" {
				info, err := h.agentGateway.OnConnect(c, conn, dn)
				if err != nil {
					logger.Warn("agent session establish failed",
						zap.String("conn_id", conn.ConnID()),
						zap.String("dn", dn),
						zap.Error(err),
					)
					h.sendJSON(conn, map[string]interface{}{
						"type":   "session",
						"action": "error",
						"data":   map[string]interface{}{"message": "座席签入失败: " + err.Error()},
					})
					// 给写协程 flush 窗口后再关闭连接
					time.Sleep(100 * time.Millisecond)
					conn.Close()
					sessionFailed = true
				} else {
					h.sendJSON(conn, map[string]interface{}{
						"type":   "session",
						"action": "ready",
						"data":   info,
					})
				}
			}
		}

		// 8. 发送连接成功消息（签入失败的连接已关闭，不再发送）
		if !sessionFailed {
			h.sendWelcome(conn)
		}

		logger.Info("websocket connection established",
			zap.String("conn_id", connID),
			zap.String("user_type", userType),
			zap.Int64("user_id", userID),
			zap.Int64("tenant_id", tenantID),
		)

		// 9. 启动读协程（阻塞直到连接关闭；签入失败的连接已关闭，ReadMessage 立即报错退出）
		h.readPump(conn)

		// 10. 连接关闭后的清理
		h.cleanupConnection(conn)
	})

	if err != nil {
		logger.Error("websocket upgrade failed",
			zap.String("user_type", userType),
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		response.FailWithMsg(ctx, errors.CodeSystemError, "WebSocket 升级失败")
		return
	}
}

// ===== 鉴权 =====

// authenticate 从请求中验证用户身份
// 支持从 query 参数和 Authorization header 获取 token
// 返回值：userID, tenantID, username, agentNum, err
// agentNum 仅座席角色有效，用于通话事件按工号路由
func (h *WebSocketHandler) authenticate(ctx *app.RequestContext) (userID int64, tenantID int64, username string, agentNum string, err error) {
	// 从 query 参数获取 token（WebSocket 握手时不能设置自定义 header）
	token := ctx.Query("token")
	if token == "" {
		// 从 Authorization header 获取
		token = string(ctx.GetHeader("Authorization"))
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		return 0, 0, "", "", fmt.Errorf("token missing")
	}

	// 解析 JWT token
	claims, err := parseToken(token, h.jwtSecret)
	if err != nil {
		return 0, 0, "", "", err
	}

	// 从 claims 中提取用户信息
	userID, err = parseUserID(claims)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("invalid user_id in token: %w", err)
	}

	username, _ = claims["username"].(string)

	// 座席工号（座席角色 token 中携带，用于 push 服务按工号路由通话事件）
	agentNum, _ = claims["agent_num"].(string)

	// 从 claims 中获取租户ID（如果有）
	tenantID = parseTenantID(claims)

	if userID <= 0 {
		return 0, 0, "", "", fmt.Errorf("invalid user id")
	}

	return userID, tenantID, username, agentNum, nil
}

// parseUserID 从 claims 中解析 userID
func parseUserID(claims jwt.MapClaims) (int64, error) {
	raw, ok := claims["user_id"]
	if !ok {
		return 0, fmt.Errorf("user_id not found")
	}

	switch v := raw.(type) {
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported user_id type: %T", raw)
	}
}

// parseTenantID 从 claims 中解析 tenantID
func parseTenantID(claims jwt.MapClaims) int64 {
	raw, ok := claims["tenant_id"]
	if !ok {
		return 0
	}

	switch v := raw.(type) {
	case float64:
		return int64(v)
	case string:
		id, _ := strconv.ParseInt(v, 10, 64)
		return id
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

// parseToken 解析 JWT token
func parseToken(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims type")
	}

	return claims, nil
}

// ===== 自动订阅 =====

// autoSubscribe 连接建立后自动订阅默认主题
func (h *WebSocketHandler) autoSubscribe(conn *model.Connection) {
	switch conn.UserType() {
	case model.UserTypeAgent:
		h.subService.AutoSubscribeForAgent(conn)
	case model.UserTypeAdmin:
		h.subService.AutoSubscribeForAdmin(conn)
	case model.UserTypeMonitor:
		h.subService.AutoSubscribeForMonitor(conn)
	}
}

// sendWelcome 发送欢迎消息
func (h *WebSocketHandler) sendWelcome(conn *model.Connection) {
	welcomeData := map[string]interface{}{
		"conn_id":       conn.ConnID(),
		"user_type":     conn.UserType(),
		"user_id":       conn.UserID(),
		"tenant_id":     conn.TenantID(),
		"server_time":   time.Now().UnixMilli(),
		"subscriptions": conn.GetSubscriptions(),
	}
	msg := model.NewMessage("connected", welcomeData)
	msgBytes, _ := json.Marshal(msg)
	conn.Send(msgBytes)
}

// ===== 读协程 =====

// readPump 读取客户端消息
// 阻塞直到连接关闭
func (h *WebSocketHandler) readPump(conn *model.Connection) {
	defer func() {
		conn.Close()
	}()

	// 设置读超时和 pong 处理
	wsConn := conn.WSConn()
	if wsConn == nil {
		return
	}

	// 设置初始读超时
	_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))

	// 设置 pong 处理
	wsConn.SetPongHandler(func(appData string) error {
		_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
		conn.UpdatePing()
		return nil
	})

	for {
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Warn("websocket read error",
					zap.String("conn_id", conn.ConnID()),
					zap.Error(err),
				)
			}
			break
		}

		// 处理客户端消息
		h.handleClientMessage(conn, message)
	}
}

// handleClientMessage 处理客户端发送的消息
func (h *WebSocketHandler) handleClientMessage(conn *model.Connection, data []byte) {
	var clientMsg model.ClientMessage
	if err := json.Unmarshal(data, &clientMsg); err != nil {
		logger.Warn("invalid client message",
			zap.String("conn_id", conn.ConnID()),
			zap.Error(err),
		)
		h.pushService.PushError(conn, errors.CodeParamError, "消息格式错误", err.Error())
		return
	}

	switch clientMsg.Action {
	case model.ClientMsgSubscribe:
		h.handleSubscribe(conn, clientMsg.Data)

	case model.ClientMsgUnsubscribe:
		h.handleUnsubscribe(conn, clientMsg.Data)

	case model.ClientMsgHeartbeat, model.ClientMsgPong:
		// 心跳消息，更新心跳时间
		conn.UpdatePing()
		// 响应心跳
		h.pushService.SendHeartbeat(conn)

	case model.ClientMsgReady:
		if h.agentGateway != nil && h.agentGateway.Enabled() {
			h.agentGateway.HandleReady(conn)
		}
	case model.ClientMsgBusy:
		if h.agentGateway != nil && h.agentGateway.Enabled() {
			reason, _ := clientMsg.Data["reason"].(string)
			h.agentGateway.HandleBusy(conn, reason)
		}
	case model.ClientMsgSignout:
		if h.agentGateway != nil && h.agentGateway.Enabled() {
			h.agentGateway.HandleSignout(conn)
		}

	default:
		logger.Warn("unknown client action",
			zap.String("conn_id", conn.ConnID()),
			zap.String("action", clientMsg.Action),
		)
		h.pushService.PushError(conn, errors.CodeParamInvalid, "未知的操作类型", clientMsg.Action)
	}
}

// handleSubscribe 处理订阅请求
func (h *WebSocketHandler) handleSubscribe(conn *model.Connection, data map[string]interface{}) {
	// 从 data 中提取 topics
	topics := extractStringArray(data, "topics")
	if len(topics) == 0 {
		h.pushService.PushError(conn, errors.CodeParamMissing, "缺少订阅主题", "")
		return
	}

	// 校验主题合法性
	validTopics := make([]string, 0, len(topics))
	for _, topic := range topics {
		if h.subService.IsValidTopic(topic) {
			validTopics = append(validTopics, topic)
		} else {
			logger.Warn("invalid topic for subscription",
				zap.String("conn_id", conn.ConnID()),
				zap.String("topic", topic),
			)
		}
	}

	if len(validTopics) == 0 {
		h.pushService.PushError(conn, errors.CodeParamInvalid, "没有合法的订阅主题", "")
		return
	}

	h.subService.Subscribe(conn, validTopics)

	logger.Info("subscribe success",
		zap.String("conn_id", conn.ConnID()),
		zap.Strings("topics", validTopics),
	)
}

// handleUnsubscribe 处理取消订阅请求
func (h *WebSocketHandler) handleUnsubscribe(conn *model.Connection, data map[string]interface{}) {
	topics := extractStringArray(data, "topics")
	if len(topics) == 0 {
		h.pushService.PushError(conn, errors.CodeParamMissing, "缺少取消订阅主题", "")
		return
	}

	h.subService.Unsubscribe(conn, topics)

	logger.Info("unsubscribe success",
		zap.String("conn_id", conn.ConnID()),
		zap.Strings("topics", topics),
	)
}

// ===== 清理 =====

// cleanupConnection 连接关闭后的清理工作
func (h *WebSocketHandler) cleanupConnection(conn *model.Connection) {
	// 1. 取消所有订阅
	h.subService.UnsubscribeAll(conn)

	// 2. 注销路由（座席类型）
	if conn.UserType() == model.UserTypeAgent && h.routerService != nil && h.routerService.IsEnabled() {
		if err := h.routerService.UnregisterRoute(conn.UserID(), conn.ConnID()); err != nil {
			logger.Warn("unregister route failed",
				zap.String("conn_id", conn.ConnID()),
				zap.Int64("agent_id", conn.UserID()),
				zap.Error(err),
			)
		}
	}

	// 3. 从连接管理器移除
	h.connManager.RemoveConnection(conn)

	// 4. 通知网关停止保活协程（D3：不变更座席状态，会话自然过期）
	if conn.UserType() == model.UserTypeAgent && h.agentGateway != nil {
		h.agentGateway.OnDisconnect(conn)
	}

	logger.Info("websocket connection closed",
		zap.String("conn_id", conn.ConnID()),
		zap.String("user_type", conn.UserType()),
		zap.Int64("user_id", conn.UserID()),
		zap.Duration("duration", time.Since(conn.ConnectAt())),
	)
}

// sendJSON 向连接发送自定义 JSON 消息（网关 session/kick/ack 等下行消息）
func (h *WebSocketHandler) sendJSON(conn *model.Connection, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	conn.Send(data)
}

// ===== 工具函数 =====

// generateConnID 生成连接ID
func generateConnID() string {
	return fmt.Sprintf("conn_%d_%s", time.Now().UnixNano(), randomString(8))
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[int(now>>uint(i*5))%len(letters)]
	}
	return string(b)
}

// extractStringArray 从 map 中提取字符串数组
func extractStringArray(data map[string]interface{}, key string) []string {
	if data == nil {
		return nil
	}

	val, ok := data[key]
	if !ok {
		return nil
	}

	// 尝试断言为 []interface{}
	if arr, ok := val.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	// 尝试断言为 []string
	if arr, ok := val.([]string); ok {
		return arr
	}

	return nil
}

// ===== HTTP 健康检查 =====

// HealthCheck 健康检查
func (h *WebSocketHandler) HealthCheck(c context.Context, ctx *app.RequestContext) {
	stats := h.connManager.GetStats()
	response.OK(ctx, map[string]interface{}{
		"status": "ok",
		"stats":  stats,
	})
}

// GetStats 获取统计信息
func (h *WebSocketHandler) GetStats(c context.Context, ctx *app.RequestContext) {
	stats := h.connManager.GetStats()
	// 补充主题统计
	stats.Topics = h.subService.GetAllTopics()
	response.OK(ctx, stats)
}

// 确保 consts 包被引用
var _ = consts.StatusOK
