// Package raw is the Go guest SDK's host API. On wasip2, each call is a typed
// Component Model import generated from kyzmet-host.wit (internal/wit); other
// targets compile against stubs that return ErrNotImplemented, so host-side
// tests build. Go guests are wasip2 components: build with
// `tinygo build -target=wasip2`.

package raw

// ── Errors ──

type HostError struct {
	Code int32
	Msg  string
}

func (e *HostError) Error() string {
	return e.Msg
}

var (
	ErrNotGranted          = &HostError{Code: -1, Msg: "capability not granted"}
	ErrSensitivityExceeded = &HostError{Code: -4, Msg: "sensitivity exceeded"}
	ErrApprovalRequired    = &HostError{Code: -5, Msg: "approval required"}
	ErrSerializationError  = &HostError{Code: -10, Msg: "serialization error"}
	ErrRateLimited         = &HostError{Code: -11, Msg: "rate limited"}
	ErrNotImplemented      = &HostError{Code: -20, Msg: "not implemented"}
	ErrDispatchError       = &HostError{Code: -30, Msg: "dispatch error"}
)

// ── Types ──

type EntryId = string

type Sensitivity string

const (
	SensitivityStandard Sensitivity = "standard"
	SensitivityElevated Sensitivity = "elevated"
	SensitivityCritical Sensitivity = "critical"
)

type AlertSeverity string

const (
	AlertSeverityInfo    AlertSeverity = "info"
	AlertSeverityWarning AlertSeverity = "warning"
	AlertSeverityError   AlertSeverity = "error"
	AlertSeveritySuccess AlertSeverity = "success"
)

type CreateDocRequest struct {
	Key         string
	Value       string
	Sensitivity Sensitivity
}

type GetDocRequest struct {
	Key         string
	Sensitivity Sensitivity
}

type ListDocsRequest struct {
	Sensitivity Sensitivity
}

// GraphPropertyValue is a typed graph property value: GraphString,
// GraphLong, GraphDouble, GraphBool or GraphBytes.
type GraphPropertyValue interface{ isGraphPropertyValue() }

type (
	GraphString string
	GraphLong   int64
	GraphDouble float64
	GraphBool   bool
	GraphBytes  []byte
)

func (GraphString) isGraphPropertyValue() {}
func (GraphLong) isGraphPropertyValue()   {}
func (GraphDouble) isGraphPropertyValue() {}
func (GraphBool) isGraphPropertyValue()   {}
func (GraphBytes) isGraphPropertyValue()  {}

// GraphProperty is one named property of a node or an edge. A property list
// names each property once; the host rejects a list that repeats a name.
type GraphProperty struct {
	Name  string
	Value GraphPropertyValue
}

// GraphNode is a node as the host stores it. Properties are in name order;
// times are epoch milliseconds.
type GraphNode struct {
	NodeId     string
	NodeType   string
	BlobId     *string
	Properties []GraphProperty
	CreatedAt  int64
	UpdatedAt  int64
}

// GraphSetNodeOutcome is how a node write ended.
type GraphSetNodeOutcome string

const (
	GraphSetNodeOk GraphSetNodeOutcome = "ok"
	// GraphSetNodeInvalidBlobRef: the blob is missing or belongs to another
	// vault. Nothing was written.
	GraphSetNodeInvalidBlobRef GraphSetNodeOutcome = "invalid-blob-ref"
)

// GraphSetEdgeOutcome is how an edge write ended.
type GraphSetEdgeOutcome string

const (
	GraphSetEdgeOk GraphSetEdgeOutcome = "ok"
	// GraphSetEdgeInvalidNodeRef: an endpoint does not exist. Nothing was
	// written.
	GraphSetEdgeInvalidNodeRef GraphSetEdgeOutcome = "invalid-node-ref"
)

// DocumentMetadata is what the host keeps about a vault document outside its
// content. Times are epoch milliseconds.
type DocumentMetadata struct {
	CreatedAt int64
	UpdatedAt int64
	// The component that created the document.
	CreatedBy string
	Tags      []string
	// Absent when the document never expires.
	ExpiresAt *int64
}

// Document is a vault document as get-doc returns it. Content is exactly what
// was stored; its schema is the app's until kyzmet-ai/kyzmet-workspace#2584.
type Document struct {
	Key         string
	Sensitivity Sensitivity
	SchemaId    *string
	Content     string
	Metadata    DocumentMetadata
}

type CreateNodeRequest struct {
	NodeType   string
	NodeId     string
	Properties []GraphProperty
	// BlobId optionally names a document in the same vault the node references.
	BlobId *string
}

type DeleteNodesRequest struct {
	// NodeIds are deleted with their edges; an ID with no node is skipped.
	NodeIds []string
}

type ListNodesOfTypeRequest struct {
	NodeType string
}

type CreateEdgeRequest struct {
	FromId     string
	ToId       string
	EdgeType   string
	Properties []GraphProperty
}

// ── Host calls ──
// An unknown Sensitivity, or a graph property with a nil value, is rejected
// with ErrSerializationError before any host call.

func CreateDoc(req CreateDocRequest) (*EntryId, error) {
	id, err := createDoc(req)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func GetDoc(req GetDocRequest) (*Document, error) {
	document, err := getDoc(req)
	if err != nil {
		return nil, err
	}
	return &document, nil
}

func ListDocs(req ListDocsRequest) ([]string, error) { return listDocs(req) }

func CreateNode(req CreateNodeRequest) (GraphSetNodeOutcome, error) {
	if err := validateGraphProperties(req.Properties); err != nil {
		return "", err
	}
	return createNode(req)
}

func DeleteNodes(req DeleteNodesRequest) error { return deleteNodes(req) }

func ListNodesOfType(req ListNodesOfTypeRequest) ([]GraphNode, error) {
	return listNodesOfType(req)
}

// GetNode reads the node with ID nodeID, whatever its type, or nil when the
// vault has no node with that ID. Node IDs are unique across node types.
func GetNode(nodeID string) (*GraphNode, error) { return getNode(nodeID) }

func CreateEdge(req CreateEdgeRequest) (GraphSetEdgeOutcome, error) {
	if err := validateGraphProperties(req.Properties); err != nil {
		return "", err
	}
	return createEdge(req)
}

// validateGraphProperties rejects a property with a nil value before any host
// call: the WIT has no empty value to send.
func validateGraphProperties(properties []GraphProperty) error {
	for _, property := range properties {
		if property.Value == nil {
			return ErrSerializationError
		}
	}
	return nil
}

// ── Exports ──

type BridgeFetchRequest struct {
	Path   string
	Method string
	Body   *string
}

type BridgeFetchResponse struct {
	Status uint16
	Body   *string
}

// BridgeFetchHandleHandler serves the bridge-fetch export; the host calls it.
type BridgeFetchHandleHandler interface {
	Handle(request BridgeFetchRequest) (BridgeFetchResponse, error)
}

var bridgeFetchHandler BridgeFetchHandleHandler // nolint:gochecknoglobals

func RegisterBridgeFetchHandleHandler(h BridgeFetchHandleHandler) {
	bridgeFetchHandler = h
}
