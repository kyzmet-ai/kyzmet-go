package raw_test

import (
	"encoding/json"
	"math"
	"testing"

	raw "kyzmet.ai/go"
)

// ── Shorthand constructors ──────────────────────────────────────────────────

func TestTextComponent(t *testing.T) {
	c, err := raw.TextComponent("Hello, world!")
	if err != nil {
		t.Fatal(err)
	}
	j, err := c.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(j), &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "Text" {
		t.Errorf("expected type=Text, got %v", m["type"])
	}
	if m["text"] != "Hello, world!" {
		t.Errorf("expected text, got %v", m["text"])
	}
}

func TestCardComponent(t *testing.T) {
	c, err := raw.CardComponent("My Card")
	if err != nil {
		t.Fatal(err)
	}
	j, _ := c.ToJSON()
	var m map[string]interface{}
	json.Unmarshal([]byte(j), &m)
	if m["type"] != "Card" {
		t.Errorf("expected type=Card, got %v", m["type"])
	}
}

func TestAlertComponentSerializesSeverity(t *testing.T) {
	c, err := raw.AlertComponent("Watch out!", raw.AlertSeverityWarning)
	if err != nil {
		t.Fatal(err)
	}
	j, _ := c.ToJSON()
	var m map[string]interface{}
	json.Unmarshal([]byte(j), &m)
	if m["type"] != "Alert" {
		t.Errorf("expected Alert, got %v", m["type"])
	}
	if m["severity"] != "warning" {
		t.Errorf("expected severity=warning, got %v", m["severity"])
	}
}

func TestProgressComponentSetsValue(t *testing.T) {
	c, err := raw.ProgressComponent(0.75, "Loading...")
	if err != nil {
		t.Fatal(err)
	}
	j, _ := c.ToJSON()
	var m map[string]interface{}
	json.Unmarshal([]byte(j), &m)
	if m["type"] != "Progress" {
		t.Errorf("expected Progress, got %v", m["type"])
	}
	if m["value"].(float64) != 0.75 {
		t.Errorf("expected value=0.75, got %v", m["value"])
	}
}

// ── URL validation ──────────────────────────────────────────────────────────

func TestAssetURLAllowed(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeMedia).Src("asset://images/logo.png").Build(); err != nil {
		t.Errorf("asset:// URL should be allowed: %v", err)
	}
}

func TestRelativePathAllowed(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeMedia).Src("images/logo.png").Build(); err != nil {
		t.Errorf("relative path should be allowed: %v", err)
	}
}

func TestHTTPURLBlocked(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeMedia).Src("http://example.com/img.png").Build(); err == nil {
		t.Error("http:// URL should be blocked")
	}
}

func TestHTTPSURLBlocked(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeAction).Href("https://example.com").Build(); err == nil {
		t.Error("https:// URL should be blocked")
	}
}

func TestProtocolRelativeBlocked(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeMedia).Src("//evil.com/img.png").Build(); err == nil {
		t.Error("protocol-relative URL should be blocked")
	}
}

func TestJavascriptURLBlocked(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeAction).Href("javascript:alert(1)").Build(); err == nil {
		t.Error("javascript: URL should be blocked")
	}
}

func TestControlCharBlocked(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeMedia).Src("asset://\x00evil").Build(); err == nil {
		t.Error("URL with control char should be blocked")
	}
}

func TestBOMBlocked(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeMedia).Src("\uFEFFasset://legit").Build(); err == nil {
		t.Error("URL with BOM should be blocked")
	}
}

// ── Injection prevention ────────────────────────────────────────────────────

func TestScriptTagRejected(t *testing.T) {
	if _, err := raw.TextComponent("<script>alert(1)</script>"); err == nil {
		t.Error("text with <script> should be rejected")
	}
}

func TestIframeRejected(t *testing.T) {
	if _, err := raw.TextComponent("<iframe src='x'>"); err == nil {
		t.Error("text with <iframe> should be rejected")
	}
}

func TestTitleScriptRejected(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeCard).Title("<SCRIPT>bad()</SCRIPT>").Build(); err == nil {
		t.Error("title with <SCRIPT> should be rejected")
	}
}

// ── Numeric values ──────────────────────────────────────────────────────────

func TestValueFloatNaNRejected(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeProgress).ValueFloat(math.NaN()).Build(); err == nil {
		t.Error("NaN should be rejected")
	}
}

func TestValueFloatInfRejected(t *testing.T) {
	if _, err := raw.NewComponent(raw.ComponentTypeProgress).ValueFloat(math.Inf(1)).Build(); err == nil {
		t.Error("Infinity should be rejected")
	}
}

func TestValueIntSerialized(t *testing.T) {
	c, err := raw.NewComponent(raw.ComponentTypeBadge).ValueInt(42).Build()
	if err != nil {
		t.Fatal(err)
	}
	j, _ := c.ToJSON()
	var m map[string]interface{}
	json.Unmarshal([]byte(j), &m)
	if int(m["value"].(float64)) != 42 {
		t.Errorf("expected value=42, got %v", m["value"])
	}
}

// ── Children ────────────────────────────────────────────────────────────────

func TestChildrenSerialized(t *testing.T) {
	c, err := raw.NewComponent(raw.ComponentTypeCard).
		Title("My Card").
		Child(raw.NewComponent(raw.ComponentTypeText).Text("child text")).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	j, _ := c.ToJSON()
	var m map[string]interface{}
	json.Unmarshal([]byte(j), &m)
	children, ok := m["children"].([]interface{})
	if !ok || len(children) == 0 {
		t.Error("expected non-empty children array")
	}
}

// ── Descriptor ──────────────────────────────────────────────────────────────

func TestDescriptorContainsAllComponents(t *testing.T) {
	c1, _ := raw.TextComponent("line one")
	c2, _ := raw.TextComponent("line two")
	d := raw.NewDescriptor().Push(c1).Push(c2)
	j, err := d.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	json.Unmarshal([]byte(j), &m)
	comps, ok := m["components"].([]interface{})
	if !ok || len(comps) != 2 {
		t.Errorf("expected 2 components, got %v", m["components"])
	}
}

// ── All severity variants ───────────────────────────────────────────────────

func TestAllSeverityVariants(t *testing.T) {
	cases := []struct {
		sev      raw.AlertSeverity
		expected string
	}{
		{raw.AlertSeverityInfo, "info"},
		{raw.AlertSeverityWarning, "warning"},
		{raw.AlertSeverityError, "error"},
		{raw.AlertSeveritySuccess, "success"},
	}
	for _, tc := range cases {
		c, err := raw.AlertComponent("msg", tc.sev)
		if err != nil {
			t.Fatalf("severity %s: %v", tc.sev, err)
		}
		j, _ := c.ToJSON()
		var m map[string]interface{}
		json.Unmarshal([]byte(j), &m)
		if m["severity"] != tc.expected {
			t.Errorf("expected severity=%s, got %v", tc.expected, m["severity"])
		}
	}
}
