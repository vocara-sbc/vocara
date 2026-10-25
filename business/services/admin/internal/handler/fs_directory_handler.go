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

// FSDirectoryHandler FreeSWITCH xml_curl 动态目录接口。
//
// FS 加载 mod_xml_curl 后，SIP 注册时实时向本接口查询分机凭据，
// 新分机（tb_dn）即建即用，无需静态 directory xml 文件同步。
// 鉴权：xml_curl 不支持自定义 header，token 经 query 参数传递（与 app.secret_key 一致）。
type FSDirectoryHandler struct {
	dnRepo        *repository.DNRepository
	internalToken string
	defaultTenant uint64
}

// NewFSDirectoryHandler 创建 FS 动态目录处理器。
// internalToken 与 app.secret_key 一致；defaultTenant 为号码查询的默认租户（当前单租户部署为 1）。
func NewFSDirectoryHandler(internalToken string, defaultTenant uint64) *FSDirectoryHandler {
	return &FSDirectoryHandler{
		dnRepo:        repository.NewDNRepository(),
		internalToken: internalToken,
		defaultTenant: defaultTenant,
	}
}

// Directory xml_curl directory binding 回调（POST 表单：action/user/domain/...）。
// 命中启用分机返回 directory XML 文档；未命中/禁用返回 not found，FS 据此拒绝注册（401）。
func (h *FSDirectoryHandler) Directory(ctx context.Context, c *app.RequestContext) {
	// query token 鉴权（xml_curl 无法自定义 header）
	if string(c.Query("token")) != h.internalToken {
		c.Data(403, "text/xml; charset=utf-8", notFoundDocument())
		return
	}

	user := string(c.PostForm("user"))
	domain := string(c.PostForm("domain"))

	// user/domain 由 xml_curl directory 绑定通道携带，缺一即无法定位分机与所属节点：
	// domain 标识发起注册的 FS 节点（fs-sbc / fs-agent 各有其域名），
	// 缺失时无从确定 realm，按未命中处理，不臆造默认 domain。
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

// notFoundDocument xml_curl 的"未找到"响应（FS 收到后按无此用户处理）。
// 注意：XML 声明必须独占一行（switch_xml 预处理按行跳过 <? 行，同行内容会一并丢弃）。
func notFoundDocument() []byte {
	return []byte("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n" +
		`<document type="freeswitch/xml"><section name="result" result="not found"/></document>`)
}

// directoryDocument 生成 directory XML（分机注册所需的标准账号变量集合）。
func directoryDocument(domain, user, name, password string) []byte {
	displayName := name
	if displayName == "" {
		displayName = "Agent " + user
	}

	var buf bytes.Buffer
	// XML 声明独占一行：switch_xml 预处理按行跳过 <? 行，声明与根元素同行会导致全文被丢弃
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n")
	buf.WriteString(`<document type="freeswitch/xml"><section name="directory">`)
	fmt.Fprintf(&buf, `<domain name=%q>`, xmlAttr(domain))
	// domain 级 dial-string 与静态 directory/default.xml 保持一致（mod_xml_curl 解析 domain 时必需）
	buf.WriteString(`<params><param name="dial-string" value="{sip_invite_domain=${dialed_domain},presence_id=${dialed_user}@${dialed_domain}}${sofia_contact(*/${dialed_user}@${dialed_domain})}"/></params>`)
	buf.WriteString(`<groups><group name="default"><users>`)
	fmt.Fprintf(&buf, `<user id=%q>`, xmlAttr(user))
	buf.WriteString(`<params>`)
	fmt.Fprintf(&buf, `<param name="password" value=%q/>`, xmlAttr(password))
	fmt.Fprintf(&buf, `<param name="vm-password" value=%q/>`, xmlAttr(user))
	buf.WriteString(`</params><variables>`)
	buf.WriteString(`<variable name="toll_allow" value="domestic,international,local"/>`)
	fmt.Fprintf(&buf, `<variable name="accountcode" value=%q/>`, xmlAttr(user))
	// 不写死 user_context：分机拨号计划的 context 由各节点 profile 决定
	//（fs-sbc 客户走 public 转发、fs-agent 座席走 default），写死会覆盖 profile 导致方向错。
	fmt.Fprintf(&buf, `<variable name="effective_caller_id_name" value=%q/>`, xmlAttr(displayName))
	fmt.Fprintf(&buf, `<variable name="effective_caller_id_number" value=%q/>`, xmlAttr(user))
	buf.WriteString(`<variable name="callgroup" value="default"/>`)
	buf.WriteString(`</variables></user>`)
	buf.WriteString(`</users></group></groups></domain></section></document>`)
	return buf.Bytes()
}

// xmlAttr XML 属性值转义（& < > "）。
func xmlAttr(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// ===== 座席分机信息（前端签入获取 SIP 凭据）=====

// MyDnInfo 当前登录座席的分机信息（SIP 注册用；仅此接口下发密码）。
type MyDnInfo struct {
	DNNum    string `json:"dn_num"`
	DNName   string `json:"dn_name"`
	Password string `json:"password"`
}

// MyDn 查询当前登录座席绑定的启用分机（含 SIP 注册密码）。
// 绑定关系：tb_dn.agent_number = 登录账号（工号）。WebRTC 场景密码必须在浏览器可用，属行业常态。
// 未绑定分机或分机停用属于正常业务状态（如管理账号），返回 200 空数据，前端按无可用分机走默认凭据。
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
