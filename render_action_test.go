package raw

import (
	"encoding/json"
	"testing"
)

func TestRenderResponseRoundTripsStateAndDiagnostics(t *testing.T) {
	state := "state-1"
	path := "$.root"
	model := `{"type":"text","text":"Hello"}`
	response := RenderResponse{
		Content:    RenderContent{Surface: &model},
		StateToken: &state,
		Diagnostics: []Diagnostic{{
			Level:   DiagnosticLevelInfo,
			Message: "rendered",
			Path:    &path,
		}},
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded RenderResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.StateToken == nil || *decoded.StateToken != "state-1" {
		t.Fatalf("state token did not round trip: %#v", decoded.StateToken)
	}
	if decoded.Diagnostics[0].Level != DiagnosticLevelInfo {
		t.Fatalf("diagnostic level did not round trip: %s", decoded.Diagnostics[0].Level)
	}
	if decoded.Content.Surface == nil || *decoded.Content.Surface != model || decoded.Content.Nav != nil {
		t.Fatalf("surface content did not round trip: %#v", decoded.Content)
	}
}

// kyzmet-ai/kyzmet-workspace#2607: the nav request and resolution mirror the
// typed app-render WIT shapes, with the correlation IDs as fields.
func TestNavRequestAndResolutionMirrorTheTypedWIT(t *testing.T) {
	limit := int64(25)
	open := NavExpansionStateOpen
	request := RenderRequest{
		AppInstanceID: "binding-1",
		SurfaceID:     "nav",
		Input: &RenderInput{Nav: &NavRequest{
			RequestID:      "nav-1",
			DescriptorID:   "ai.example.todo.projects",
			DescriptorKind: NavDescriptorKindList,
			ExportName:     "app-nav.resolve-list",
			Topic:          "nav.projects",
			Parameters:     []NavParameter{{Name: "limit", Value: NavParameterValue{Integer: &limit}}},
			ExpansionState: &open,
		}},
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	const wantRequest = `{"app_instance_id":"binding-1","surface_id":"nav","input":{"nav":{"request_id":"nav-1","descriptor_id":"ai.example.todo.projects","descriptor_kind":"list","export_name":"app-nav.resolve-list","topic":"nav.projects","parameters":[{"name":"limit","value":{"integer":25}}],"expansion_state":"open"}}}`
	if string(encoded) != wantRequest {
		t.Fatalf("request encoding:\n got %s\nwant %s", encoded, wantRequest)
	}

	route := "work/inbox"
	parent := "ai.example.todo.work"
	export := "app-nav.resolve-list"
	detail := NavPlatformRouteAppDetail
	response := RenderResponse{Content: RenderContent{Nav: &NavResolution{
		RequestID:    request.Input.Nav.RequestID,
		DescriptorID: request.Input.Nav.DescriptorID,
		Entries: []NavEntry{
			{ID: "ai.example.todo.work", Label: "Work", Kind: NavEntryKind{Dynamic: &NavDynamic{
				Kind: NavDescriptorKindList,
				Pull: NavEntryPull{Topic: "nav.projects/work", Provider: NavPullProvider{WasmExport: &export}},
			}}},
			{ID: "ai.example.todo.work-inbox", Label: "Inbox", ParentID: &parent, Kind: NavEntryKind{Route: &route}},
			{ID: "ai.example.todo.heading", Label: "Later", Kind: NavEntryKind{Label: &struct{}{}}},
			{ID: "ai.example.todo.about", Label: "About", Kind: NavEntryKind{PlatformRoute: &detail}},
		},
	}}}
	encoded, err = json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	const wantResponse = `{"content":{"nav":{"request_id":"nav-1","descriptor_id":"ai.example.todo.projects","entries":[{"id":"ai.example.todo.work","label":"Work","kind":{"dynamic":{"kind":"list","pull":{"topic":"nav.projects/work","provider":{"wasm_export":"app-nav.resolve-list"}}}}},{"id":"ai.example.todo.work-inbox","label":"Inbox","parent_id":"ai.example.todo.work","kind":{"route":"work/inbox"}},{"id":"ai.example.todo.heading","label":"Later","kind":{"label":{}}},{"id":"ai.example.todo.about","label":"About","kind":{"platform_route":"app_detail"}}]}}}`
	if string(encoded) != wantResponse {
		t.Fatalf("response encoding:\n got %s\nwant %s", encoded, wantResponse)
	}
	var decoded RenderResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if decoded.Content.Nav == nil || decoded.Content.Nav.Entries[2].Kind.Label == nil {
		t.Fatalf("label case did not round trip: %#v", decoded.Content)
	}
}

func TestRenderEventUsesSemanticAction(t *testing.T) {
	request := RenderRequest{
		AppInstanceID: "app-1",
		SurfaceID:     "surface-1",
	}
	event := RenderEvent{
		AppInstanceID: request.AppInstanceID,
		SurfaceID:     request.SurfaceID,
		TriggerID:     "tasks.complete",
	}

	if event.AppInstanceID != request.AppInstanceID {
		t.Fatalf("instance mismatch")
	}
	if event.TriggerID != "tasks.complete" {
		t.Fatalf("unexpected trigger id: %s", event.TriggerID)
	}
}
