package raw

// DiagnosticLevel is the severity for renderer diagnostics.
type DiagnosticLevel string

const (
	DiagnosticLevelDebug   DiagnosticLevel = "debug"
	DiagnosticLevelInfo    DiagnosticLevel = "info"
	DiagnosticLevelWarning DiagnosticLevel = "warning"
	DiagnosticLevelError   DiagnosticLevel = "error"
)

// RenderRequest asks the app WASM module to produce a render model.
type RenderRequest struct {
	AppInstanceID string       `json:"app_instance_id"`
	SurfaceID     string       `json:"surface_id"`
	SphereID      *string      `json:"sphere_id,omitempty"`
	StateToken    *string      `json:"state_token,omitempty"`
	Input         *RenderInput `json:"input,omitempty"`
}

// RenderInput is the app-render render-input variant: exactly one field is set.
type RenderInput struct {
	// Surface is input for an app surface, in the app's own format.
	Surface *string `json:"surface,omitempty"`
	// Nav asks the app to resolve one of its dynamic nav descriptors.
	Nav *NavRequest `json:"nav,omitempty"`
}

// NavDescriptorKind is a dynamic nav descriptor's kind.
type NavDescriptorKind string

const (
	NavDescriptorKindItem NavDescriptorKind = "item"
	NavDescriptorKindList NavDescriptorKind = "list"
)

// NavExpansionState is a list's disclosure state in the shell.
type NavExpansionState string

const (
	NavExpansionStateOpen   NavExpansionState = "open"
	NavExpansionStateClosed NavExpansionState = "closed"
)

// NavRequest asks the app to resolve a dynamic ui.nav descriptor. Echo
// RequestID and DescriptorID in the NavResolution.
type NavRequest struct {
	RequestID      string             `json:"request_id"`
	DescriptorID   string             `json:"descriptor_id"`
	DescriptorKind NavDescriptorKind  `json:"descriptor_kind"`
	ExportName     string             `json:"export_name"`
	Topic          string             `json:"topic"`
	Parameters     []NavParameter     `json:"parameters,omitempty"`
	ExpansionState *NavExpansionState `json:"expansion_state,omitempty"`
}

// NavParameter is one named scalar pull parameter.
type NavParameter struct {
	Name  string            `json:"name"`
	Value NavParameterValue `json:"value"`
}

// NavParameterValue is the nav-parameter-value variant: exactly one field is set.
type NavParameterValue struct {
	Text    *string  `json:"text,omitempty"`
	Integer *int64   `json:"integer,omitempty"`
	Number  *float64 `json:"number,omitempty"`
	Boolean *bool    `json:"boolean,omitempty"`
}

// ResourceReference identifies a non-executable UI resource declared by the app.
type ResourceReference struct {
	ID        string  `json:"id"`
	Path      string  `json:"path"`
	Kind      string  `json:"kind"`
	Integrity *string `json:"integrity,omitempty"`
}

// Diagnostic is developer-facing render/trigger feedback.
type Diagnostic struct {
	Level   DiagnosticLevel `json:"level"`
	Message string          `json:"message"`
	Path    *string         `json:"path,omitempty"`
}

// RenderResponse carries what the app rendered plus metadata.
type RenderResponse struct {
	Content     RenderContent       `json:"content"`
	Resources   []ResourceReference `json:"resources,omitempty"`
	StateToken  *string             `json:"state_token,omitempty"`
	Diagnostics []Diagnostic        `json:"diagnostics,omitempty"`
	UpdateToken *string             `json:"update_token,omitempty"`
}

// RenderContent is the app-render render-content variant: exactly one field is set.
type RenderContent struct {
	// Surface is a surface's render model, in the app's own web-standard format.
	Surface *string `json:"surface,omitempty"`
	// Nav resolves a NavRequest.
	Nav *NavResolution `json:"nav,omitempty"`
}

// NavResolution is the app's answer to a NavRequest. The entry tree is flat:
// an entry with a ParentID belongs to that earlier list entry.
type NavResolution struct {
	RequestID    string     `json:"request_id"`
	DescriptorID string     `json:"descriptor_id"`
	Entries      []NavEntry `json:"entries,omitempty"`
}

// NavEntry is one entry of a NavResolution.
type NavEntry struct {
	ID       string       `json:"id"`
	Label    string       `json:"label"`
	Icon     *string      `json:"icon,omitempty"`
	ParentID *string      `json:"parent_id,omitempty"`
	Kind     NavEntryKind `json:"kind"`
}

// NavEntryKind is the nav-entry-kind variant: exactly one field is set. The
// payload-free Label case encodes as an empty object.
type NavEntryKind struct {
	Route         *string           `json:"route,omitempty"`
	PlatformRoute *NavPlatformRoute `json:"platform_route,omitempty"`
	Label         *struct{}         `json:"label,omitempty"`
	Dynamic       *NavDynamic       `json:"dynamic,omitempty"`
}

// NavPlatformRoute is an allowlisted platform route.
type NavPlatformRoute string

const NavPlatformRouteAppDetail NavPlatformRoute = "app_detail"

// NavDynamic is an item or list the shell resolves with its own pull.
type NavDynamic struct {
	Kind        NavDescriptorKind `json:"kind"`
	Pull        NavEntryPull      `json:"pull"`
	PushSubject *string           `json:"push_subject,omitempty"`
	Expansion   *NavExpansion     `json:"expansion,omitempty"`
}

// NavEntryPull says how the shell resolves a dynamic entry: the WIT nav-pull.
// NavPull is the manifest descriptor's pull.
type NavEntryPull struct {
	Topic      string          `json:"topic"`
	Parameters []NavParameter  `json:"parameters,omitempty"`
	Provider   NavPullProvider `json:"provider"`
}

// NavPullProvider is the nav-pull-provider variant: exactly one field is set.
type NavPullProvider struct {
	WasmExport *string              `json:"wasm_export,omitempty"`
	Platform   *NavPlatformProvider `json:"platform,omitempty"`
}

// NavPlatformProvider is an allowlisted platform nav provider.
type NavPlatformProvider string

const (
	NavPlatformProviderSphere NavPlatformProvider = "sphere"
	NavPlatformProviderApp    NavPlatformProvider = "app"
)

// NavExpansion is a list's initial expansion.
type NavExpansion string

const (
	NavExpansionOpen          NavExpansion = "open"
	NavExpansionDefaultOpen   NavExpansion = "default_open"
	NavExpansionDefaultClosed NavExpansion = "default_closed"
)

// RenderEvent is a semantic user trigger routed back into WASM.
type RenderEvent struct {
	AppInstanceID string  `json:"app_instance_id"`
	SurfaceID     string  `json:"surface_id"`
	TriggerID     string  `json:"triggerId"`
	StateToken    *string `json:"state_token,omitempty"`
	Payload       *string `json:"payload,omitempty"`
}

// CommandResponse carries a non-render command produced by event handling.
type CommandResponse struct {
	Command     string       `json:"command"`
	Payload     *string      `json:"payload,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// EventResponse is the app-render WIT variant shape for handle-event results.
type EventResponse struct {
	Render  *RenderResponse  `json:"render,omitempty"`
	Command *CommandResponse `json:"command,omitempty"`
}
