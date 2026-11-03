// fs_dialplan_service.go dynamically serves dialplans through FreeSWITCH mod_xml_curl for each call.
// This service compiles rules in memory, and license status controls outbound calling.
package service

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"vocara/pkg/config"
)

// Media node identifiers: fs-agent for the agent side and fs-sbc for carrier access.
const (
	NodeAgent = "agent"
	NodeSBC   = "sbc"
)

// xmlHeader is the FS XML declaration on its own line because switch_xml preprocessing skips <? lines.
// Sharing a line with the root element causes the entire document to be discarded.
const xmlHeader = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n"

// licenseStatus exposes only the outbound permission decision for mock-based unit tests.
// *license.Manager satisfies this interface.
type licenseStatus interface {
	// OperationAllowed reports whether outbound operations are permitted; valid, expiring, and grace return true.
	OperationAllowed() bool
}

// actionSpec specifies an FS dialplan application and its arguments.
type actionSpec struct {
	// App is the FS application name, such as answer, set, bridge, park, or hangup.
	App string
	// Data contains application arguments and is empty for actions without arguments.
	Data string
}

// extSpec specifies a dialplan extension with a number pattern and ordered actions.
type extSpec struct {
	// Name identifies the extension in logs and xml_locate.
	Name string
	// Regex matches destination_number.
	Regex string
	// Actions execute in order after a match.
	Actions []actionSpec
	// Outbound marks a carrier outbound rule for replacement with a blocking extension when unauthorized.
	Outbound bool
}

// FSDialplanService dynamically serves dialplans.
type FSDialplanService struct {
	license licenseStatus
	cfg     config.FSDialplanConfig
}

// NewFSDialplanService creates a dialplan service.
// The license manager controls operations, and cfg supplies environment-specific node addresses and the carrier gateway name.
func NewFSDialplanService(license licenseStatus, cfg config.FSDialplanConfig) *FSDialplanService {
	return &FSDialplanService{license: license, cfg: cfg}
}

// Render generates dialplan XML for a media node and context.
// Unsupported nodes or contexts return a not found document, and FS handles the call as having no matching rules.
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

// NotFoundDocument returns the mod_xml_curl not found document.
func NotFoundDocument() []byte {
	return []byte(xmlHeader +
		`<document type="freeswitch/xml"><section name="result" result="not found"/></document>`)
}

// buildExtensions returns ordered rules for a node and context.
// Unauthorized outbound rules are replaced by blocking extensions with the same patterns.
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

// agentDefault builds the fs-agent default context for direct extension dialing, ACD agent ringing, and agent outbound calls.
func (s *FSDialplanService) agentDefault() []extSpec {
	return []extSpec{
		// Direct extension dialing tries local user/$1 before routing through fs-sbc to the carrier-side extension.
		{
			Name:  "acd_agent",
			Regex: `^(\d{4})$`,
			Actions: []actionSpec{
				{App: "bridge", Data: fmt.Sprintf(
					`{absolute_codec_string=^^:PCMA:PCMU}user/$1,sofia/to-sbc/$1@%s`, s.cfg.SBCPeer)},
			},
		},
		// ACD ringing transfers the customer leg to vocara_agent_<extension> in callctrl, and bridge creates the agent leg.
		// ${} propagates vocara_* variables from the customer leg, and ^^; customizes delimiters within {} to avoid commas.
		{
			Name:  "vocara_acd_agent",
			Regex: `^vocara_agent_(\d{4})$`,
			Actions: []actionSpec{
				{App: "set", Data: `transfer_ringback=%(1000,4000,450)`},
				{App: "bridge", Data: `{^^;absolute_codec_string=^^:PCMA:PCMU;ignore_early_media=true;hangup_after_bridge=true;vocara_role=agent;vocara_leg_type=b;vocara_call_type=${vocara_call_type};vocara_call_id=${vocara_call_id};vocara_agent_id=${vocara_agent_id};vocara_tenant_id=${vocara_tenant_id};vocara_queue_id=${vocara_queue_id};vocara_fs_instance=${vocara_fs_instance};call_timeout=30}user/$1`},
			},
		},
		// Agent outbound calls with 7 to 13 digits route through fs-sbc under license control.
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

// agentPublic builds the fs-agent public context, answering and parking inbound calls for dynamic callctrl queueing.
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

// sbcDefault builds the fs-sbc default context for carrier-side extension dialing and outbound calls.
// The four-digit extension rule precedes the \d+ carrier outbound rule.
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

// sbcPublic builds the fs-sbc public context and forwards DID incoming calls to fs-agent without business waiting.
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

// sbcCarrierTest builds the isolated fs-sbc carrier-test context and loops delivery back to 1003 to simulate carrier delivery.
// It places no carrier calls and is not license-restricted.
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

// blockedExt replaces an outbound rule with a blocking extension using the same pattern and hangs up without answering.
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

// xmlAttr escapes and quotes an XML attribute value.
func xmlAttr(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	_ = xml.EscapeText(&buf, []byte(s))
	buf.WriteByte('"')
	return buf.String()
}
