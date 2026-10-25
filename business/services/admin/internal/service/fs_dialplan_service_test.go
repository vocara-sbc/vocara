package service

import (
	"bytes"
	"encoding/xml"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"vocara/pkg/config"
)

// mockLicense 授权状态桩。
type mockLicense struct {
	allowed bool
}

func (m mockLicense) OperationAllowed() bool { return m.allowed }

// newDialplanSvc 以固定节点地址构造被测服务。
func newDialplanSvc(allowed bool) *FSDialplanService {
	return NewFSDialplanService(mockLicense{allowed: allowed}, config.FSDialplanConfig{
		SBCPeer:       "172.16.1.13:5061",
		AgentExternal: "172.16.1.20:5080",
		Gateway:       "carrier",
	})
}

var extNameRe = regexp.MustCompile(`<extension name="([^"]+)"`)

// extNames 按文档顺序提取全部 extension 名。
func extNames(doc []byte) []string {
	matches := extNameRe.FindAllSubmatch(doc, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, string(m[1]))
	}
	return names
}

// assertContains 断言文档含子串。
func assertContains(t *testing.T, doc []byte, sub string) {
	t.Helper()
	if !bytes.Contains(doc, []byte(sub)) {
		t.Errorf("文档缺少子串 %q\n文档:\n%s", sub, doc)
	}
}

// assertNotContains 断言文档不含子串。
func assertNotContains(t *testing.T, doc []byte, sub string) {
	t.Helper()
	if bytes.Contains(doc, []byte(sub)) {
		t.Errorf("文档不应含子串 %q\n文档:\n%s", sub, doc)
	}
}

// assertWellFormed 断言文档是可解析的 XML。
func assertWellFormed(t *testing.T, doc []byte) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(doc))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("XML 解析失败: %v\n文档:\n%s", err, doc)
		}
	}
}

func TestAgentDefault_Allowed(t *testing.T) {
	doc := newDialplanSvc(true).Render(NodeAgent, "default")

	want := []string{"acd_agent", "vocara_acd_agent", "agent_outbound"}
	if got := extNames(doc); !reflect.DeepEqual(got, want) {
		t.Fatalf("extension 顺序不符, got %v want %v", got, want)
	}

	// 分机直拨：多 endpoint 与配置的 sbc 地址
	assertContains(t, doc, `data="{absolute_codec_string=^^:PCMA:PCMU}user/$1,sofia/to-sbc/$1@172.16.1.13:5061"`)
	// ACD 座席振铃：自定义分隔符、透传变量与超时
	assertContains(t, doc, `{^^;absolute_codec_string=^^:PCMA:PCMU;ignore_early_media=true`)
	assertContains(t, doc, `vocara_call_type=${vocara_call_type}`)
	assertContains(t, doc, `vocara_fs_instance=${vocara_fs_instance};call_timeout=30}user/$1`)
	// 座席外呼
	assertContains(t, doc, `data="sofia/to-sbc/$1@172.16.1.13:5061"`)
}

func TestAgentDefault_Blocked(t *testing.T) {
	doc := newDialplanSvc(false).Render(NodeAgent, "default")

	want := []string{"acd_agent", "vocara_acd_agent", "agent_outbound_blocked"}
	if got := extNames(doc); !reflect.DeepEqual(got, want) {
		t.Fatalf("extension 顺序不符, got %v want %v", got, want)
	}

	// 外呼规则被阻断段替换
	assertContains(t, doc, `data="hangup_cause=INCOMPATIBLE_DESTINATION"`)
	assertContains(t, doc, `<action application="hangup" data=""/>`)
	// 分机直拨（含其 sbc 寻址 endpoint）与座席振铃不受影响
	assertContains(t, doc, `user/$1,sofia/to-sbc/$1@172.16.1.13:5061`)
	assertContains(t, doc, `call_timeout=30}user/$1`)
}

func TestAgentPublic_UnaffectedByLicense(t *testing.T) {
	doc := newDialplanSvc(false).Render(NodeAgent, "public")

	if got := extNames(doc); !reflect.DeepEqual(got, []string{"vocara_inbound_did"}) {
		t.Fatalf("extension 不符, got %v", got)
	}
	assertContains(t, doc, `<action application="answer" data=""/>`)
	assertContains(t, doc, `data="vocara_did=$1"`)
	assertContains(t, doc, `<action application="park" data=""/>`)
}

func TestSBCDefault_OrderAndBlock(t *testing.T) {
	allowed := newDialplanSvc(true).Render(NodeSBC, "default")
	if got := extNames(allowed); !reflect.DeepEqual(got,
		[]string{"vocara_local_ext", "vocara_outbound_carrier"}) {
		t.Fatalf("允许态 extension 不符, got %v", got)
	}
	// 4 位分机规则先于出局规则
	if i, j := bytes.Index(allowed, []byte("vocara_local_ext")),
		bytes.Index(allowed, []byte("vocara_outbound_carrier")); i > j {
		t.Fatal("分机规则必须在出局规则之前")
	}
	assertContains(t, allowed, `data="{absolute_codec_string=^^:PCMA:PCMU}user/$1"`)
	assertContains(t, allowed, `data="sofia/gateway/carrier/$1"`)

	blocked := newDialplanSvc(false).Render(NodeSBC, "default")
	if got := extNames(blocked); !reflect.DeepEqual(got,
		[]string{"vocara_local_ext", "vocara_outbound_carrier_blocked"}) {
		t.Fatalf("阻断态 extension 不符, got %v", got)
	}
	// 内部分机照常，出局网关规则被阻断
	assertContains(t, blocked, `data="{absolute_codec_string=^^:PCMA:PCMU}user/$1"`)
	assertContains(t, blocked, `data="hangup_cause=INCOMPATIBLE_DESTINATION"`)
	assertNotContains(t, blocked, `sofia/gateway/carrier/$1`)
}

func TestSBCPublic_UnaffectedByLicense(t *testing.T) {
	doc := newDialplanSvc(false).Render(NodeSBC, "public")

	if got := extNames(doc); !reflect.DeepEqual(got, []string{"vocara_inbound_carrier"}) {
		t.Fatalf("extension 不符, got %v", got)
	}
	assertContains(t, doc, `data="vocara_role=sbc"`)
	assertContains(t, doc, `data="vocara_fs_type=inbound"`)
	assertContains(t, doc, `data="sofia/internal-to-agent/$1@172.16.1.20:5080"`)
}

func TestSBCCarrierTest_UnaffectedByLicense(t *testing.T) {
	doc := newDialplanSvc(false).Render(NodeSBC, "carrier-test")

	if got := extNames(doc); !reflect.DeepEqual(got, []string{"carrier_test_deliver"}) {
		t.Fatalf("extension 不符, got %v", got)
	}
	assertContains(t, doc, `data="{absolute_codec_string=^^:PCMA:PCMU}user/1003"`)
}

func TestUnknownNodeOrContext(t *testing.T) {
	for _, tc := range [][2]string{
		{NodeAgent, "unknown"},
		{NodeSBC, "unknown"},
		{"other", "default"},
	} {
		doc := newDialplanSvc(true).Render(tc[0], tc[1])
		assertContains(t, doc, `result="not found"`)
	}
}

func TestXMLDeclarationAndWellFormed(t *testing.T) {
	docs := [][]byte{
		newDialplanSvc(true).Render(NodeAgent, "default"),
		newDialplanSvc(false).Render(NodeAgent, "default"),
		newDialplanSvc(false).Render(NodeAgent, "public"),
		newDialplanSvc(true).Render(NodeSBC, "default"),
		newDialplanSvc(false).Render(NodeSBC, "public"),
		newDialplanSvc(false).Render(NodeSBC, "carrier-test"),
		NotFoundDocument(),
	}
	for _, doc := range docs {
		// XML 声明独占首行
		if !bytes.HasPrefix(doc, []byte(xmlHeader)) {
			t.Fatalf("文档未以独占一行的 XML 声明开头:\n%s", doc)
		}
		if !strings.Contains(string(doc), "\n<document") {
			t.Fatalf("根元素应另起一行:\n%s", doc)
		}
		assertWellFormed(t, doc)
	}
}
