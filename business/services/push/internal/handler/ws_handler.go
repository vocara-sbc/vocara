// Package handler provides HTTP/WebSocket request handling,
// including WebSocket connection upgrades, message processing, and HTTP push APIs.
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
	// pongWait is the WebSocket pong wait time.
	pongWait = 60 * time.Second
)

// WebSocketHandler handles WebSocket requests.
// It establishes connections, exchanges messages, and checks heartbeats.
type WebSocketHandler struct {
	connManager   *service.ConnectionManager
	subService    *service.SubscriptionService
	pushService   *service.PushService
	routerService *service.RouterService

	// agentGateway is the agent command gateway; nil means it is disabled.
	agentGateway *service.AgentGateway

	// Hertz WebSocket upgrader.
	upgrader websocket.HertzUpgrader

	// JWT secret.
	jwtSecret string
}

// NewWebSocketHandler creates a WebSocket handler.
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
			// Allow all origins; production deployments should configure proper Origin checks.
			CheckOrigin: func(ctx *app.RequestContext) bool {
				return true
			},
		},
	}
}

// ===== Connection entry points =====

// AgentWS is the WebSocket connection entry point for agents.
// GET /ws/agent?token=xxx
func (h *WebSocketHandler) AgentWS(c context.Context, ctx *app.RequestContext) {
	h.handleWSConnection(c, ctx, model.UserTypeAgent)
}

// AdminWS is the WebSocket connection entry point for administrators.
// GET /ws/admin?token=xxx
func (h *WebSocketHandler) AdminWS(c context.Context, ctx *app.RequestContext) {
	h.handleWSConnection(c, ctx, model.UserTypeAdmin)
}

// MonitorWS is the WebSocket connection entry point for monitoring dashboards.
// GET /ws/monitor?token=xxx
func (h *WebSocketHandler) MonitorWS(c context.Context, ctx *app.RequestContext) {
	h.handleWSConnection(c, ctx, model.UserTypeMonitor)
}

// handleWSConnection contains common WebSocket connection handling logic.
func (h *WebSocketHandler) handleWSConnection(
	c context.Context,
	ctx *app.RequestContext,
	userType string,
) {
	// 1. Authenticate the user.
	userID, tenantID, username, agentNum, err := h.authenticate(ctx)
	if err != nil {
		logger.Warn("websocket auth failed",
			zap.String("user_type", userType),
			zap.Error(err),
		)
		response.Fail(ctx, errors.ErrTokenInvalid)
		return
	}

	// For agents, use agent_num (the agent number) as the connection userID.
	// This matches agent_id in call events so call.ringing and other events route to the correct agent connection.
	connUserID := userID
	if userType == model.UserTypeAgent && agentNum != "" {
		if id, err := strconv.ParseInt(agentNum, 10, 64); err == nil && id > 0 {
			connUserID = id
		} else {
			// Agent numbers can be non-numeric, such as "A1001".
			// ConnectionManager uses an int64 userID and does not support string agent numbers.
			// Fall back to the original userID; supporting non-numeric agent numbers requires string keys in ConnectionManager.
			logger.Warn("agent_num is not numeric, fallback to user_id",
				zap.String("agent_num", agentNum),
			)
		}
	}

	// 2. Upgrade the WebSocket connection using the Hertz WebSocket adapter.
	err = h.upgrader.Upgrade(ctx, func(wsConn *websocket.Conn) {
		// 3. Create the connection object.
		connID := generateConnID()
		conn := model.NewConnection(wsConn, connID, userType, connUserID, tenantID, username)

		// 4. Register the connection.
		h.connManager.AddConnection(conn)

		// 5. Automatically subscribe to default topics.
		h.autoSubscribe(conn)

		// 6. Register the route for agents.
		// Use connUserID (the agent number) to match agent_id in call events;
		// otherwise events addressed as agent_id=1001, such as call.ringing, cannot find the route.
		if userType == model.UserTypeAgent && h.routerService != nil && h.routerService.IsEnabled() {
			if err := h.routerService.RegisterRoute(tenantID, connUserID, connID); err != nil {
				logger.Warn("register route failed",
					zap.String("conn_id", connID),
					zap.Int64("agent_id", connUserID),
					zap.Error(err),
				)
			}
		}

		// 7. Start the write goroutine before gateway sign-in and the welcome message so buffered outbound frames are consumed.
		go h.connManager.StartWritePump(conn)

		// 7.5 Agent command gateway: sign in on connection when the dn parameter is present.
		// On failure, send an error frame, allow a flush window, and close the connection instead of returning early.
		// readPump then exits immediately on the closed connection and cleanupConnection performs normal cleanup.
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
					// Allow the write goroutine a flush window before closing the connection.
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

		// 8. Send the connection success message unless sign-in failed and the connection is closed.
		if !sessionFailed {
			h.sendWelcome(conn)
		}

		logger.Info("websocket connection established",
			zap.String("conn_id", connID),
			zap.String("user_type", userType),
			zap.Int64("user_id", userID),
			zap.Int64("tenant_id", tenantID),
		)

		// 9. Start the read goroutine, which blocks until the connection closes; for failed sign-ins, ReadMessage returns an error immediately.
		h.readPump(conn)

		// 10. Clean up after the connection closes.
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

// ===== Authentication =====

// authenticate validates the user's identity from the request.
// It accepts a token from a query parameter or the Authorization header.
// It returns userID, tenantID, username, agentNum, and err.
// agentNum is valid only for agent roles and routes call events by agent number.
func (h *WebSocketHandler) authenticate(ctx *app.RequestContext) (userID int64, tenantID int64, username string, agentNum string, err error) {
	// Get the token from a query parameter because custom headers cannot be set during a WebSocket handshake.
	token := ctx.Query("token")
	if token == "" {
		// Get the token from the Authorization header.
		token = string(ctx.GetHeader("Authorization"))
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		return 0, 0, "", "", fmt.Errorf("token missing")
	}

	// Parse the JWT token.
	claims, err := parseToken(token, h.jwtSecret)
	if err != nil {
		return 0, 0, "", "", err
	}

	// Extract user information from the claims.
	userID, err = parseUserID(claims)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("invalid user_id in token: %w", err)
	}

	username, _ = claims["username"].(string)

	// Agent number from an agent-role token, used by the push service to route call events by agent number.
	agentNum, _ = claims["agent_num"].(string)

	// Get the tenant ID from the claims when present.
	tenantID = parseTenantID(claims)

	if userID <= 0 {
		return 0, 0, "", "", fmt.Errorf("invalid user id")
	}

	return userID, tenantID, username, agentNum, nil
}

// parseUserID parses userID from the claims.
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

// parseTenantID parses tenantID from the claims.
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

// parseToken parses a JWT token.
func parseToken(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method.
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

// ===== Automatic subscriptions =====

// autoSubscribe automatically subscribes a new connection to default topics.
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

// sendWelcome sends a welcome message.
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

// ===== Read goroutine =====

// readPump reads client messages.
// It blocks until the connection closes.
func (h *WebSocketHandler) readPump(conn *model.Connection) {
	defer func() {
		conn.Close()
	}()

	// Set the read timeout and pong handler.
	wsConn := conn.WSConn()
	if wsConn == nil {
		return
	}

	// Set the initial read timeout.
	_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))

	// Set the pong handler.
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

		// Process the client message.
		h.handleClientMessage(conn, message)
	}
}

// handleClientMessage processes a message sent by a client.
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
		// Update the heartbeat time for heartbeat messages.
		conn.UpdatePing()
		// Respond to the heartbeat.
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

// handleSubscribe processes a subscription request.
func (h *WebSocketHandler) handleSubscribe(conn *model.Connection, data map[string]interface{}) {
	// Extract topics from data.
	topics := extractStringArray(data, "topics")
	if len(topics) == 0 {
		h.pushService.PushError(conn, errors.CodeParamMissing, "缺少订阅主题", "")
		return
	}

	// Validate topic names.
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

// handleUnsubscribe processes an unsubscription request.
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

// ===== Cleanup =====

// cleanupConnection performs cleanup after a connection closes.
func (h *WebSocketHandler) cleanupConnection(conn *model.Connection) {
	// 1. Cancel all subscriptions.
	h.subService.UnsubscribeAll(conn)

	// 2. Unregister the route for agents.
	if conn.UserType() == model.UserTypeAgent && h.routerService != nil && h.routerService.IsEnabled() {
		if err := h.routerService.UnregisterRoute(conn.UserID(), conn.ConnID()); err != nil {
			logger.Warn("unregister route failed",
				zap.String("conn_id", conn.ConnID()),
				zap.Int64("agent_id", conn.UserID()),
				zap.Error(err),
			)
		}
	}

	// 3. Remove the connection from the connection manager.
	h.connManager.RemoveConnection(conn)

	// 4. Notify the gateway to stop keepalive without changing agent status; the session expires naturally.
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

// sendJSON sends a custom JSON message to a connection, including gateway session, kick, and ack messages.
func (h *WebSocketHandler) sendJSON(conn *model.Connection, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	conn.Send(data)
}

// ===== Utility functions =====

// generateConnID generates a connection ID.
func generateConnID() string {
	return fmt.Sprintf("conn_%d_%s", time.Now().UnixNano(), randomString(8))
}

// randomString generates a random string.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[int(now>>uint(i*5))%len(letters)]
	}
	return string(b)
}

// extractStringArray extracts a string array from a map.
func extractStringArray(data map[string]interface{}, key string) []string {
	if data == nil {
		return nil
	}

	val, ok := data[key]
	if !ok {
		return nil
	}

	// Try to assert []interface{}.
	if arr, ok := val.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	// Try to assert []string.
	if arr, ok := val.([]string); ok {
		return arr
	}

	return nil
}

// ===== HTTP health checks =====

// HealthCheck performs a health check.
func (h *WebSocketHandler) HealthCheck(c context.Context, ctx *app.RequestContext) {
	stats := h.connManager.GetStats()
	response.OK(ctx, map[string]interface{}{
		"status": "ok",
		"stats":  stats,
	})
}

// GetStats returns statistics.
func (h *WebSocketHandler) GetStats(c context.Context, ctx *app.RequestContext) {
	stats := h.connManager.GetStats()
	// Add topic statistics.
	stats.Topics = h.subService.GetAllTopics()
	response.OK(ctx, stats)
}

// Ensure the consts package is referenced.
var _ = consts.StatusOK
