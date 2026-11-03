package raw_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	raw "kyzmet.ai/go"
)

func strPtr(s string) *string { return &s }

func minimalBuilder() *raw.ManifestBuilder {
	return raw.NewManifest("com.example.skill", "Example Skill", "1.0.0", "com.example", "Example Developer", "ed25519:example-public-key")
}

func skillComp(id string) raw.ComponentDesc {
	return raw.ComponentDesc{ID: id, Kind: raw.ComponentKindSkill, Wasm: id + ".wasm", EntryPoint: "main"}
}

func TestRequiredAppFieldsAreValidated(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *raw.ManifestBuilder
		want string
	}{
		{"id", raw.NewManifest("", "Name", "1.0.0", "com.example", "Developer", "ed25519:key"), "app.id"},
		{"name", raw.NewManifest("com.example.id", "", "1.0.0", "com.example", "Developer", "ed25519:key"), "app.name"},
		{"version", raw.NewManifest("com.example.id", "Name", "", "com.example", "Developer", "ed25519:key"), "app.version"},
		{"vendor_id", raw.NewManifest("com.example.id", "Name", "1.0.0", "", "Developer", "ed25519:key"), "app.vendor.id"},
		{"vendor_prefix", raw.NewManifest("org.other.id", "Name", "1.0.0", "com.example", "Developer", "ed25519:key"), "app.id must be namespaced"},
		{"vendor_name", raw.NewManifest("com.example.id", "Name", "1.0.0", "com.example", "", "ed25519:key"), "app.vendor.name"},
		{"vendor_public_key", raw.NewManifest("com.example.id", "Name", "1.0.0", "com.example", "Developer", ""), "app.vendor.public_key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.b.Build()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestComponentWasmIsRequiredAndSerialized(t *testing.T) {
	_, err := minimalBuilder().
		Component(raw.ComponentDesc{ID: "core", Kind: raw.ComponentKindSkill, EntryPoint: "main"}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "wasm") {
		t.Fatalf("expected component wasm error, got %v", err)
	}

	m, err := minimalBuilder().Component(skillComp("core")).Build()
	if err != nil {
		t.Fatal(err)
	}
	jsonStr, err := m.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatal(err)
	}
	app := parsed["app"].(map[string]interface{})
	if _, ok := app["wasm"]; ok {
		t.Fatal("app.wasm must not serialize")
	}
	vendor := app["vendor"].(map[string]interface{})
	if vendor["public_key"] != "ed25519:example-public-key" {
		t.Fatalf("unexpected vendor public key: %v", vendor["public_key"])
	}
	component := parsed["components"].([]interface{})[0].(map[string]interface{})
	if component["wasm"] != "core.wasm" {
		t.Fatalf("unexpected component wasm: %v", component["wasm"])
	}
}

func TestDuplicateComponentIDFails(t *testing.T) {
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Component(skillComp("core")).
		Build()
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate component error, got %v", err)
	}
}

func TestScriptAndTriggerConstraints(t *testing.T) {
	_, err := minimalBuilder().
		Component(raw.ComponentDesc{ID: "runner", Kind: raw.ComponentKindScript, Wasm: "runner.wasm", EntryPoint: "main"}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature error, got %v", err)
	}

	_, err = minimalBuilder().
		Component(raw.ComponentDesc{ID: "runner", Kind: raw.ComponentKindScript, Wasm: "runner.wasm", EntryPoint: "main", Signature: strPtr("sha256:abc123")}).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	_, err = minimalBuilder().
		Component(raw.ComponentDesc{ID: "watcher", Kind: raw.ComponentKindTrigger, Wasm: "watcher.wasm", EntryPoint: "main"}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("expected listen error, got %v", err)
	}
}

func TestTriggerListenIsSyntactic(t *testing.T) {
	_, err := minimalBuilder().
		Component(raw.ComponentDesc{
			ID: "calendar", Kind: raw.ComponentKindTrigger, Wasm: "calendar.wasm",
			EntryPoint: "main", Listen: []string{"connector.calendar.event-created"},
		}).
		Build()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCapabilitiesTopicsAndA2aAgentsSerialize(t *testing.T) {
	m, err := minimalBuilder().
		WithCapabilities(raw.Capabilities{
			Topics: []raw.TopicDecl{{
				ID: "nav.projects", Schema: "schema://nav-projects", SensitivityCeiling: raw.SensitivityStandard,
			}},
			A2aAgents: []raw.A2aAgentDeclaration{{
				AgentCardURL: "https://agent.example/card.json",
				Skills:       []string{"summarize"}, Purpose: "Summarize documents",
				DataDirection: raw.DataDirectionOutbound, DataClassification: raw.DataClassificationContent,
			}},
		}).
		Component(skillComp("core")).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	jsonStr, _ := m.ToJSON()
	if !strings.Contains(jsonStr, `"topics"`) || !strings.Contains(jsonStr, `"a2a_agents"`) {
		t.Fatalf("expected capabilities in JSON: %s", jsonStr)
	}
}

func TestTypedNavStaticDividerAndDynamicDescriptors(t *testing.T) {
	m, err := minimalBuilder().
		Component(skillComp("core")).
		UI(raw.UiManifest{Nav: []raw.NavDescriptor{
			{
				Type: "static", ID: "com.example.skill.inbox", Label: "Inbox", Icon: "lucide:inbox",
				Target: &raw.NavTarget{Kind: "route", Route: "inbox"},
			},
			{Type: "divider", ID: "com.example.skill.divider"},
			{
				Type: "item", ID: "com.example.skill.next", Label: "Next",
				Pull: &raw.NavPull{Kind: "wasm", Export: "nav.resolve", Topic: "nav.next"},
				Push: &raw.NavPush{ByfrostSubject: "app.example.nav.next"},
			},
		}}).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	jsonStr, _ := m.ToJSON()
	if !strings.Contains(jsonStr, `"type":"static"`) || !strings.Contains(jsonStr, `"type":"divider"`) {
		t.Fatalf("expected typed nav descriptors in JSON: %s", jsonStr)
	}
}

func TestTypedNavRejectsSettingsAndBadIcons(t *testing.T) {
	_, err := minimalBuilder().
		Component(skillComp("core")).
		UI(raw.UiManifest{Nav: []raw.NavDescriptor{{
			Type: "static", ID: "inbox", Label: "Inbox",
			Target: &raw.NavTarget{Kind: "route", Route: "inbox"},
		}}}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "app.id") {
		t.Fatalf("expected namespaced nav id error, got %v", err)
	}

	_, err = minimalBuilder().
		Component(skillComp("core")).
		UI(raw.UiManifest{Nav: []raw.NavDescriptor{{
			Type: "static", ID: "settings", Label: "Settings",
			Target: &raw.NavTarget{Kind: "route", Route: "settings"},
		}}}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "app.id") {
		t.Fatalf("expected namespaced nav id error, got %v", err)
	}

	_, err = minimalBuilder().
		Component(skillComp("core")).
		UI(raw.UiManifest{Nav: []raw.NavDescriptor{{
			Type: "static", ID: "com.example.skill.bad-icon", Label: "Bad", Icon: "https://example.test/icon.svg",
			Target: &raw.NavTarget{Kind: "route", Route: "bad"},
		}}}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "lucide:") {
		t.Fatalf("expected icon namespace error, got %v", err)
	}
}

func TestTriggerBindingSerializes(t *testing.T) {
	pattern := "com.example.skill.actions.prepare-brief"
	m, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:                   "prepare-meeting-brief",
			Type:                 "platform.action.requested",
			Pattern:              &pattern,
			Component:            "core",
			ExportName:           "handle-trigger",
			RequiredCapabilities: []string{"workplace.meeting_brief.prepare"},
		}).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	jsonStr, err := m.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatal(err)
	}
	triggers := parsed["triggers"].([]interface{})
	trigger := triggers[0].(map[string]interface{})
	if trigger["id"] != "prepare-meeting-brief" {
		t.Fatalf("unexpected trigger id: %v", trigger["id"])
	}
	if trigger["type"] != "platform.action.requested" {
		t.Fatalf("unexpected trigger type: %v", trigger["type"])
	}
	if trigger["pattern"] != "com.example.skill.actions.prepare-brief" {
		t.Fatalf("unexpected trigger pattern: %v", trigger["pattern"])
	}
	if trigger["export_name"] != "handle-trigger" {
		t.Fatalf("unexpected trigger export_name: %v", trigger["export_name"])
	}
	caps := trigger["required_capabilities"].([]interface{})
	if len(caps) != 1 || caps[0] != "workplace.meeting_brief.prepare" {
		t.Fatalf("unexpected required_capabilities: %v", caps)
	}
}

func TestTriggerBindingWithoutPatternValid(t *testing.T) {
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "platform-trigger",
			Type:       "platform.action.requested",
			Component:  "core",
			ExportName: "handle-trigger",
		}).
		Build()
	if err != nil {
		t.Fatalf("expected no error for trigger without pattern, got %v", err)
	}
}

func TestTriggerBindingRejectsDuplicateIds(t *testing.T) {
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{ID: "dup", Type: "platform.action.requested", Component: "a", ExportName: "handle-trigger"}).
		Trigger(raw.TriggerBindingDeclaration{ID: "dup", Type: "platform.action.requested", Component: "b", ExportName: "handle-trigger"}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "duplicates another trigger") {
		t.Fatalf("expected duplicate trigger error, got %v", err)
	}
}

func TestTriggerBindingRejectsPatternOutsideAppNamespace(t *testing.T) {
	pattern := "org.other.skill.actions.something"
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-pattern",
			Type:       "platform.action.requested",
			Pattern:    &pattern,
			Component:  "core",
			ExportName: "handle-trigger",
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "namespaced under app.id") {
		t.Fatalf("expected namespace error, got %v", err)
	}
}

func TestTriggerBindingAcceptsPatternUnderAppId(t *testing.T) {
	pattern := "com.example.skill.actions.something"
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "good-pattern",
			Type:       "platform.action.requested",
			Pattern:    &pattern,
			Component:  "core",
			ExportName: "handle-trigger",
		}).
		Build()
	if err != nil {
		t.Fatalf("expected no error for pattern under app id, got %v", err)
	}
}

func TestTriggerBindingAcceptsWildcardPattern(t *testing.T) {
	pattern := "com.example.skill.actions.*"
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "wildcard-pattern",
			Type:       "platform.action.requested",
			Pattern:    &pattern,
			Component:  "core",
			ExportName: "handle-trigger",
		}).
		Build()
	if err != nil {
		t.Fatalf("expected no error for wildcard pattern, got %v", err)
	}
}

func TestTriggerBindingRejectsInvalidType(t *testing.T) {
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-type",
			Type:       "unknown.family.event",
			Component:  "core",
			ExportName: "handle-trigger",
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "known trigger type") {
		t.Fatalf("expected type validation error, got %v", err)
	}
}

func TestTriggerCatalogRejectsNamespaceShapedUnknowns(t *testing.T) {
	entry, ok := raw.TriggerCatalogEntryFor("time.schedule")
	if !ok || entry.PayloadSchema != "kyzmet.trigger.time-schedule.v1" {
		t.Fatalf("expected time.schedule catalog entry, got %#v", entry)
	}
	reserved, ok := raw.TriggerCatalogEntryFor("location.geofence.entered")
	if !ok || reserved.Status != raw.TriggerCatalogStatusReserved {
		t.Fatalf("expected reserved location catalog entry, got %#v", reserved)
	}

	for idx, triggerType := range []string{
		"connector.github.issue-commented",
		"device.state.battery.low",
		"location.geofence.exited",
		"health.record.created",
		"composite.and",
	} {
		_, err := minimalBuilder().
			Component(skillComp("core")).
			Trigger(raw.TriggerBindingDeclaration{
				ID:         fmt.Sprintf("bad-type-%d", idx),
				Type:       triggerType,
				Component:  "core",
				ExportName: "handle-trigger",
			}).
			Build()
		if err == nil || !strings.Contains(err.Error(), "known trigger type") {
			t.Fatalf("expected type validation error for %s, got %v", triggerType, err)
		}
	}
}

func TestTriggerBindingProviderConfigValidation(t *testing.T) {
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "github-push",
			Type:       "connector.github.push",
			Component:  "core",
			ExportName: "handle-trigger",
			Connector:  &raw.ConnectorConfig{Provider: "github"},
		}).
		Build()
	if err != nil {
		t.Fatalf("expected connector trigger config to validate, got %v", err)
	}

	_, err = minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-connector-config",
			Type:       "time.schedule",
			Component:  "core",
			ExportName: "handle-trigger",
			Connector:  &raw.ConnectorConfig{Provider: "github"},
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "connector.<provider>.<event>") {
		t.Fatalf("expected connector config type error, got %v", err)
	}

	_, err = minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-schedule-config",
			Type:       "platform.action.requested",
			Component:  "core",
			ExportName: "handle-trigger",
			Schedule:   &raw.ScheduleConfig{Cron: "0 8 * * *"},
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "time.schedule") {
		t.Fatalf("expected schedule config type error, got %v", err)
	}
}

func TestTriggerBindingScheduleConfigValidation(t *testing.T) {
	manifest, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "daily",
			Type:       "time.schedule",
			Component:  "core",
			ExportName: "handle-trigger",
			Schedule:   &raw.ScheduleConfig{Cron: "*/15 8-17 * * 1-5", Timezone: "America/Los_Angeles"},
		}).
		Build()
	if err != nil {
		t.Fatalf("expected schedule config to validate, got %v", err)
	}
	if manifest.Triggers[0].Schedule.Cron != "*/15 8-17 * * 1-5" {
		t.Fatalf("expected cron to serialize, got %#v", manifest.Triggers[0].Schedule)
	}

	_, err = minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "missing-schedule",
			Type:       "time.schedule",
			Component:  "core",
			ExportName: "handle-trigger",
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "schedule is required") {
		t.Fatalf("expected missing schedule error, got %v", err)
	}

	_, err = minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-cron",
			Type:       "time.schedule",
			Component:  "core",
			ExportName: "handle-trigger",
			Schedule:   &raw.ScheduleConfig{Cron: "60 8 * * *"},
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "five-field cron") {
		t.Fatalf("expected cron validation error, got %v", err)
	}

	_, err = minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-zone",
			Type:       "time.schedule",
			Component:  "core",
			ExportName: "handle-trigger",
			Schedule:   &raw.ScheduleConfig{Cron: "0 8 * * *", Timezone: "Los_Angeles"},
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "IANA timezone") {
		t.Fatalf("expected timezone validation error, got %v", err)
	}
}

func TestTriggerBindingRejectsNonFinalWildcard(t *testing.T) {
	pattern := "com.example.skill.*.send"
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-wildcard",
			Type:       "platform.action.requested",
			Pattern:    &pattern,
			Component:  "core",
			ExportName: "handle-trigger",
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "subject pattern") {
		t.Fatalf("expected subject pattern error, got %v", err)
	}
}

func TestTriggerBindingRejectsInvalidExportName(t *testing.T) {
	_, err := minimalBuilder().
		Component(skillComp("core")).
		Trigger(raw.TriggerBindingDeclaration{
			ID:         "bad-export",
			Type:       "platform.action.requested",
			Component:  "core",
			ExportName: "123 bad export!",
		}).
		Build()
	if err == nil || !strings.Contains(err.Error(), "export_name") {
		t.Fatalf("expected export name validation error, got %v", err)
	}
}
