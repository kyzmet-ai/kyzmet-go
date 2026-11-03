//go:build wasip2

package raw

import (
	bridgefetch "kyzmet.ai/go/world/app/kyzmet/host/bridge-fetch"
	"kyzmet.ai/go/world/app/kyzmet/host/graph"
	"kyzmet.ai/go/world/app/kyzmet/host/types"
	"kyzmet.ai/go/world/app/kyzmet/host/vault"
	"go.bytecodealliance.org/cm"
)

func init() {
	bridgefetch.Exports.Handle = func(request bridgefetch.BridgeFetchRequest) cm.Result[
		bridgefetch.BridgeFetchResponseShape,
		bridgefetch.BridgeFetchResponse,
		bridgefetch.HostError,
	] {
		var result cm.Result[
			bridgefetch.BridgeFetchResponseShape,
			bridgefetch.BridgeFetchResponse,
			bridgefetch.HostError,
		]
		if bridgeFetchHandler == nil {
			result.SetErr(types.HostErrorNotImplemented)
			return result
		}
		response, err := bridgeFetchHandler.Handle(BridgeFetchRequest{
			Path: request.Path, Method: request.Method, Body: request.Body.Some(),
		})
		if err != nil {
			result.SetErr(witError(err))
			return result
		}
		witResponse := bridgefetch.BridgeFetchResponse{Status: response.Status, Body: cm.None[string]()}
		if response.Body != nil {
			witResponse.Body = cm.Some(*response.Body)
		}
		result.SetOK(witResponse)
		return result
	}
}

// witError maps an SDK error returned by a handler onto the WIT host-error enum.
func witError(err error) types.HostError {
	switch err {
	case ErrNotGranted:
		return types.HostErrorNotGranted
	case ErrSensitivityExceeded:
		return types.HostErrorSensitivityExceeded
	case ErrApprovalRequired:
		return types.HostErrorApprovalRequired
	case ErrSerializationError:
		return types.HostErrorSerializationError
	case ErrRateLimited:
		return types.HostErrorRateLimited
	case ErrNotImplemented:
		return types.HostErrorNotImplemented
	}
	return types.HostErrorDispatchError
}

// sdkError maps a WIT host error onto the SDK's error values.
func sdkError(err types.HostError) error {
	switch err {
	case types.HostErrorNotGranted:
		return ErrNotGranted
	case types.HostErrorSensitivityExceeded:
		return ErrSensitivityExceeded
	case types.HostErrorApprovalRequired:
		return ErrApprovalRequired
	case types.HostErrorSerializationError:
		return ErrSerializationError
	case types.HostErrorRateLimited:
		return ErrRateLimited
	case types.HostErrorNotImplemented:
		return ErrNotImplemented
	}
	return ErrDispatchError
}

func witSensitivity(value Sensitivity) (types.Sensitivity, error) {
	switch value {
	case SensitivityStandard:
		return types.SensitivityStandard, nil
	case SensitivityElevated:
		return types.SensitivityElevated, nil
	case SensitivityCritical:
		return types.SensitivityCritical, nil
	}
	return 0, ErrSerializationError
}

func sdkSensitivity(value types.Sensitivity) (Sensitivity, error) {
	switch value {
	case types.SensitivityStandard:
		return SensitivityStandard, nil
	case types.SensitivityElevated:
		return SensitivityElevated, nil
	case types.SensitivityCritical:
		return SensitivityCritical, nil
	}
	return "", ErrSerializationError
}

func unwrap[Shape, OK any](result cm.Result[Shape, OK, types.HostError]) (OK, error) {
	value, hostError, isError := result.Result()
	if isError {
		return value, sdkError(hostError)
	}
	return value, nil
}

func createDoc(req CreateDocRequest) (EntryId, error) {
	sensitivity, err := witSensitivity(req.Sensitivity)
	if err != nil {
		return "", err
	}
	id, err := unwrap(vault.CreateDoc(req.Key, req.Value, sensitivity))
	return EntryId(id), err
}

func getDoc(req GetDocRequest) (Document, error) {
	sensitivity, err := witSensitivity(req.Sensitivity)
	if err != nil {
		return Document{}, err
	}
	document, err := unwrap(vault.GetDoc(req.Key, sensitivity))
	if err != nil {
		return Document{}, err
	}
	stored, err := sdkSensitivity(document.Sensitivity)
	if err != nil {
		return Document{}, err
	}
	metadata := document.Metadata
	return Document{
		Key:         document.DocumentID,
		Sensitivity: stored,
		SchemaId:    document.SchemaID.Some(),
		Content:     document.Content,
		Metadata: DocumentMetadata{
			CreatedAt: metadata.CreatedAt,
			UpdatedAt: metadata.UpdatedAt,
			CreatedBy: string(metadata.CreatedBy),
			Tags:      metadata.Tags.Slice(),
			ExpiresAt: metadata.ExpiresAt.Some(),
		},
	}, nil
}

func listDocs(req ListDocsRequest) ([]string, error) {
	sensitivity, err := witSensitivity(req.Sensitivity)
	if err != nil {
		return nil, err
	}
	keys, err := unwrap(vault.ListDocs(sensitivity))
	if err != nil {
		return nil, err
	}
	return keys.Slice(), nil
}

func createNode(req CreateNodeRequest) (GraphSetNodeOutcome, error) {
	properties, err := witGraphProperties(req.Properties)
	if err != nil {
		return "", err
	}
	blobID := cm.None[string]()
	if req.BlobId != nil {
		blobID = cm.Some(*req.BlobId)
	}
	outcome, err := unwrap(graph.CreateNode(req.NodeType, req.NodeId, properties, blobID))
	if err != nil {
		return "", err
	}
	switch outcome {
	case graph.GraphSetNodeOutcomeOK:
		return GraphSetNodeOk, nil
	case graph.GraphSetNodeOutcomeInvalidBlobRef:
		return GraphSetNodeInvalidBlobRef, nil
	}
	// An outcome this SDK does not know: fail closed rather than report a write.
	return "", ErrDispatchError
}

func deleteNodes(req DeleteNodesRequest) error {
	_, err := unwrap(graph.DeleteNodes(cm.ToList(req.NodeIds)))
	return err
}

func listNodesOfType(req ListNodesOfTypeRequest) ([]GraphNode, error) {
	nodes, err := unwrap(graph.ListNodesOfType(req.NodeType))
	if err != nil {
		return nil, err
	}
	out := make([]GraphNode, 0, nodes.Len())
	for _, node := range nodes.Slice() {
		out = append(out, sdkGraphNode(node))
	}
	return out, nil
}

func getNode(nodeID string) (*GraphNode, error) {
	found, err := unwrap(graph.GetNode(nodeID))
	if err != nil {
		return nil, err
	}
	node := found.Some()
	if node == nil {
		return nil, nil
	}
	out := sdkGraphNode(*node)
	return &out, nil
}

func sdkGraphNode(node graph.GraphNode) GraphNode {
	var blobID *string
	if id := node.BlobID.Some(); id != nil {
		value := *id
		blobID = &value
	}
	return GraphNode{
		NodeId:     node.NodeID,
		NodeType:   node.NodeType,
		BlobId:     blobID,
		Properties: sdkGraphProperties(node.Properties),
		CreatedAt:  node.CreatedAt,
		UpdatedAt:  node.UpdatedAt,
	}
}

func createEdge(req CreateEdgeRequest) (GraphSetEdgeOutcome, error) {
	properties, err := witGraphProperties(req.Properties)
	if err != nil {
		return "", err
	}
	outcome, err := unwrap(graph.CreateEdge(req.FromId, req.ToId, req.EdgeType, properties))
	if err != nil {
		return "", err
	}
	switch outcome {
	case graph.GraphSetEdgeOutcomeOK:
		return GraphSetEdgeOk, nil
	case graph.GraphSetEdgeOutcomeInvalidNodeRef:
		return GraphSetEdgeInvalidNodeRef, nil
	}
	// An outcome this SDK does not know: fail closed rather than report a write.
	return "", ErrDispatchError
}

// witGraphProperties maps SDK properties to the WIT list. A nil value is a
// malformed request.
func witGraphProperties(properties []GraphProperty) (cm.List[graph.GraphProperty], error) {
	out := make([]graph.GraphProperty, 0, len(properties))
	for _, property := range properties {
		var value graph.GraphPropertyValue
		switch v := property.Value.(type) {
		case GraphString:
			value = graph.GraphPropertyValueStringValue(string(v))
		case GraphLong:
			value = graph.GraphPropertyValueLongValue(int64(v))
		case GraphDouble:
			value = graph.GraphPropertyValueDoubleValue(float64(v))
		case GraphBool:
			value = graph.GraphPropertyValueBoolValue(bool(v))
		case GraphBytes:
			value = graph.GraphPropertyValueBytesValue(cm.ToList([]uint8(v)))
		default:
			return cm.List[graph.GraphProperty]{}, ErrSerializationError
		}
		out = append(out, graph.GraphProperty{Name: property.Name, Value: value})
	}
	return cm.ToList(out), nil
}

func sdkGraphProperties(properties cm.List[graph.GraphProperty]) []GraphProperty {
	out := make([]GraphProperty, 0, properties.Len())
	for _, property := range properties.Slice() {
		value := property.Value
		var sdk GraphPropertyValue
		if v := value.StringValue(); v != nil {
			sdk = GraphString(*v)
		} else if v := value.LongValue(); v != nil {
			sdk = GraphLong(*v)
		} else if v := value.DoubleValue(); v != nil {
			sdk = GraphDouble(*v)
		} else if v := value.BoolValue(); v != nil {
			sdk = GraphBool(*v)
		} else if v := value.BytesValue(); v != nil {
			sdk = GraphBytes(append([]byte(nil), v.Slice()...))
		}
		out = append(out, GraphProperty{Name: property.Name, Value: sdk})
	}
	return out
}
