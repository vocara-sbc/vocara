package handler

import (
	"bytes"
	"context"
	"encoding/json"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// LicenseHandler 授权管理接口：授权信息查询、License 导入与重新加载。
type LicenseHandler struct {
	licenseSvc *service.LicenseService
}

// NewLicenseHandler 创建授权管理处理器。
func NewLicenseHandler(licenseSvc *service.LicenseService) *LicenseHandler {
	return &LicenseHandler{licenseSvc: licenseSvc}
}

// Info 返回当前授权状态与载荷摘要。
func (h *LicenseHandler) Info(ctx context.Context, c *app.RequestContext) {
	response.OK(c, h.licenseSvc.Info())
}

// Import 接收 License 文本，校验通过后落盘并热加载。
// 请求体支持 {"content": "<License 文本>"}，也支持直接以原始 License 文本作为 body。
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

// Reload 放弃内存状态，从磁盘重新加载 License。
func (h *LicenseHandler) Reload(ctx context.Context, c *app.RequestContext) {
	if err := h.licenseSvc.Reload(); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, h.licenseSvc.Info())
}

// extractLicenseContent 从请求体提取 License 文本：JSON 包裹取 content 字段，否则原样返回。
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
