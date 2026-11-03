// Package service provides the business logic layer for the push service.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AgentCenterAPI abstracts the agent center internal API to support test fakes.
type AgentCenterAPI interface {
	EstablishSession(ctx context.Context, tenantID, agentID, dn string) (*AgentSessionInfo, error)
	RefreshSession(ctx context.Context, tenantID, agentID string) error
	DeleteSession(ctx context.Context, tenantID, agentID string) error
	UpdateStatus(ctx context.Context, tenantID, agentID string, status int, reason string) error
}

// AgentSessionInfo contains agent session information returned by the internal session API.
type AgentSessionInfo struct {
	Status     int      `json:"status"`
	StatusName string   `json:"status_name"`
	DnNumber   string   `json:"dn_number"`
	Skills     []string `json:"skills"`
}

type httpAgentCenterClient struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
}

// NewAgentCenterClient creates a client for the agent center internal API.
func NewAgentCenterClient(baseURL, internalToken string) AgentCenterAPI {
	return &httpAgentCenterClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		httpClient:    &http.Client{Timeout: 8 * time.Second},
	}
}

// doPost sends a POST request and validates the code in the standard response.
func (c *httpAgentCenterClient) doPost(ctx context.Context, path string, reqBody interface{}, respData interface{}) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call agentcenter: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("agentcenter http status %d", resp.StatusCode)
	}

	var apiResp struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if apiResp.Code != 0 {
		return fmt.Errorf("agentcenter biz error %d: %s", apiResp.Code, apiResp.Message)
	}
	if respData != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, respData); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}
	return nil
}

func (c *httpAgentCenterClient) EstablishSession(ctx context.Context, tenantID, agentID, dn string) (*AgentSessionInfo, error) {
	var info AgentSessionInfo
	err := c.doPost(ctx, "/api/v1/internal/agent/session", map[string]string{
		"tenant_id": tenantID, "agent_id": agentID, "dn_number": dn,
	}, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *httpAgentCenterClient) RefreshSession(ctx context.Context, tenantID, agentID string) error {
	return c.doPost(ctx, "/api/v1/internal/agent/session/refresh", map[string]string{
		"tenant_id": tenantID, "agent_id": agentID,
	}, nil)
}

func (c *httpAgentCenterClient) DeleteSession(ctx context.Context, tenantID, agentID string) error {
	return c.doPost(ctx, "/api/v1/internal/agent/session/delete", map[string]string{
		"tenant_id": tenantID, "agent_id": agentID,
	}, nil)
}

func (c *httpAgentCenterClient) UpdateStatus(ctx context.Context, tenantID, agentID string, status int, reason string) error {
	return c.doPost(ctx, "/api/v1/internal/agent/update-status", map[string]interface{}{
		"tenant_id": tenantID, "agent_id": agentID, "status": status, "reason": reason,
	}, nil)
}
