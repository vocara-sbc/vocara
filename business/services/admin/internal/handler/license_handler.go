package handler

import (
	"bytes"
	"context"
	"encoding/json"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// LicenseHandler provides license management endpoints for querying license information, importing licenses, and reloading them.
type LicenseHandler struct {
	licenseSvc *service.LicenseService
}

// NewLicenseHandler creates a license management handler.
func NewLicenseHandler(licenseSvc *service.LicenseService) *LicenseHandler {
	return &LicenseHandler{licenseSvc: licenseSvc}
}

// Info returns the current license status and payload summary.
func (h *LicenseHandler) Info(ctx context.Context, c *app.RequestContext) {
	response.OK(c, h.licenseSvc.Info())
}

// Import accepts license text, validates it, persists it, and loads it without a restart.
// The request body supports {"content": "<license text>"} or raw license text.
func (h *LicenseHandler) Import(ctx context.Context, c *app.RequestContext) {
	content := extractLicenseContent(c)
	if len(bytes.TrimSpace(content)) == 0 {
		response.FailParamError(c, "License 内容为空")
		return
	}
	if err := h.licenseSvc.Import(content); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, h.licenseSvc.Info())
}

// Reload discards the in-memory state and reloads the license from disk.
func (h *LicenseHandler) Reload(ctx context.Context, c *app.RequestContext) {
	if err := h.licenseSvc.Reload(); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, h.licenseSvc.Info())
}

// extractLicenseContent extracts license text from the content field of a JSON request body or returns the raw body unchanged.
func extractLicenseContent(c *app.RequestContext) []byte {
	body := c.Request.Body()
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var request struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(body, &request); err == nil {
			return []byte(request.Content)
		}
	}
	return body
}
