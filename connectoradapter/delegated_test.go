package connectoradapter

import (
	"errors"
	"strings"
	"testing"

	"kyzmet.ai/go/world/local-mcp-service/kyzmet/host/types"
)

func codeOf(t *testing.T, err error) ErrorCode {
	t.Helper()
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("expected a *ToolError, got %v", err)
	}
	return toolErr.Code
}

func TestDelegatedResultsAreBoundedAttributedJSON(t *testing.T) {
	success, err := delegatedSuccess(`{"items":[]}`, "token", 64)
	if err != nil || string(success.ToolResult) != `{"items":[]}` || success.ProvenanceToken != "token" {
		t.Fatalf("unexpected %+v %v", success, err)
	}

	cases := []struct {
		result, token string
		want          ErrorCode
	}{
		{strings.Repeat("1", 65), "token", ErrorResponseTooLarge},
		{`{}`, "", ErrorProviderSchemaChanged},
		{`{`, "token", ErrorProviderSchemaChanged},
	}
	for _, c := range cases {
		_, err := delegatedSuccess(c.result, c.token, 64)
		if got := codeOf(t, err); got != c.want {
			t.Fatalf("%q/%q: expected %s, got %s", c.result, c.token, c.want, got)
		}
	}
}

func TestDelegatedFailuresKeepTheSourceCode(t *testing.T) {
	err := delegatedFailure(types.ToolErrorCredentialRejected)
	if got := codeOf(t, err); got != ErrorCredentialRejected {
		t.Fatalf("expected credential_rejected, got %s", got)
	}
}
