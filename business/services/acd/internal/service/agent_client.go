package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"vocara/pkg/logger"
)

const (
	// internalAPIPrefix 内部 API 路径前缀
	internalAPIPrefix = "/api/v1/internal"
	// defaultHTTPTimeout 默认 HTTP 超时时间
	defaultHTTPTimeout = 5 * time.Second
)

// AgentCenterClient 座席中心客户端接口
type AgentCenterClient interface {
	// GetReadyAgent 获取一个空闲座席
	// 返回 agentID（字符串形式）、分机号、是否成功
	GetReadyAgent(ctx context.Context, tenantID, skillGroupID string) (agentID string, dnNumber string, err error)

	// AssignCall 将会话分配给座席（通知座席振铃）
	AssignCall(ctx context.Context, tenantID, agentID, callID string, callInfo interface{}) error

	// ResumeReady 分配失败，重置座席为就绪状态
	ResumeReady(ctx context.Context, tenantID, agentID string) error
}

// httpAgentCenterClient 基于 HTTP 的座席中心客户端
type httpAgentCenterClient struct {
	baseURL       string       // 座席中心地址，如 "http://127.0.0.1:8081"
	internalToken string       // 内部服务鉴权 Token
	httpClient    *http.Client // HTTP 客户端
}

// NewHTTPAgentCenterClient 创建 HTTP 座席中心客户端
func NewHTTPAgentCenterClient(baseURL, internalToken string) AgentCenterClient {
	return &httpAgentCenterClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ===== API 响应结构 =====

// APIResponse 通用 API 响应
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// tryAcquireResponse 分配座席响应
type tryAcquireResponse struct {
	AgentID  string `json:"agent_id"`
	DnNumber string `json:"dn_number"`
	Success  bool   `json:"success"`
}

// ===== GetReadyAgent =====

func (c *httpAgentCenterClient) GetReadyAgent(ctx context.Context, tenantID, skillGroupID string) (string, string, error) {
	reqBody := map[string]string{
		"tenant_id":      tenantID,
		"skill_group_id": skillGroupID,
	}

	url := c.baseURL + internalAPIPrefix + "/agent/try-acquire"
	var resp tryAcquireResponse
	err := c.doPost(ctx, url, reqBody, &resp)
	if err != nil {
		logger.ErrorCtx(ctx, "get ready agent failed",
			zap.String("tenant_id", tenantID),
			zap.String("skill_group_id", skillGroupID),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("get ready agent: %w", err)
	}

	if !resp.Success {
		logger.DebugCtx(ctx, "no ready agent available",
			zap.String("tenant_id", tenantID),
			zap.String("skill_group_id", skillGroupID),
		)
		return "", "", nil
	}

	logger.DebugCtx(ctx, "got ready agent",
		zap.String("agent_id", resp.AgentID),
		zap.String("dn_number", resp.DnNumber),
		zap.String("skill_group_id", skillGroupID),
	)

	return resp.AgentID, resp.DnNumber, nil
}

// ===== AssignCall =====

func (c *httpAgentCenterClient) AssignCall(ctx context.Context, tenantID, agentID, callID string, callInfo interface{}) error {
	reqBody := map[string]string{
		"tenant_id": tenantID,
		"agent_id":  agentID,
		"call_id":   callID,
	}

	url := c.baseURL + internalAPIPrefix + "/agent/ring"
	err := c.doPost(ctx, url, reqBody, nil)
	if err != nil {
		logger.ErrorCtx(ctx, "assign call to agent failed",
			zap.String("agent_id", agentID),
			zap.String("call_id", callID),
			zap.Error(err),
		)
		return fmt.Errorf("assign call: %w", err)
	}

	logger.DebugCtx(ctx, "call assigned to agent",
		zap.String("agent_id", agentID),
		zap.String("call_id", callID),
	)

	return nil
}

// ===== ResumeReady =====

func (c *httpAgentCenterClient) ResumeReady(ctx context.Context, tenantID, agentID string) error {
	reqBody := map[string]string{
		"tenant_id": tenantID,
		"agent_id":  agentID,
	}

	url := c.baseURL + internalAPIPrefix + "/agent/resume-ready"
	err := c.doPost(ctx, url, reqBody, nil)
	if err != nil {
		logger.ErrorCtx(ctx, "resume agent ready failed",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
		return fmt.Errorf("resume ready: %w", err)
	}

	logger.DebugCtx(ctx, "agent resumed to ready",
		zap.String("agent_id", agentID),
	)

	return nil
}

// ===== HTTP 请求工具 =====

// doPost 发送 POST 请求
func (c *httpAgentCenterClient) doPost(ctx context.Context, url string, reqBody interface{}, respData interface{}) error {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBody))
	}

	// 如果不需要解析响应数据，直接返回
	if respData == nil {
		return nil
	}

	// 解析通用响应
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Code != 0 {
		return fmt.Errorf("api error code=%d message=%s", apiResp.Code, apiResp.Message)
	}

	// 解析 data 部分
	if apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, respData); err != nil {
			return fmt.Errorf("unmarshal response data: %w", err)
		}
	}

	return nil
}

// ===== Mock 客户端（用于开发/测试）=====

// mockAgentClient 模拟座席中心客户端（开发阶段使用）
type mockAgentClient struct{}

// NewMockAgentClient 创建模拟座席中心客户端
func NewMockAgentClient() AgentCenterClient {
	return &mockAgentClient{}
}

func (m *mockAgentClient) GetReadyAgent(ctx context.Context, tenantID, skillGroupID string) (string, string, error) {
	// 模拟：没有空闲座席
	return "", "", nil
}

func (m *mockAgentClient) AssignCall(ctx context.Context, tenantID, agentID, callID string, callInfo interface{}) error {
	return nil
}

func (m *mockAgentClient) ResumeReady(ctx context.Context, tenantID, agentID string) error {
	return nil
}
