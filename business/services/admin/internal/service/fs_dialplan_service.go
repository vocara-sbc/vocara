// fs_dialplan_service.go 拨号计划动态下发：FreeSWITCH 经 mod_xml_curl 在每通呼叫定位
// 拨号计划时实时调用，规则编译于本服务、不落媒体节点磁盘，并与 License 状态联动控制外呼。
package service

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"vocara/pkg/config"
)

// 媒体节点标识：fs-agent（座席侧）/ fs-sbc（外线接入侧）
const (
	NodeAgent = "agent"
	NodeSBC   = "sbc"
)

// xmlHeader FS XML 文档声明。声明必须独占一行：switch_xml 预处理按行跳过 <? 行，
// 与根元素同行会导致整份文档被丢弃。
const xmlHeader = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n"

// licenseStatus 授权状态依赖（仅取外呼放行判定，便于以 mock 进行单元测试）。
// *license.Manager 天然满足本接口。
type licenseStatus interface {
	// OperationAllowed 返回当前是否允许外呼类操作（valid/expiring/grace 为 true）
	OperationAllowed() bool
}

// actionSpec 拨号动作规格：FS 应用名及其入参。
type actionSpec struct {
	// App FS 应用名（answer/set/bridge/park/hangup 等）
	App string
	// Data 应用入参；无参动作留空
	Data string
}

// extSpec 拨号 extension 规格：号码正则与有序动作。
type extSpec struct {
	// Name extension 名称（日志与 xml_locate 中标识）
	Name string
	// Regex destination_number 匹配正则
	Regex string
	// Actions 命中后按序执行的动作
	Actions []actionSpec
	// Outbound 是否为外呼出局规则：未授权时被替换为阻断段
	Outbound bool
}

// FSDialplanService 拨号计划动态下发服务。
type FSDialplanService struct {
	license licenseStatus
	cfg     config.FSDialplanConfig
}

// NewFSDialplanService 创建拨号计划下发服务。
// license 为授权管理器；cfg 提供环境相关的节点地址与出局网关名。
func NewFSDialplanService(license licenseStatus, cfg config.FSDialplanConfig) *FSDialplanService {
	return &FSDialplanService{license: license, cfg: cfg}
}

// Render 按媒体节点与 context 生成拨号计划 XML。
// 不支持的节点 / context 返回 not found 文档，FS 按无规则处理（呼叫失败）。
func (s *FSDialplanService) Render(node, contextName string) []byte {
	exts, ok := s.buildExtensions(node, contextName)
	if !ok {
		return NotFoundDocument()
	}

	var buf bytes.Buffer
	buf.WriteString(xmlHeader)
	buf.WriteString("<document type=\"freeswitch/xml\">\n")
	buf.WriteString("  <section name=\"dialplan\">\n")
	fmt.Fprintf(&buf, "    <context name=%s>\n", xmlAttr(contextName))
	for _, e := range exts {
		fmt.Fprintf(&buf, "      <extension name=%s>\n", xmlAttr(e.Name))
		fmt.Fprintf(&buf, "        <condition field=%s expression=%s>\n",
			xmlAttr("destination_number"), xmlAttr(e.Regex))
		for _, a := range e.Actions {
			fmt.Fprintf(&buf, "          <action application=%s data=%s/>\n",
				xmlAttr(a.App), xmlAttr(a.Data))
		}
		buf.WriteString("        </condition>\n")
		buf.WriteString("      </extension>\n")
	}
	buf.WriteString("    </context>\n")
	buf.WriteString("  </section>\n")
	buf.WriteString("</document>")
	return buf.Bytes()
}

// NotFoundDocument 返回 mod_xml_curl 的"未找到"文档。
func NotFoundDocument() []byte {
	return []byte(xmlHeader +
		`<document type="freeswitch/xml"><section name="result" result="not found"/></document>`)
}

// buildExtensions 返回指定节点 / context 的有序规则；
// 未授权时将外呼出局规则替换为同正则的阻断段。
func (s *FSDialplanService) buildExtensions(node, contextName string) ([]extSpec, bool) {
	var specs []extSpec
	switch node {
	case NodeAgent:
		switch contextName {
		case "default":
			specs = s.agentDefault()
		case "public":
			specs = s.agentPublic()
		default:
			return nil, false
		}
	case NodeSBC:
		switch contextName {
		case "default":
			specs = s.sbcDefault()
		case "public":
			specs = s.sbcPublic()
		case "carrier-test":
			specs = s.sbcCarrierTest()
		default:
			return nil, false
		}
	default:
		return nil, false
	}

	if !s.license.OperationAllowed() {
		for i, e := range specs {
			if e.Outbound {
				specs[i] = blockedExt(e)
			}
		}
	}
	return specs, true
}

// agentDefault fs-agent default context：分机直拨、ACD 座席振铃、座席外呼。
func (s *FSDialplanService) agentDefault() []extSpec {
	return []extSpec{
		// 分机直拨：先本机 user/$1，未注册再转发 fs-sbc 寻址外线侧分机
		{
			Name:  "acd_agent",
			Regex: `^(\d{4})$`,
			Actions: []actionSpec{
				{App: "bridge", Data: fmt.Sprintf(
					`{absolute_codec_string=^^:PCMA:PCMU}user/$1,sofia/to-sbc/$1@%s`, s.cfg.SBCPeer)},
			},
		},
		// ACD 座席振铃：callctrl 将客户腿转入 vocara_agent_<分机>，bridge 原生创建座席腿。
		// vocara_* 经 ${} 从客户腿透传；{} 内用 ^^; 自定义分隔符避开默认逗号。
		{
			Name:  "vocara_acd_agent",
			Regex: `^vocara_agent_(\d{4})$`,
			Actions: []actionSpec{
				{App: "set", Data: `transfer_ringback=%(1000,4000,450)`},
				{App: "bridge", Data: `{^^;absolute_codec_string=^^:PCMA:PCMU;ignore_early_media=true;hangup_after_bridge=true;vocara_role=agent;vocara_leg_type=b;vocara_call_type=${vocara_call_type};vocara_call_id=${vocara_call_id};vocara_agent_id=${vocara_agent_id};vocara_tenant_id=${vocara_tenant_id};vocara_queue_id=${vocara_queue_id};vocara_fs_instance=${vocara_fs_instance};call_timeout=30}user/$1`},
			},
		},
		// 座席主动外呼：7~13 位转发 fs-sbc，受 License 控制
		{
			Name:     "agent_outbound",
			Regex:    `^(\d{7,13})$`,
			Outbound: true,
			Actions: []actionSpec{
				{App: "bridge", Data: fmt.Sprintf(`sofia/to-sbc/$1@%s`, s.cfg.SBCPeer)},
			},
		},
	}
}

// agentPublic fs-agent public context：外线呼入 answer + park，由 callctrl 动态入队。
func (s *FSDialplanService) agentPublic() []extSpec {
	return []extSpec{
		{
			Name:  "vocara_inbound_did",
			Regex: `^(\d{4,12})$`,
			Actions: []actionSpec{
				{App: "answer"},
				{App: "set", Data: `vocara_did=$1`},
				{App: "park"},
			},
		},
	}
}

// sbcDefault fs-sbc default context：外线侧分机直拨、外呼出局。
// 4 位分机规则必须先于 \d+ 出局规则匹配。
func (s *FSDialplanService) sbcDefault() []extSpec {
	return []extSpec{
		{
			Name:  "vocara_local_ext",
			Regex: `^(\d{4})$`,
			Actions: []actionSpec{
				{App: "bridge", Data: `{absolute_codec_string=^^:PCMA:PCMU}user/$1`},
			},
		},
		{
			Name:     "vocara_outbound_carrier",
			Regex:    `^(\d+)$`,
			Outbound: true,
			Actions: []actionSpec{
				{App: "bridge", Data: fmt.Sprintf(`sofia/gateway/%s/$1`, s.cfg.Gateway)},
			},
		},
	}
}

// sbcPublic fs-sbc public context：DID 来话统一转发 fs-agent，不承载业务等待。
func (s *FSDialplanService) sbcPublic() []extSpec {
	return []extSpec{
		{
			Name:  "vocara_inbound_carrier",
			Regex: `^(\d+)$`,
			Actions: []actionSpec{
				{App: "set", Data: `vocara_role=sbc`},
				{App: "set", Data: `vocara_fs_type=inbound`},
				{App: "bridge", Data: fmt.Sprintf(
					`sofia/internal-to-agent/$1@%s`, s.cfg.AgentExternal)},
			},
		},
	}
}

// sbcCarrierTest fs-sbc carrier-test context：测试落地回环到 1003，模拟运营商送达。
// 仅在隔离的测试 profile 可达，不产生真实出局，不受 License 限制。
func (s *FSDialplanService) sbcCarrierTest() []extSpec {
	return []extSpec{
		{
			Name:  "carrier_test_deliver",
			Regex: `^(\d+)$`,
			Actions: []actionSpec{
				{App: "bridge", Data: `{absolute_codec_string=^^:PCMA:PCMU}user/1003`},
			},
		},
	}
}

// blockedExt 将外呼规则替换为同正则的阻断段：不接听、不计费，直接挂断。
func blockedExt(spec extSpec) extSpec {
	return extSpec{
		Name:  spec.Name + "_blocked",
		Regex: spec.Regex,
		Actions: []actionSpec{
			{App: "set", Data: "hangup_cause=INCOMPATIBLE_DESTINATION"},
			{App: "hangup"},
		},
	}
}

// xmlAttr XML 属性值编码（含转义与引号包裹）。
func xmlAttr(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	_ = xml.EscapeText(&buf, []byte(s))
	buf.WriteByte('"')
	return buf.String()
}
