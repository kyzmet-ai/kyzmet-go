package raw

import "testing"

// A graph property with a nil value is rejected before any host call.
func TestGraphWritesRejectANilPropertyValue(t *testing.T) {
	nilValue := []GraphProperty{{Name: "title", Value: GraphString("Paint")}, {Name: "broken"}}

	if _, err := CreateNode(CreateNodeRequest{NodeType: "task", NodeId: "t1", Properties: nilValue}); err != ErrSerializationError {
		t.Fatalf("CreateNode: got %v, want ErrSerializationError", err)
	}
	if _, err := CreateEdge(CreateEdgeRequest{FromId: "t1", ToId: "p1", EdgeType: "e", Properties: nilValue}); err != ErrSerializationError {
		t.Fatalf("CreateEdge: got %v, want ErrSerializationError", err)
	}
}

// Valid properties pass validation and reach the host call, which host
// builds stub as not implemented.
func TestGraphWritesWithValidPropertiesReachTheHost(t *testing.T) {
	valid := []GraphProperty{
		{Name: "title", Value: GraphString("Paint")},
		{Name: "done", Value: GraphBool(false)},
		{Name: "estimate", Value: GraphDouble(1.5)},
		{Name: "priority", Value: GraphLong(-3)},
		{Name: "raw", Value: GraphBytes{0, 255}},
	}
	if _, err := CreateNode(CreateNodeRequest{NodeType: "task", NodeId: "t1", Properties: valid}); err != ErrNotImplemented {
		t.Fatalf("CreateNode: got %v, want ErrNotImplemented from the host stub", err)
	}
	if _, err := CreateEdge(CreateEdgeRequest{FromId: "t1", ToId: "p1", EdgeType: "e"}); err != ErrNotImplemented {
		t.Fatalf("CreateEdge: got %v, want ErrNotImplemented from the host stub", err)
	}
}

func TestDeleteNodesReachesTheHost(t *testing.T) {
	if err := DeleteNodes(DeleteNodesRequest{NodeIds: []string{"t1"}}); err != ErrNotImplemented {
		t.Fatalf("DeleteNodes: got %v, want ErrNotImplemented from the host stub", err)
	}
}

func TestGetNodeReachesTheHost(t *testing.T) {
	if node, err := GetNode("t1"); node != nil || err != ErrNotImplemented {
		t.Fatalf("GetNode: got (%v, %v), want (nil, ErrNotImplemented) from the host stub", node, err)
	}
}
