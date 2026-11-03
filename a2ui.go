// protobuf-policy-file: allowed external (2 sites) [9958203ec35d] — A2UI is an external agent-to-UI format (Ross, D4 in kyzmet-workspace docs/design/host-guest-envelopes.md); guests build descriptors for ui.prompt-user.
// A2UI component descriptor builder.
//
// Provides construction-time validation against the A2UI allowed subset.
// Mirrors the validation in sandbox/runtime/ipc/src/a2ui.rs so that
// skills catch violations before they reach the sandbox.
//
// Allowed component types: Text, Card, List, Table, Form, Media, Chart,
// Progress, Alert, Divider, Badge, Action.
package raw

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// ComponentType is the set of A2UI component types accepted by the sandbox validator.
type ComponentType string

const (
	ComponentTypeText     ComponentType = "Text"
	ComponentTypeCard     ComponentType = "Card"
	ComponentTypeList     ComponentType = "List"
	ComponentTypeTable    ComponentType = "Table"
	ComponentTypeForm     ComponentType = "Form"
	ComponentTypeMedia    ComponentType = "Media"
	ComponentTypeChart    ComponentType = "Chart"
	ComponentTypeProgress ComponentType = "Progress"
	ComponentTypeAlert    ComponentType = "Alert"
	ComponentTypeDivider  ComponentType = "Divider"
	ComponentTypeBadge    ComponentType = "Badge"
	ComponentTypeAction   ComponentType = "Action"
)

// A2uiError is returned when a component or URL fails validation.
type A2uiError struct {
	msg string
}

func (e *A2uiError) Error() string { return "a2ui validation error: " + e.msg }

func a2uiErr(msg string) *A2uiError { return &A2uiError{msg: msg} }

// ComponentBuilder constructs and validates an A2UI component.
// Errors accumulate; call Build() to retrieve the result or the first error.
type ComponentBuilder struct {
	componentType ComponentType
	props         map[string]interface{}
	children      []*ComponentBuilder
	err           error
}

// NewComponent creates a new builder for the given component type.
func NewComponent(t ComponentType) *ComponentBuilder {
	return &ComponentBuilder{
		componentType: t,
		props:         make(map[string]interface{}),
	}
}

// Text sets the primary text/content property.
func (b *ComponentBuilder) Text(value string) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if err := checkStringContent("text", value); err != nil {
		b.err = err
		return b
	}
	b.props["text"] = value
	return b
}

// Title sets the title property.
func (b *ComponentBuilder) Title(value string) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if err := checkStringContent("title", value); err != nil {
		b.err = err
		return b
	}
	b.props["title"] = value
	return b
}

// Label sets the label property.
func (b *ComponentBuilder) Label(value string) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if err := checkStringContent("label", value); err != nil {
		b.err = err
		return b
	}
	b.props["label"] = value
	return b
}

// Src sets the src property (Media, Chart). Only asset:// URLs or relative paths allowed.
func (b *ComponentBuilder) Src(url string) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if err := checkURL("src", url); err != nil {
		b.err = err
		return b
	}
	b.props["src"] = url
	return b
}

// Href sets the href property (Action). Only asset:// URLs or relative paths allowed.
func (b *ComponentBuilder) Href(url string) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if err := checkURL("href", url); err != nil {
		b.err = err
		return b
	}
	b.props["href"] = url
	return b
}

// ActionURL sets the action property (Form). Only asset:// URLs or relative paths allowed.
func (b *ComponentBuilder) ActionURL(url string) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if err := checkURL("action", url); err != nil {
		b.err = err
		return b
	}
	b.props["action"] = url
	return b
}

// Poster sets the poster property (Media). Only asset:// URLs or relative paths allowed.
func (b *ComponentBuilder) Poster(url string) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if err := checkURL("poster", url); err != nil {
		b.err = err
		return b
	}
	b.props["poster"] = url
	return b
}

// Severity sets the severity for Alert components.
func (b *ComponentBuilder) Severity(s AlertSeverity) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	b.props["severity"] = string(s)
	return b
}

// ValueFloat sets a numeric float value (Progress, Chart, etc.).
// Returns an error builder if v is not finite (NaN or infinity).
func (b *ComponentBuilder) ValueFloat(v float64) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		b.err = a2uiErr("value must be finite (not NaN or infinity)")
		return b
	}
	b.props["value"] = v
	return b
}

// ValueInt sets an integer value.
func (b *ComponentBuilder) ValueInt(v int64) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	b.props["value"] = v
	return b
}

// Child adds a child component.
func (b *ComponentBuilder) Child(child *ComponentBuilder) *ComponentBuilder {
	if b.err != nil {
		return b
	}
	b.children = append(b.children, child)
	return b
}

// Children adds multiple child components.
func (b *ComponentBuilder) Children(children ...*ComponentBuilder) *ComponentBuilder {
	for _, c := range children {
		b.Child(c)
	}
	return b
}

// Component is a validated, immutable A2UI component ready to pass to UiPresent.
type Component struct {
	data map[string]interface{}
}

// Build validates the builder and returns a Component or the first accumulated error.
func (b *ComponentBuilder) Build() (*Component, error) {
	if b.err != nil {
		return nil, b.err
	}
	data, err := buildComponentData(b)
	if err != nil {
		return nil, err
	}
	return &Component{data: data}, nil
}

func buildComponentData(b *ComponentBuilder) (map[string]interface{}, error) {
	if b.err != nil {
		return nil, b.err
	}
	m := make(map[string]interface{}, len(b.props)+2)
	m["type"] = string(b.componentType)
	for k, v := range b.props {
		m[k] = v
	}
	if len(b.children) > 0 {
		children := make([]map[string]interface{}, 0, len(b.children))
		for _, c := range b.children {
			cd, err := buildComponentData(c)
			if err != nil {
				return nil, err
			}
			children = append(children, cd)
		}
		m["children"] = children
	}
	return m, nil
}

// ToJSON serializes the component to a JSON string for use with UiPresent.
func (c *Component) ToJSON() (string, error) {
	b, err := json.Marshal(c.data)
	return string(b), err
}

// Descriptor is a top-level A2UI descriptor containing a list of root components.
type Descriptor struct {
	components []*Component
}

// NewDescriptor creates an empty Descriptor.
func NewDescriptor() *Descriptor { return &Descriptor{} }

// Push adds a root component to the descriptor.
func (d *Descriptor) Push(c *Component) *Descriptor {
	d.components = append(d.components, c)
	return d
}

// ToJSON serializes the descriptor to JSON for use with UiPresent.
func (d *Descriptor) ToJSON() (string, error) {
	comps := make([]map[string]interface{}, len(d.components))
	for i, c := range d.components {
		comps[i] = c.data
	}
	b, err := json.Marshal(map[string]interface{}{"components": comps})
	return string(b), err
}

// ── Shorthand constructors ──────────────────────────────────────────────────

// TextComponent creates a validated Text component.
func TextComponent(content string) (*Component, error) {
	return NewComponent(ComponentTypeText).Text(content).Build()
}

// CardComponent creates a validated Card component.
func CardComponent(title string) (*Component, error) {
	return NewComponent(ComponentTypeCard).Title(title).Build()
}

// AlertComponent creates a validated Alert component.
func AlertComponent(msg string, severity AlertSeverity) (*Component, error) {
	return NewComponent(ComponentTypeAlert).Text(msg).Severity(severity).Build()
}

// ProgressComponent creates a validated Progress component (value 0.0–1.0).
func ProgressComponent(value float64, label string) (*Component, error) {
	return NewComponent(ComponentTypeProgress).Label(label).ValueFloat(value).Build()
}

// ── Validation helpers ──────────────────────────────────────────────────────

func checkStringContent(field, value string) error {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "<iframe") {
		return a2uiErr(fmt.Sprintf("field '%s': embedded HTML/JS not allowed", field))
	}
	return nil
}

func checkURL(field, value string) error {
	if isBlockedURL(value) {
		return a2uiErr(fmt.Sprintf("field '%s': blocked URL", field))
	}
	return checkStringContent(field, value)
}

// isBlockedURL mirrors sandbox/runtime/ipc/src/a2ui.rs::is_blocked_url.
func isBlockedURL(s string) bool {
	if strings.HasPrefix(s, "//") {
		return true
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b <= 0x20 || b == 0x7f {
			return true
		}
	}
	if strings.ContainsRune(s, '\uFEFF') {
		return true
	}
	scheme, ok := extractURLScheme(s)
	if ok {
		return scheme != "asset"
	}
	return false
}

func extractURLScheme(s string) (string, bool) {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return "", false
	}
	candidate := s[:idx]
	if len(candidate) == 0 {
		return "", false
	}
	c0 := candidate[0]
	if !((c0 >= 'a' && c0 <= 'z') || (c0 >= 'A' && c0 <= 'Z')) {
		return "", false
	}
	for _, c := range candidate[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.') {
			return "", false
		}
	}
	return strings.ToLower(candidate), true
}
