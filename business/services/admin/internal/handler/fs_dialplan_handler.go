package handler

import (
	"context"

	"vocara/admin/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// FSDialplanHandler FreeSWITCH xml_curl 动态拨号计划接口。
//
// FS 加载 mod_xml_curl 后，每通呼叫定位拨号计划时实时向本接口拉取规则，
// 拨号规则不落媒体节点磁盘；外呼出局规则随 License 状态动态生效或阻断。
// 鉴权：xml_curl 不支持自定义 header，token 经 query 参数传递（与 app.secret_key 一致）。
type FSDialplanHandler struct {
	svc           *service.FSDialplanService
	internalToken string
}

// NewFSDialplanHandler 创建 FS 动态拨号计划处理器。internalToken 与 app.secret_key 一致。
func NewFSDialplanHandler(svc *service.FSDialplanService, internalToken string) *FSDialplanHandler {
	return &FSDialplanHandler{svc: svc, internalToken: internalToken}
}

// Dialplan xml_curl dialplan binding 回调（POST 表单：Context/Caller-Destination-Number 等）。
// 返回当前节点 / context 的拨号计划 XML；未授权时外呼规则为阻断段，来话与分机规则照常。
func (h *FSDialplanHandler) Dialplan(_ context.Context, c *app.RequestContext) {
	// query token 鉴权（xml_curl 无法自定义 header）
	if string(c.Query("token")) != h.internalToken {
		c.Data(403, "text/xml; charset=utf-8", notFoundDocument())
		return
	}

	node := string(c.Query("node"))
	contextName := dialplanContext(c)
	if node == "" || contextName == "" {
		c.Data(200, "text/xml; charset=utf-8", notFoundDocument())
		return
	}

	c.Data(200, "text/xml; charset=utf-8", h.svc.Render(node, contextName))
}

// dialplanContext 从 xml_curl 表单取当前 context。
// 不同 FS 版本字段名略有差异，按常见取值依次尝试。
func dialplanContext(c *app.RequestContext) string {
	for _, key := range []string{
		"Context",
		"Hunt-context",
		"Hunt-Context",
		"Caller-Context",
		"context",
	} {
		if v := string(c.PostForm(key)); v != "" {
			return v
		}
	}
	return ""
}
