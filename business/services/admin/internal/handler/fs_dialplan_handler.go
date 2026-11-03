package handler

import (
	"context"

	"vocara/admin/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// FSDialplanHandler provides the FreeSWITCH xml_curl dynamic dialplan endpoint.
//
// With mod_xml_curl loaded, FS retrieves rules from this endpoint in real time when locating the dialplan for each call,
// so dialplan rules are not stored on media node disks; outbound trunk rules are enabled or blocked according to license status.
// Authentication uses a query parameter because xml_curl does not support custom headers; the token matches app.secret_key.
type FSDialplanHandler struct {
	svc           *service.FSDialplanService
	internalToken string
}

// NewFSDialplanHandler creates an FS dynamic dialplan handler. internalToken matches app.secret_key.
func NewFSDialplanHandler(svc *service.FSDialplanService, internalToken string) *FSDialplanHandler {
	return &FSDialplanHandler{svc: svc, internalToken: internalToken}
}

// Dialplan handles xml_curl dialplan binding callbacks using POST form fields such as Context and Caller-Destination-Number.
// It returns dialplan XML for the current node and context; outbound rules become blocking sections without a license, while incoming call and extension rules remain available.
func (h *FSDialplanHandler) Dialplan(_ context.Context, c *app.RequestContext) {
	// Authenticate with the query token because xml_curl cannot set custom headers.
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

// dialplanContext reads the current context from the xml_curl form.
// Field names vary between FS versions, so try the common names in order.
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
