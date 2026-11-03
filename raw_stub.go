//go:build !wasip2 && !wasip1

package raw

// Host builds (go test, go vet) have no Kyzmet host: every call reports ErrNotImplemented.

func createDoc(CreateDocRequest) (EntryId, error)               { return "", ErrNotImplemented }
func getDoc(GetDocRequest) (Document, error)                    { return Document{}, ErrNotImplemented }
func listDocs(ListDocsRequest) ([]string, error)                { return nil, ErrNotImplemented }
func createNode(CreateNodeRequest) (GraphSetNodeOutcome, error) { return "", ErrNotImplemented }
func deleteNodes(DeleteNodesRequest) error                      { return ErrNotImplemented }
func listNodesOfType(ListNodesOfTypeRequest) ([]GraphNode, error) {
	return nil, ErrNotImplemented
}
func getNode(string) (*GraphNode, error)                        { return nil, ErrNotImplemented }
func createEdge(CreateEdgeRequest) (GraphSetEdgeOutcome, error) { return "", ErrNotImplemented }
