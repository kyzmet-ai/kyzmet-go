// protobuf-policy-file: migrate (1 site) — mod app manifest serialized for packaging.
// App manifest builder with construction-time validation.
//
// Mirrors the validation in core's ManifestParser.kt so that skill
// authors catch violations before packaging.
package raw

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ManifestError is returned when a manifest fails validation.
type ManifestError struct {
	msg string
}

func (e *ManifestError) Error() string { return "manifest validation error: " + e.msg }

func manifestErr(msg string) *ManifestError { return &ManifestError{msg: msg} }

// ── Enums ──────────────────────────────────────────────────────────────────

// ComponentKind is the component type as it appears in the YAML manifest.
type ComponentKind string

const (
	ComponentKindSkill   ComponentKind = "skill"
	ComponentKindScript  ComponentKind = "script"
	ComponentKindTrigger ComponentKind = "trigger"
)

// DataDirection is the direction of an A2A agent data flow.
type DataDirection string

const (
	DataDirectionInbound       DataDirection = "inbound"
	DataDirectionOutbound      DataDirection = "outbound"
	DataDirectionBidirectional DataDirection = "bidirectional"
)

// DataClassification classifies data sensitivity in an A2A agent declaration.
type DataClassification string

const (
	DataClassificationMetadata  DataClassification = "metadata"
	DataClassificationContent   DataClassification = "content"
	DataClassificationSensitive DataClassification = "sensitive"
)

// VaultAccess describes the access type exposed to a host program.
type VaultAccess string

const (
	VaultAccessRead  VaultAccess = "read"
	VaultAccessWrite VaultAccess = "write"
	VaultAccessQuery VaultAccess = "query"
)

// FrontendSurface is a platform UI surface an app may request.
type FrontendSurface string

const (
	FrontendSurfaceChatWidget FrontendSurface = "chat_widget"
)

// UiSurface is a platform-owned UI surface an app may request.
type UiSurface string

const (
	UiSurfaceAppIframe  UiSurface = "app_iframe"
	UiSurfaceChatWidget UiSurface = "chat_widget"
)

// ── Data structs ───────────────────────────────────────────────────────────

// AppMeta holds the app identity fields.
type AppMeta struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Vendor      VendorMeta `json:"vendor"`
	Description *string    `json:"description,omitempty"`
}

// VendorMeta holds publisher identity for review and package provenance.
type VendorMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// ComponentDesc describes a skill component in the manifest.
type ComponentDesc struct {
	ID         string        `json:"id"`
	Kind       ComponentKind `json:"type"`
	Wasm       string        `json:"wasm"`
	EntryPoint string        `json:"entry_point"`
	Signature  *string       `json:"signature,omitempty"`
	Listen     []string      `json:"listen,omitempty"`
}

// TopicDecl declares a topic capability.
// Named TopicDecl to avoid collision with the raw.go type alias `type Topic = string`.
type TopicDecl struct {
	ID                 string      `json:"id"`
	Schema             string      `json:"schema"`
	SensitivityCeiling Sensitivity `json:"sensitivity_ceiling"`
	Description        *string     `json:"description,omitempty"`
}

// A2aAgentDeclaration declares an A2A agent this skill interacts with.
type A2aAgentDeclaration struct {
	AgentCardURL       string             `json:"agent_card_url"`
	Skills             []string           `json:"skills"`
	Purpose            string             `json:"purpose"`
	DataDirection      DataDirection      `json:"data_direction"`
	DataClassification DataClassification `json:"data_classification"`
}

// FrontendManifest declares frontend entry points and platform UI surfaces.
type FrontendManifest struct {
	Entry    string            `json:"entry,omitempty"`
	Mode     string            `json:"mode,omitempty"`
	Surfaces []FrontendSurface `json:"surfaces,omitempty"`
}

// NavTarget is a typed shell navigation target.
type NavTarget struct {
	Kind  string `json:"kind"`
	Route string `json:"route"`
}

// NavPull is a typed dynamic nav resolver.
type NavPull struct {
	Kind     string                 `json:"kind"`
	Export   string                 `json:"export,omitempty"`
	Provider string                 `json:"provider,omitempty"`
	Topic    string                 `json:"topic"`
	Params   map[string]interface{} `json:"params,omitempty"`
}

// NavPush is a Byfrost subject hint for asynchronous dynamic nav updates.
type NavPush struct {
	ByfrostSubject string `json:"byfrost_subject"`
}

// NavDescriptor describes a platform-rendered shell nav contribution.
type NavDescriptor struct {
	Type         string          `json:"type"`
	ID           string          `json:"id"`
	Icon         string          `json:"icon,omitempty"`
	IconFallback string          `json:"icon_fallback,omitempty"`
	Label        string          `json:"label,omitempty"`
	Target       *NavTarget      `json:"target,omitempty"`
	Pull         *NavPull        `json:"pull,omitempty"`
	Push         *NavPush        `json:"push,omitempty"`
	Expansion    string          `json:"expansion,omitempty"`
	Items        []NavDescriptor `json:"items,omitempty"`
}

// UiManifest declares platform-owned UI and shell nav contributions.
type UiManifest struct {
	Surface UiSurface       `json:"surface,omitempty"`
	Nav     []NavDescriptor `json:"nav,omitempty"`
}

// TriggerBindingDeclaration declares a trigger binding in the manifest.
type TriggerBindingDeclaration struct {
	ID                        string             `json:"id"`
	Type                      string             `json:"type"`
	Pattern                   *string            `json:"pattern,omitempty"`
	Component                 string             `json:"component"`
	ExportName                string             `json:"export_name"`
	Label                     *string            `json:"label,omitempty"`
	Description               *string            `json:"description,omitempty"`
	RequiredCapabilities      []string           `json:"required_capabilities,omitempty"`
	RequiredConsentCategories []string           `json:"required_consent_categories,omitempty"`
	Idempotency               *IdempotencyConfig `json:"idempotency,omitempty"`
	Receipt                   *ReceiptConfig     `json:"receipt,omitempty"`
	Schedule                  *ScheduleConfig    `json:"schedule,omitempty"`
	Connector                 *ConnectorConfig   `json:"connector,omitempty"`
}

// IdempotencyConfig declares idempotency behavior for a trigger binding.
type IdempotencyConfig struct {
	KeySource       *string `json:"key_source,omitempty"`
	DuplicatePolicy *string `json:"duplicate_policy,omitempty"`
}

// ReceiptConfig declares receipt visibility for a trigger binding.
type ReceiptConfig struct {
	UserVisible *bool `json:"user_visible,omitempty"`
}

// ScheduleConfig declares schedule-specific trigger configuration.
type ScheduleConfig struct {
	Cron     string `json:"cron,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// ConnectorConfig declares connector-specific trigger filters.
type ConnectorConfig struct {
	Provider string `json:"provider"`
}

// TriggerCatalogStatus describes rollout state for a trigger type.
type TriggerCatalogStatus string

const (
	TriggerCatalogStatusAvailable TriggerCatalogStatus = "available"
	TriggerCatalogStatusDevOnly   TriggerCatalogStatus = "dev_only"
	TriggerCatalogStatusReserved  TriggerCatalogStatus = "reserved"
	TriggerCatalogStatusDesigned  TriggerCatalogStatus = "designed"
	TriggerCatalogStatusBlocked   TriggerCatalogStatus = "blocked"
)

// TriggerCatalogFamily groups trigger types by broker provider family.
type TriggerCatalogFamily string

const (
	TriggerCatalogFamilyPlatformAction  TriggerCatalogFamily = "platform_action"
	TriggerCatalogFamilySchedule        TriggerCatalogFamily = "schedule"
	TriggerCatalogFamilyConnector       TriggerCatalogFamily = "connector"
	TriggerCatalogFamilyDeviceState     TriggerCatalogFamily = "device_state"
	TriggerCatalogFamilySensitiveMobile TriggerCatalogFamily = "sensitive_mobile"
	TriggerCatalogFamilyDataChange      TriggerCatalogFamily = "data_change"
	TriggerCatalogFamilyComposite       TriggerCatalogFamily = "composite"
)

// TriggerCatalogEntry describes a platform-owned trigger type definition.
type TriggerCatalogEntry struct {
	Type                    string
	Family                  TriggerCatalogFamily
	ProviderClass           string
	Status                  TriggerCatalogStatus
	PayloadSchema           string
	MinimumClassification   string
	DefaultDenialVisibility string
	ReceiptPosture          string
	RequiredAuthorityInputs []string
	ProviderTrustLevel      int
	RateLimitProfile        string
	RetentionProfile        string
}

func (e TriggerCatalogEntry) DeveloperDeclarable() bool {
	return e.Status == TriggerCatalogStatusAvailable || e.Status == TriggerCatalogStatusDevOnly
}

// TriggerCatalog is the canonical trigger type catalog used by Go manifest validation.
var TriggerCatalog = []TriggerCatalogEntry{
	{
		Type:                    "platform.action.requested",
		Family:                  TriggerCatalogFamilyPlatformAction,
		ProviderClass:           "platform-ui",
		Status:                  TriggerCatalogStatusAvailable,
		PayloadSchema:           "kyzmet.trigger.platform-action-requested.v1",
		MinimumClassification:   "normal",
		DefaultDenialVisibility: "developer_visible",
		ReceiptPosture:          "user-and-developer",
		RequiredAuthorityInputs: []string{"capability", "consent"},
		ProviderTrustLevel:      0,
		RateLimitProfile:        "interactive",
		RetentionProfile:        "receipt",
	},
	{
		Type:                    "dev.test.event",
		Family:                  TriggerCatalogFamilyPlatformAction,
		ProviderClass:           "dev-test",
		Status:                  TriggerCatalogStatusDevOnly,
		PayloadSchema:           "kyzmet.trigger.dev-test-event.v1",
		MinimumClassification:   "public",
		DefaultDenialVisibility: "developer_visible",
		ReceiptPosture:          "developer",
		RequiredAuthorityInputs: []string{},
		ProviderTrustLevel:      0,
		RateLimitProfile:        "dev",
		RetentionProfile:        "ephemeral",
	},
	{
		Type:                    "time.schedule",
		Family:                  TriggerCatalogFamilySchedule,
		ProviderClass:           "schedule",
		Status:                  TriggerCatalogStatusAvailable,
		PayloadSchema:           "kyzmet.trigger.time-schedule.v1",
		MinimumClassification:   "public",
		DefaultDenialVisibility: "developer_visible",
		ReceiptPosture:          "audit-and-developer",
		RequiredAuthorityInputs: []string{"capability", "consent"},
		ProviderTrustLevel:      0,
		RateLimitProfile:        "background",
		RetentionProfile:        "receipt",
	},
	{
		Type:                    "connector.github.push",
		Family:                  TriggerCatalogFamilyConnector,
		ProviderClass:           "connector",
		Status:                  TriggerCatalogStatusAvailable,
		PayloadSchema:           "kyzmet.trigger.connector-github-push.v1",
		MinimumClassification:   "personal",
		DefaultDenialVisibility: "developer_visible",
		ReceiptPosture:          "audit-and-developer",
		RequiredAuthorityInputs: []string{"capability", "consent", "provider_trust", "credential_health"},
		ProviderTrustLevel:      1,
		RateLimitProfile:        "connector",
		RetentionProfile:        "receipt",
	},
	{
		Type:                    "connector.calendar.invite",
		Family:                  TriggerCatalogFamilyConnector,
		ProviderClass:           "connector",
		Status:                  TriggerCatalogStatusAvailable,
		PayloadSchema:           "kyzmet.trigger.connector-calendar-invite.v1",
		MinimumClassification:   "sensitive",
		DefaultDenialVisibility: "user_visible",
		ReceiptPosture:          "user-audit-redacted-developer",
		RequiredAuthorityInputs: []string{"capability", "consent", "provider_trust", "credential_health"},
		ProviderTrustLevel:      1,
		RateLimitProfile:        "connector",
		RetentionProfile:        "receipt",
	},
	{
		Type:                    "device.state.battery.charging",
		Family:                  TriggerCatalogFamilyDeviceState,
		ProviderClass:           "mobile-device-state",
		Status:                  TriggerCatalogStatusReserved,
		PayloadSchema:           "kyzmet.trigger.device-state-battery.v1",
		MinimumClassification:   "personal",
		DefaultDenialVisibility: "developer_visible",
		ReceiptPosture:          "audit",
		RequiredAuthorityInputs: []string{"os_permission"},
		ProviderTrustLevel:      1,
		RateLimitProfile:        "device-state",
		RetentionProfile:        "ephemeral",
	},
	{
		Type:                    "location.geofence.entered",
		Family:                  TriggerCatalogFamilySensitiveMobile,
		ProviderClass:           "mobile-location",
		Status:                  TriggerCatalogStatusReserved,
		PayloadSchema:           "kyzmet.trigger.location-geofence.v1",
		MinimumClassification:   "sensitive",
		DefaultDenialVisibility: "user_visible",
		ReceiptPosture:          "user-audit-redacted-developer",
		RequiredAuthorityInputs: []string{"capability", "consent", "os_permission"},
		ProviderTrustLevel:      1,
		RateLimitProfile:        "sensitive-mobile",
		RetentionProfile:        "redacted-receipt",
	},
	{
		Type:                    "health.record.updated",
		Family:                  TriggerCatalogFamilySensitiveMobile,
		ProviderClass:           "health",
		Status:                  TriggerCatalogStatusReserved,
		PayloadSchema:           "kyzmet.trigger.health-record-updated.v1",
		MinimumClassification:   "phi",
		DefaultDenialVisibility: "user_visible",
		ReceiptPosture:          "user-audit-redacted-developer",
		RequiredAuthorityInputs: []string{"capability", "consent", "os_permission"},
		ProviderTrustLevel:      1,
		RateLimitProfile:        "sensitive-mobile",
		RetentionProfile:        "redacted-receipt",
	},
	{
		Type:                    "composite.sequence",
		Family:                  TriggerCatalogFamilyComposite,
		ProviderClass:           "composite",
		Status:                  TriggerCatalogStatusReserved,
		PayloadSchema:           "kyzmet.trigger.composite-sequence.v1",
		MinimumClassification:   "derived",
		DefaultDenialVisibility: "user_visible",
		ReceiptPosture:          "audit",
		RequiredAuthorityInputs: []string{"capability", "consent", "standing_consent"},
		ProviderTrustLevel:      1,
		RateLimitProfile:        "composite",
		RetentionProfile:        "receipt",
	},
}

// TriggerCatalogEntryFor returns the catalog entry for a trigger type.
func TriggerCatalogEntryFor(triggerType string) (TriggerCatalogEntry, bool) {
	for _, entry := range TriggerCatalog {
		if entry.Type == triggerType {
			return entry, true
		}
	}
	return TriggerCatalogEntry{}, false
}

// Capabilities holds the app capabilities block. Always present in the manifest.
type Capabilities struct {
	A2aAgents []A2aAgentDeclaration `json:"a2a_agents,omitempty"`
	Topics    []TopicDecl           `json:"topics,omitempty"`
}

// AppManifest is a fully validated app manifest ready for packaging.
type AppManifest struct {
	App          AppMeta                     `json:"app"`
	Capabilities Capabilities                `json:"capabilities"`
	Frontend     *FrontendManifest           `json:"frontend,omitempty"`
	UI           *UiManifest                 `json:"ui,omitempty"`
	Components   []ComponentDesc             `json:"components,omitempty"`
	Triggers     []TriggerBindingDeclaration `json:"triggers,omitempty"`
}

// ToJSON serializes the manifest to JSON for use with packaging tooling.
func (m *AppManifest) ToJSON() (string, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

// ── Builder ────────────────────────────────────────────────────────────────

// ManifestBuilder builds an AppManifest with all validation run at Build() time.
type ManifestBuilder struct {
	appID           string
	appName         string
	appVersion      string
	vendorID        string
	vendorName      string
	vendorPublicKey string
	appDescription  *string
	components      []ComponentDesc
	capabilities    Capabilities
	frontend        FrontendManifest
	ui              UiManifest
	triggers        []TriggerBindingDeclaration
}

// NewManifest creates a ManifestBuilder with the required app identity fields.
func NewManifest(id, name, version, vendorID, vendorName, vendorPublicKey string) *ManifestBuilder {
	return &ManifestBuilder{
		appID:           id,
		appName:         name,
		appVersion:      version,
		vendorID:        vendorID,
		vendorName:      vendorName,
		vendorPublicKey: vendorPublicKey,
	}
}

// Description sets the optional app description.
func (b *ManifestBuilder) Description(desc string) *ManifestBuilder {
	b.appDescription = &desc
	return b
}

// Component adds a component descriptor.
func (b *ManifestBuilder) Component(c ComponentDesc) *ManifestBuilder {
	b.components = append(b.components, c)
	return b
}

// AddComponent adds a component descriptor (alias for Component).
func (b *ManifestBuilder) AddComponent(c ComponentDesc) *ManifestBuilder {
	return b.Component(c)
}

// WithCapabilities sets the capabilities block.
func (b *ManifestBuilder) WithCapabilities(c Capabilities) *ManifestBuilder {
	b.capabilities = c
	return b
}

// Frontend sets the frontend declaration.
func (b *ManifestBuilder) Frontend(f FrontendManifest) *ManifestBuilder {
	b.frontend = f
	return b
}

// UI sets the platform-rendered UI declaration.
func (b *ManifestBuilder) UI(ui UiManifest) *ManifestBuilder {
	b.ui = ui
	return b
}

// Trigger adds a trigger binding declaration.
func (b *ManifestBuilder) Trigger(t TriggerBindingDeclaration) *ManifestBuilder {
	b.triggers = append(b.triggers, t)
	return b
}

// AddFrontendSurface appends a frontend UI surface declaration.
func (b *ManifestBuilder) AddFrontendSurface(surface FrontendSurface) *ManifestBuilder {
	b.frontend.Surfaces = append(b.frontend.Surfaces, surface)
	return b
}

// Build runs all validation checks and returns an AppManifest on success.
func (b *ManifestBuilder) Build() (*AppManifest, error) {
	// ── Required app fields ──
	if strings.TrimSpace(b.appID) == "" {
		return nil, manifestErr("app.id is required and must not be empty")
	}
	if strings.TrimSpace(b.appName) == "" {
		return nil, manifestErr("app.name is required and must not be empty")
	}
	if strings.TrimSpace(b.appVersion) == "" {
		return nil, manifestErr("app.version is required and must not be empty")
	}
	if strings.TrimSpace(b.vendorID) == "" {
		return nil, manifestErr("app.vendor.id is required and must not be empty")
	}
	if !strings.HasPrefix(b.appID, b.vendorID+".") {
		return nil, manifestErr(fmt.Sprintf("app.id must be namespaced under app.vendor.id (%s)", b.vendorID))
	}
	if strings.TrimSpace(b.vendorName) == "" {
		return nil, manifestErr("app.vendor.name is required and must not be empty")
	}
	if strings.TrimSpace(b.vendorPublicKey) == "" {
		return nil, manifestErr("app.vendor.public_key is required and must not be empty")
	}

	seenFrontendSurfaces := map[FrontendSurface]bool{}
	for _, surface := range b.frontend.Surfaces {
		if surface != FrontendSurfaceChatWidget {
			return nil, manifestErr(fmt.Sprintf("frontend surface '%s' is not supported", surface))
		}
		if seenFrontendSurfaces[surface] {
			return nil, manifestErr(fmt.Sprintf("duplicate frontend surface: '%s'", surface))
		}
		seenFrontendSurfaces[surface] = true
	}
	if err := validateNavDescriptors(b.ui.Nav, "ui.nav", b.appID, 0); err != nil {
		return nil, err
	}
	if err := validateTriggers(b.triggers, b.appID); err != nil {
		return nil, err
	}

	// ── Component ID uniqueness ──
	seen := make(map[string]bool, len(b.components))
	for _, c := range b.components {
		if seen[c.ID] {
			return nil, manifestErr(fmt.Sprintf("duplicate component id: '%s'", c.ID))
		}
		seen[c.ID] = true
	}

	// ── script / trigger constraints ──
	for _, c := range b.components {
		if strings.TrimSpace(c.Wasm) == "" {
			return nil, manifestErr(fmt.Sprintf("component '%s' requires a 'wasm' field", c.ID))
		}
		switch c.Kind {
		case ComponentKindScript:
			if c.Signature == nil {
				return nil, manifestErr(fmt.Sprintf("script component '%s' requires a 'signature' field", c.ID))
			}
		case ComponentKindTrigger:
			if len(c.Listen) == 0 {
				return nil, manifestErr(fmt.Sprintf("trigger component '%s' requires at least one 'listen' port", c.ID))
			}
		}
	}

	// ── Trigger port validation ──
	for _, c := range b.components {
		if c.Kind == ComponentKindTrigger {
			for _, port := range c.Listen {
				if err := validateTriggerPort(port); err != nil {
					return nil, err
				}
			}
		}
	}

	var frontend *FrontendManifest
	if !frontendManifestEmpty(b.frontend) {
		copy := b.frontend
		frontend = &copy
	}
	var ui *UiManifest
	if !uiManifestEmpty(b.ui) {
		copy := b.ui
		ui = &copy
	}

	return &AppManifest{
		App: AppMeta{
			ID:          b.appID,
			Name:        b.appName,
			Version:     b.appVersion,
			Vendor:      VendorMeta{ID: b.vendorID, Name: b.vendorName, PublicKey: b.vendorPublicKey},
			Description: b.appDescription,
		},
		Capabilities: b.capabilities,
		Frontend:     frontend,
		UI:           ui,
		Components:   b.components,
		Triggers:     b.triggers,
	}, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

func frontendManifestEmpty(f FrontendManifest) bool {
	return f.Entry == "" && f.Mode == "" && len(f.Surfaces) == 0
}

func uiManifestEmpty(ui UiManifest) bool {
	return ui.Surface == "" && len(ui.Nav) == 0
}

func validateNavDescriptors(items []NavDescriptor, path string, appID string, depth int) error {
	if depth > 5 {
		return manifestErr(fmt.Sprintf("%s exceeds maximum nav nesting depth", path))
	}
	ids := map[string]bool{}
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		if !validNavID(item.ID, appID) {
			return manifestErr(fmt.Sprintf("%s.id must be namespaced under app.id (%s)", itemPath, appID))
		}
		if ids[item.ID] {
			return manifestErr(fmt.Sprintf("%s.id duplicates another ui.nav id: %s", itemPath, item.ID))
		}
		ids[item.ID] = true
		if err := validateIconRef(item.Icon, itemPath+".icon"); err != nil {
			return err
		}
		if err := validateIconRef(item.IconFallback, itemPath+".icon_fallback"); err != nil {
			return err
		}
		if item.Type != "divider" && strings.TrimSpace(item.Label) == "" {
			return manifestErr(fmt.Sprintf("%s.label is required", itemPath))
		}
		switch item.Type {
		case "static":
			if err := validateNavTarget(item.Target, itemPath+".target"); err != nil {
				return err
			}
		case "label", "divider":
		case "item", "list":
			if err := validateNavPull(item.Pull, itemPath+".pull"); err != nil {
				return err
			}
		default:
			return manifestErr(fmt.Sprintf("%s.type is invalid: %s", itemPath, item.Type))
		}
		if item.Type == "list" {
			if err := validateNavDescriptors(item.Items, itemPath+".items", appID, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateNavTarget(target *NavTarget, path string) error {
	if target == nil {
		return manifestErr(fmt.Sprintf("%s is required", path))
	}
	switch target.Kind {
	case "route":
		if !validRoute(target.Route) || reservedRoute(target.Route) {
			return manifestErr(fmt.Sprintf("%s.route must be an app-local route", path))
		}
	case "platform":
		if target.Route != "app_detail" {
			return manifestErr(fmt.Sprintf("%s.route contains unknown platform route: %s", path, target.Route))
		}
	default:
		return manifestErr(fmt.Sprintf("%s.kind contains unknown target kind: %s", path, target.Kind))
	}
	return nil
}

func validateNavPull(pull *NavPull, path string) error {
	if pull == nil {
		return manifestErr(fmt.Sprintf("%s is required", path))
	}
	if !validNavTopic(pull.Topic) {
		return manifestErr(fmt.Sprintf("%s.topic must be a relative nav topic", path))
	}
	switch pull.Kind {
	case "wasm":
		if !validExportName(pull.Export) {
			return manifestErr(fmt.Sprintf("%s.export must be a valid WASM export name", path))
		}
	case "platform":
		if pull.Provider != "sphere" && pull.Provider != "app" {
			return manifestErr(fmt.Sprintf("%s.provider contains unknown platform provider: %s", path, pull.Provider))
		}
	default:
		return manifestErr(fmt.Sprintf("%s.kind contains unknown pull kind: %s", path, pull.Kind))
	}
	return nil
}

func validateIconRef(icon, path string) error {
	if icon == "" {
		return nil
	}
	if !(strings.HasPrefix(icon, "lucide:") || strings.HasPrefix(icon, "sphere:") || strings.HasPrefix(icon, "app:")) {
		return manifestErr(fmt.Sprintf("%s must use lucide:, sphere:, or app:", path))
	}
	if strings.TrimSpace(strings.SplitN(icon, ":", 2)[1]) == "" {
		return manifestErr(fmt.Sprintf("%s must include an icon name", path))
	}
	return nil
}

func validNavID(id string, appID string) bool {
	if id == "settings" || id == "security" || id == "privacy" || id == "connectors" || id == "consents" || id == "shell" ||
		strings.HasPrefix(id, "shell.") || strings.HasPrefix(id, "shell:") || len(id) == 0 || len(id) > 80 ||
		!strings.HasPrefix(id, appID+".") {
		return false
	}
	for i, r := range id {
		if i == 0 && (r < 'a' || r > 'z') {
			return false
		}
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-') {
			return false
		}
	}
	return true
}

func validRoute(route string) bool {
	if route == "" {
		return false
	}
	for i, r := range route {
		if i == 0 && !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ':' || r == '.' || r == '_' || r == '/' || r == '-') {
			return false
		}
	}
	return true
}

func reservedRoute(route string) bool {
	for _, prefix := range []string{"settings", "security", "privacy", "connectors", "consents", "shell:"} {
		if strings.HasPrefix(route, prefix) {
			return true
		}
	}
	return strings.Contains(route, "://")
}

func validExportName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 && !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == '_') {
			return false
		}
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == ':' || r == '-') {
			return false
		}
	}
	return true
}

func validNavTopic(topic string) bool {
	if topic == "" || strings.HasPrefix(topic, "/") {
		return false
	}
	for _, part := range strings.Split(topic, "/") {
		if part == "" {
			return false
		}
		for _, segment := range strings.Split(part, ".") {
			if !validTopicSegment(segment) {
				return false
			}
		}
	}
	return true
}

func validTopicSegment(segment string) bool {
	if segment == "" {
		return false
	}
	for i, r := range segment {
		if i == 0 && (r < 'a' || r > 'z') {
			return false
		}
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

func validateTriggers(triggers []TriggerBindingDeclaration, appID string) error {
	seen := make(map[string]bool, len(triggers))
	for idx, t := range triggers {
		path := fmt.Sprintf("triggers[%d]", idx)
		if !validTriggerBindingID(t.ID) {
			return manifestErr(fmt.Sprintf("%s.id must be a valid trigger binding id", path))
		}
		if seen[t.ID] {
			return manifestErr(fmt.Sprintf("%s.id duplicates another trigger: %s", path, t.ID))
		}
		seen[t.ID] = true
		if !validTriggerType(t.Type) {
			return manifestErr(fmt.Sprintf("%s.type must be a known trigger type namespace", path))
		}
		if t.Pattern != nil {
			if !validTriggerPattern(*t.Pattern) {
				return manifestErr(fmt.Sprintf("%s.pattern must be a dotted subject pattern", path))
			}
			if !strings.HasPrefix(*t.Pattern, appID+".") {
				return manifestErr(fmt.Sprintf("%s.pattern must be namespaced under app.id (%s)", path, appID))
			}
		}
		if strings.TrimSpace(t.Component) == "" {
			return manifestErr(fmt.Sprintf("%s.component is required", path))
		}
		if !validExportName(t.ExportName) {
			return manifestErr(fmt.Sprintf("%s.export_name must be a valid WASM export name", path))
		}
		if t.Schedule != nil && t.Type != "time.schedule" {
			return manifestErr(fmt.Sprintf("%s.schedule is only valid for type time.schedule", path))
		}
		if t.Connector != nil {
			provider, ok := connectorTriggerProvider(t.Type)
			if !ok {
				return manifestErr(fmt.Sprintf("%s.connector is only valid for connector.<provider>.<event> trigger types", path))
			}
			if t.Connector.Provider == "" {
				return manifestErr(fmt.Sprintf("%s.connector.provider is required", path))
			}
			if t.Connector.Provider != provider {
				return manifestErr(fmt.Sprintf("%s.connector.provider must match trigger type provider (%s)", path, provider))
			}
		}
		if t.Type == "time.schedule" {
			if err := validateScheduleConfig(t.Schedule, path+".schedule"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateScheduleConfig(schedule *ScheduleConfig, path string) error {
	if schedule == nil {
		return manifestErr(fmt.Sprintf("%s is required for type time.schedule", path))
	}
	if !validScheduleCron(schedule.Cron) {
		return manifestErr(fmt.Sprintf("%s.cron must be a valid five-field cron expression", path))
	}
	if schedule.Timezone != "" && !validScheduleTimezone(schedule.Timezone) {
		return manifestErr(fmt.Sprintf("%s.timezone must be UTC or an IANA timezone identifier", path))
	}
	return nil
}

// validTriggerBindingID checks the pattern ^[a-z][a-z0-9._-]{0,79}$.
func validTriggerBindingID(id string) bool {
	if len(id) == 0 || len(id) > 80 {
		return false
	}
	for i, r := range id {
		if i == 0 && (r < 'a' || r > 'z') {
			return false
		}
		if i > 0 && !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// validTriggerType checks the cataloged, developer-declarable trigger types.
func validTriggerType(t string) bool {
	entry, ok := TriggerCatalogEntryFor(t)
	return ok && entry.DeveloperDeclarable()
}

func connectorTriggerProvider(t string) (string, bool) {
	parts := strings.Split(t, ".")
	if len(parts) < 3 || parts[0] != "connector" {
		return "", false
	}
	return parts[1], true
}

func validScheduleCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}
	ranges := [][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	for i, field := range fields {
		if !validCronField(field, ranges[i][0], ranges[i][1]) {
			return false
		}
	}
	return true
}

func validScheduleTimezone(timezone string) bool {
	if timezone == "UTC" {
		return true
	}
	parts := strings.Split(timezone, "/")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '+') {
				return false
			}
		}
	}
	return true
}

func validCronField(field string, min int, max int) bool {
	for _, atom := range strings.Split(field, ",") {
		if !validCronAtom(atom, min, max) {
			return false
		}
	}
	return true
}

func validCronAtom(atom string, min int, max int) bool {
	if atom == "" {
		return false
	}
	stepParts := strings.Split(atom, "/")
	if len(stepParts) > 2 {
		return false
	}
	if len(stepParts) == 2 {
		step, err := strconv.Atoi(stepParts[1])
		if err != nil || step <= 0 {
			return false
		}
	}

	base := stepParts[0]
	if base == "*" {
		return true
	}
	bounds := strings.Split(base, "-")
	switch len(bounds) {
	case 1:
		value, err := strconv.Atoi(bounds[0])
		return err == nil && value >= min && value <= max
	case 2:
		start, startErr := strconv.Atoi(bounds[0])
		end, endErr := strconv.Atoi(bounds[1])
		return startErr == nil && endErr == nil && start >= min && end <= max && start <= end
	default:
		return false
	}
}

func validDottedIdentifier(t string) bool {
	if t == "" {
		return false
	}
	for _, part := range strings.Split(t, ".") {
		if len(part) == 0 {
			return false
		}
		for i, r := range part {
			if i == 0 && (r < 'a' || r > 'z') {
				return false
			}
			if i > 0 && !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

// validTriggerPattern accepts dotted identifiers with an optional final "*" segment.
func validTriggerPattern(p string) bool {
	if p == "" {
		return false
	}
	parts := strings.Split(p, ".")
	for i, part := range parts {
		if len(part) == 0 {
			return false
		}
		if part == "*" {
			return i == len(parts)-1
		}
		for j, r := range part {
			if j == 0 && (r < 'a' || r > 'z') {
				return false
			}
			if j > 0 && !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

func validateTriggerPort(port string) *ManifestError {
	parts := strings.SplitN(port, ".", 3)
	if len(parts) != 3 {
		return manifestErr(fmt.Sprintf("trigger port '%s' must follow <source>.<name>.<event> format", port))
	}
	source := parts[0]
	switch source {
	case "connector", "vault", "host", "timer":
	default:
		return manifestErr(fmt.Sprintf(
			"trigger port '%s': unknown source '%s' (allowed: connector, vault, host, timer)",
			port, source,
		))
	}
	return nil
}
