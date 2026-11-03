package handler

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"

	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// FSDirectoryHandler provides the FreeSWITCH xml_curl dynamic directory endpoint.
//
// With mod_xml_curl loaded, FS queries this endpoint in real time for extension credentials during SIP registration,
// so new extensions in tb_dn are immediately usable without synchronizing static directory XML files.
// Authentication uses a query parameter because xml_curl does not support custom headers; the token matches app.secret_key.
type FSDirectoryHandler struct {
	dnRepo        *repository.DNRepository
	internalToken string
	defaultTenant uint64
}

// NewFSDirectoryHandler creates an FS dynamic directory handler.
// internalToken matches app.secret_key, and defaultTenant is the default tenant for number lookups (1 in single-tenant deployments).
func NewFSDirectoryHandler(internalToken string, defaultTenant uint64) *FSDirectoryHandler {
	return &FSDirectoryHandler{
		dnRepo:        repository.NewDNRepository(),
		internalToken: internalToken,
		defaultTenant: defaultTenant,
	}
}

// Directory handles xml_curl directory binding callbacks using POST form fields such as action, user, and domain.
// It returns a directory XML document for an enabled extension; missing or disabled extensions return not found so FS rejects registration with 401.
func (h *FSDirectoryHandler) Directory(ctx context.Context, c *app.RequestContext) {
	// Authenticate with the query token because xml_curl cannot set custom headers.
	if string(c.Query("token")) != h.internalToken {
		c.Data(403, "text/xml; charset=utf-8", notFoundDocument())
		return
	}

	user := string(c.PostForm("user"))
	domain := string(c.PostForm("domain"))

	// The xml_curl directory binding supplies user and domain; both are required to locate the extension and its node.
	// domain identifies the FS node initiating registration (fs-sbc and fs-agent have distinct domains).
	// Without it, the realm is unknown, so return not found rather than assuming a default domain.
	if user == "" || domain == "" {
		c.Data(200, "text/xml; charset=utf-8", notFoundDocument())
		return
	}

	dn, err := h.dnRepo.GetByDNNum(ctx, h.defaultTenant, user)
	if err != nil || dn == nil || dn.Status != 1 {
		c.Data(200, "text/xml; charset=utf-8", notFoundDocument())
		return
	}

	c.Data(200, "text/xml; charset=utf-8", directoryDocument(domain, dn.DNNum, dn.DNName, dn.Password))
}

// notFoundDocument returns the xml_curl not-found response, which FS treats as an unknown user.
// The XML declaration must occupy its own line because switch_xml preprocessing skips <? lines and discards any content on the same line.
func notFoundDocument() []byte {
	return []byte("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n" +
		`<document type="freeswitch/xml"><section name="result" result="not found"/></document>`)
}

// directoryDocument generates directory XML containing the standard account variables required for extension registration.
func directoryDocument(domain, user, name, password string) []byte {
	displayName := name
	if displayName == "" {
		displayName = "Agent " + user
	}

	var buf bytes.Buffer
	// Keep the XML declaration on its own line because switch_xml preprocessing skips <? lines and discards the document if the root element shares that line.
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n")
	buf.WriteString(`<document type="freeswitch/xml"><section name="directory">`)
	fmt.Fprintf(&buf, `<domain name=%q>`, xmlAttr(domain))
	// The domain-level dial-string matches static directory/default.xml and is required when mod_xml_curl parses the domain.
	buf.WriteString(`<params><param name="dial-string" value="{sip_invite_domain=${dialed_domain},presence_id=${dialed_user}@${dialed_domain}}${sofia_contact(*/${dialed_user}@${dialed_domain})}"/></params>`)
	buf.WriteString(`<groups><group name="default"><users>`)
	fmt.Fprintf(&buf, `<user id=%q>`, xmlAttr(user))
	buf.WriteString(`<params>`)
	fmt.Fprintf(&buf, `<param name="password" value=%q/>`, xmlAttr(password))
	fmt.Fprintf(&buf, `<param name="vm-password" value=%q/>`, xmlAttr(user))
	buf.WriteString(`</params><variables>`)
	buf.WriteString(`<variable name="toll_allow" value="domestic,international,local"/>`)
	fmt.Fprintf(&buf, `<variable name="accountcode" value=%q/>`, xmlAttr(user))
	// Do not hard-code user_context: each node profile determines the extension dialplan context
	// (public forwarding for fs-sbc customers and default for fs-agent agents); a fixed value would override the profile and misroute the call.
	fmt.Fprintf(&buf, `<variable name="effective_caller_id_name" value=%q/>`, xmlAttr(displayName))
	fmt.Fprintf(&buf, `<variable name="effective_caller_id_number" value=%q/>`, xmlAttr(user))
	buf.WriteString(`<variable name="callgroup" value="default"/>`)
	buf.WriteString(`</variables></user>`)
	buf.WriteString(`</users></group></groups></domain></section></document>`)
	return buf.Bytes()
}

// xmlAttr escapes an XML attribute value (& < > ").
func xmlAttr(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// ===== Agent extension information (SIP credentials for frontend sign-in) =====

// MyDnInfo contains the signed-in agent's extension information for SIP registration; only this endpoint returns the password.
type MyDnInfo struct {
	DNNum    string `json:"dn_num"`
	DNName   string `json:"dn_name"`
	Password string `json:"password"`
}

// MyDn returns the enabled extension bound to the signed-in agent, including its SIP registration password.
// The binding is tb_dn.agent_number = login account (agent number). WebRTC requires the password in the browser.
// An unbound or disabled extension is a normal state for accounts such as administrators, so return 200 with empty data and let the frontend use default credentials.
func (h *FSDirectoryHandler) MyDn(ctx context.Context, c *app.RequestContext) {
	username := GetUsernameFromCtx(c)
	if username == "" {
		response.Fail(c, errors.ErrUnauthorized)
		return
	}

	dn, err := h.dnRepo.GetByAgentNum(ctx, h.defaultTenant, username)
	if err != nil || dn == nil || dn.Status != 1 {
		response.OK(c, nil)
		return
	}

	response.OK(c, &MyDnInfo{
		DNNum:    dn.DNNum,
		DNName:   dn.DNName,
		Password: dn.Password,
	})
}
