package connectoradapter_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"kyzmet.ai/go/connectoradapter"
	handler "kyzmet.ai/go/world/local-mcp-service/kyzmet/host/local-mcp-service-handler"
	"kyzmet.ai/go/world/local-mcp-service/kyzmet/host/types"
)

// invoke registers h with Export and calls the export the host calls.
func invoke(t *testing.T, h connectoradapter.Handler, tool, arguments string) handler.InvocationResult {
	t.Helper()
	handler.Exports.Invoke = nil
	connectoradapter.Export(h)
	if handler.Exports.Invoke == nil {
		t.Fatal("Export did not register local-mcp-service-handler.invoke")
	}
	result, hostErr, failed := handler.Exports.Invoke(handler.InvocationRequest{Tool: tool, Arguments: arguments}).Result()
	if failed {
		t.Fatalf("invoke failed with host error %v", hostErr)
	}
	return result
}

func toolError(t *testing.T, result handler.InvocationResult) types.ToolError {
	t.Helper()
	code := result.Error()
	if code == nil {
		t.Fatalf("expected a tool error, got %v", result)
	}
	return *code
}

func TestSuccessCrossesAsResultJSON(t *testing.T) {
	result := invoke(t, func(request connectoradapter.Request) (connectoradapter.Response, error) {
		if request.Tool != "list_tasks" {
			t.Fatalf("expected list_tasks, got %q", request.Tool)
		}
		var args map[string]int
		if err := json.Unmarshal(request.Arguments, &args); err != nil || args["limit"] != 2 {
			t.Fatalf("expected limit=2, got %s (%v)", request.Arguments, err)
		}
		return connectoradapter.Success(map[string][]string{"tasks": {}}), nil
	}, "list_tasks", `{"limit":2}`)

	body := result.ResultJSON()
	if body == nil || *body != `{"tasks":[]}` {
		t.Fatalf("expected result-json {\"tasks\":[]}, got %v", result)
	}
}

func TestAttributedItemsKeepTheirProvenance(t *testing.T) {
	result := invoke(t, func(connectoradapter.Request) (connectoradapter.Response, error) {
		return connectoradapter.AttributedResponse(
			[]connectoradapter.AttributedItem{{Content: map[string]int{"id": 1}, ProvenanceTokens: []string{"p1", "p2"}}},
			map[string]int{"count": 1},
		), nil
	}, "search", `{}`)

	attributed := result.Attributed()
	if attributed == nil {
		t.Fatalf("expected attributed, got %v", result)
	}
	items := attributed.Items.Slice()
	if len(items) != 1 || items[0].ContentJSON != `{"id":1}` || strings.Join(items[0].ProvenanceTokens.Slice(), ",") != "p1,p2" {
		t.Fatalf("unexpected items %+v", items)
	}
	if summary := attributed.SummaryJSON.Some(); summary == nil || *summary != `{"count":1}` {
		t.Fatalf("unexpected summary %v", attributed.SummaryJSON)
	}

	none := invoke(t, func(connectoradapter.Request) (connectoradapter.Response, error) {
		return connectoradapter.AttributedResponse(nil, nil), nil
	}, "search", `{}`)
	if !none.Attributed().SummaryJSON.None() {
		t.Fatal("a nil summary must cross as none")
	}
}

func TestErrorsCrossAsTheirToolErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want types.ToolError
	}{
		{"unknown tool", connectoradapter.ErrUnknownTool, types.ToolErrorNotFound},
		{"wrapped unknown tool", fmt.Errorf("routing: %w", connectoradapter.ErrUnknownTool), types.ToolErrorNotFound},
		{"tool error", connectoradapter.Error(connectoradapter.ErrorConflict, "etag mismatch"), types.ToolErrorConflict},
		{"refused host call", &connectoradapter.HostCallError{Failure: connectoradapter.HostRateLimited}, types.ToolErrorRateLimited},
		{"any other error", errors.New("boom"), types.ToolErrorProviderUnavailable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := invoke(t, func(connectoradapter.Request) (connectoradapter.Response, error) {
				return connectoradapter.Response{}, c.err
			}, "tool", `{}`)
			if got := toolError(t, result); got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}

	result := invoke(t, func(connectoradapter.Request) (connectoradapter.Response, error) {
		return connectoradapter.ErrorResponse(connectoradapter.ErrorSyncTokenExpired, "resync"), nil
	}, "tool", `{}`)
	if got := toolError(t, result); got != types.ToolErrorSyncTokenExpired {
		t.Fatalf("expected sync-token-expired, got %v", got)
	}
}

func TestEveryErrorCodeMatchesTheWITEnum(t *testing.T) {
	want := map[connectoradapter.ErrorCode]types.ToolError{
		connectoradapter.ErrorInvalidArguments:      types.ToolErrorInvalidArguments,
		connectoradapter.ErrorNotFound:              types.ToolErrorNotFound,
		connectoradapter.ErrorPermissionDenied:      types.ToolErrorPermissionDenied,
		connectoradapter.ErrorCredentialRejected:    types.ToolErrorCredentialRejected,
		connectoradapter.ErrorRateLimited:           types.ToolErrorRateLimited,
		connectoradapter.ErrorQuotaExceeded:         types.ToolErrorQuotaExceeded,
		connectoradapter.ErrorConflict:              types.ToolErrorConflict,
		connectoradapter.ErrorProviderUnavailable:   types.ToolErrorProviderUnavailable,
		connectoradapter.ErrorResponseTooLarge:      types.ToolErrorResponseTooLarge,
		connectoradapter.ErrorProviderSchemaChanged: types.ToolErrorProviderSchemaChanged,
		connectoradapter.ErrorSyncTokenExpired:      types.ToolErrorSyncTokenExpired,
	}
	if len(connectoradapter.ErrorCodes) != len(want) {
		t.Fatalf("ErrorCodes has %d codes, the WIT enum %d", len(connectoradapter.ErrorCodes), len(want))
	}
	for _, code := range connectoradapter.ErrorCodes {
		result := invoke(t, func(connectoradapter.Request) (connectoradapter.Response, error) {
			return connectoradapter.ErrorResponse(code, ""), nil
		}, "tool", `{}`)
		if got := toolError(t, result); got != want[code] {
			t.Fatalf("%s crosses as %v, want %v", code, got, want[code])
		}
	}
}

func TestInvalidArgumentsFailBeforeTheHandlerRuns(t *testing.T) {
	for _, arguments := range []string{"{", `"` + strings.Repeat("x", connectoradapter.MaxInvocationBytes) + `"`} {
		result := invoke(t, func(connectoradapter.Request) (connectoradapter.Response, error) {
			t.Fatal("the handler must not run")
			return connectoradapter.Response{}, nil
		}, "tool", arguments)
		if got := toolError(t, result); got != types.ToolErrorInvalidArguments {
			t.Fatalf("expected invalid-arguments, got %v", got)
		}
	}
}

func TestAnUnencodableResultIsASerializationError(t *testing.T) {
	connectoradapter.Export(func(connectoradapter.Request) (connectoradapter.Response, error) {
		return connectoradapter.Success(make(chan int)), nil
	})
	_, hostErr, failed := handler.Exports.Invoke(handler.InvocationRequest{Tool: "tool", Arguments: `{}`}).Result()
	if !failed || hostErr != types.HostErrorSerializationError {
		t.Fatalf("expected serialization-error, got failed=%v %v", failed, hostErr)
	}
}

func TestRefusedHostCallsSurfaceAsToolContractCodes(t *testing.T) {
	want := map[connectoradapter.HostFailure]connectoradapter.ErrorCode{
		connectoradapter.HostNotGranted:          connectoradapter.ErrorPermissionDenied,
		connectoradapter.HostApprovalRequired:    connectoradapter.ErrorPermissionDenied,
		connectoradapter.HostRateLimited:         connectoradapter.ErrorRateLimited,
		connectoradapter.HostSerializationError:  connectoradapter.ErrorProviderSchemaChanged,
		connectoradapter.HostSensitivityExceeded: connectoradapter.ErrorProviderUnavailable,
		connectoradapter.HostNotImplemented:      connectoradapter.ErrorProviderUnavailable,
		connectoradapter.HostDispatchError:       connectoradapter.ErrorProviderUnavailable,
	}
	for failure, code := range want {
		if got := failure.ErrorCode(); got != code {
			t.Fatalf("%v surfaces as %s, want %s", failure, got, code)
		}
	}
	if connectoradapter.HostFailure(types.HostErrorDispatchError) != connectoradapter.HostDispatchError {
		t.Fatal("HostFailure must follow the WIT host-error order")
	}
}

func TestHostCallsOutsideAComponentReportNoHost(t *testing.T) {
	if _, err := connectoradapter.HTTPRequestDo(connectoradapter.NewHTTPRequest("GET", "https://example.com")); !errors.Is(err, connectoradapter.ErrHostUnavailable) {
		t.Fatalf("expected ErrHostUnavailable, got %v", err)
	}
	if _, err := connectoradapter.HTTPRequestBytes(connectoradapter.NewHTTPRequest("GET", "https://example.com")); !errors.Is(err, connectoradapter.ErrHostUnavailable) {
		t.Fatalf("expected ErrHostUnavailable, got %v", err)
	}
	if _, err := connectoradapter.InvokeTool("source", map[string]int{}, 1024); !errors.Is(err, connectoradapter.ErrHostUnavailable) {
		t.Fatalf("expected ErrHostUnavailable, got %v", err)
	}
}

func TestHTTPRequestBuilderKeepsHeaderOrder(t *testing.T) {
	request := connectoradapter.NewHTTPRequest("POST", "https://example.com").
		Header("Accept", "application/json").
		Header("Accept", "text/plain").
		WithBody(`{"a":1}`).
		WithTimeoutMS(250)
	if len(request.Headers) != 2 || request.Headers[1][1] != "text/plain" || *request.Body != `{"a":1}` || request.TimeoutMS != 250 {
		t.Fatalf("unexpected request %+v", request)
	}
}
