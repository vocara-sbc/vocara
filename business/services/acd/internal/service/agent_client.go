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
	// internalAPIPrefix is the internal API path prefix.
	internalAPIPrefix = "/api/v1/internal"
	// defaultHTTPTimeout is the default HTTP timeout.
	defaultHTTPTimeout = 5 * time.Second
)

// AgentCenterClient defines the agent center client interface.
type AgentCenterClient interface {
	// GetReadyAgent gets a ready agent.
	// It returns the agent ID as a string, the extension number, and any error.
	GetReadyAgent(ctx context.Context, tenantID, skillGroupID string) (agentID string, dnNumber string, err error)

	// AssignCall assigns a session to an agent and triggers ringing.
	AssignCall(ctx context.Context, tenantID, agentID, callID string, callInfo interface{}) error

	// ResumeReady resets an agent to the ready state after allocation failure.
	ResumeReady(ctx context.Context, tenantID, agentID string) error
}

// httpAgentCenterClient is an HTTP-based agent center client.
type httpAgentCenterClient struct {
	baseURL       string       // Agent center address, such as "http://127.0.0.1:8081".
	internalToken string       // Token for internal service authentication.
	httpClient    *http.Client // HTTP client.
}

// NewHTTPAgentCenterClient creates an HTTP agent center client.
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

// ===== API response structures =====

// apiResponse is a generic API response.
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// tryAcquireResponse contains the agent allocation response.
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

// ===== HTTP request helpers =====

// doPost sends a POST request.
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

	// Return directly when response data does not need parsing.
	if respData == nil {
		return nil
	}

	// Parse the generic response.
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Code != 0 {
		return fmt.Errorf("api error code=%d message=%s", apiResp.Code, apiResp.Message)
	}

	// Parse the data field.
	if apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, respData); err != nil {
			return fmt.Errorf("unmarshal response data: %w", err)
		}
	}

	return nil
}

// ===== Mock client for development and testing =====

// mockAgentClient simulates an agent center client for development.
type mockAgentClient struct{}

// NewMockAgentClient creates a mock agent center client.
func NewMockAgentClient() AgentCenterClient {
	return &mockAgentClient{}
}

func (m *mockAgentClient) GetReadyAgent(ctx context.Context, tenantID, skillGroupID string) (string, string, error) {
	// Simulate no ready agents.
	return "", "", nil
}

func (m *mockAgentClient) AssignCall(ctx context.Context, tenantID, agentID, callID string, callInfo interface{}) error {
	return nil
}

func (m *mockAgentClient) ResumeReady(ctx context.Context, tenantID, agentID string) error {
	return nil
}
