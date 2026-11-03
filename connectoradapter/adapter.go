// protobuf-policy-file: allowed external (7 sites) [86075378d983] — tool-call arguments and results, defined by the tool contract.

// Package connectoradapter builds local MCP services on the typed
// local-mcp-service world.
//
// An adapter is a Handler from a Request to a Response. Export registers it as
// the component's local-mcp-service-handler. Provider HTTP goes through
// HTTPRequestDo and HTTPRequestBytes, delegated tools through InvokeTool. Only
// a tool's arguments and results are JSON, as its tool contract defines them;
// everything else crosses the host boundary as WIT types.
//
// Build the component for the local-mcp-service world:
//
//	func init() { connectoradapter.Export(handle) }
//	func main() {}
package connectoradapter

import (
	"encoding/json"
	"errors"
	"fmt"

	handler "kyzmet.ai/go/world/local-mcp-service/kyzmet/host/local-mcp-service-handler"
	"kyzmet.ai/go/world/local-mcp-service/kyzmet/host/types"
	"go.bytecodealliance.org/cm"
)

// MaxInvocationBytes bounds a tool invocation's arguments.
const MaxInvocationBytes = 64 * 1024

// ErrorCode is the stable tool-contract error vocabulary. Only the code
// crosses the host boundary; messages stay in the guest.
type ErrorCode string

const (
	ErrorInvalidArguments      ErrorCode = "invalid_arguments"
	ErrorNotFound              ErrorCode = "not_found"
	ErrorPermissionDenied      ErrorCode = "permission_denied"
	ErrorCredentialRejected    ErrorCode = "credential_rejected"
	ErrorRateLimited           ErrorCode = "rate_limited"
	ErrorQuotaExceeded         ErrorCode = "quota_exceeded"
	ErrorConflict              ErrorCode = "conflict"
	ErrorProviderUnavailable   ErrorCode = "provider_unavailable"
	ErrorResponseTooLarge      ErrorCode = "response_too_large"
	ErrorProviderSchemaChanged ErrorCode = "provider_schema_changed"
	ErrorSyncTokenExpired      ErrorCode = "sync_token_expired"
)

// ErrorCodes lists every ErrorCode, in the order of the WIT tool-error enum.
var ErrorCodes = []ErrorCode{
	ErrorInvalidArguments,
	ErrorNotFound,
	ErrorPermissionDenied,
	ErrorCredentialRejected,
	ErrorRateLimited,
	ErrorQuotaExceeded,
	ErrorConflict,
	ErrorProviderUnavailable,
	ErrorResponseTooLarge,
	ErrorProviderSchemaChanged,
	ErrorSyncTokenExpired,
}

// ErrUnknownTool reports a tool this adapter does not implement. It crosses as
// not_found.
var ErrUnknownTool = errors.New("unknown connector tool")

// ErrHostUnavailable is returned by host calls outside a wasip2 component.
var ErrHostUnavailable = errors.New("connector host imports are only available inside the adapter sandbox")

// ToolError is a tool-contract error.
type ToolError struct {
	Code    ErrorCode
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

// Error returns a tool-contract error.
func Error(code ErrorCode, message string) error {
	return &ToolError{Code: code, Message: message}
}

// HostFailure is the host's reason for refusing a host call: the WIT
// host-error cases.
type HostFailure uint8

const (
	HostNotGranted HostFailure = iota
	HostSensitivityExceeded
	HostApprovalRequired
	HostSerializationError
	HostRateLimited
	HostNotImplemented
	HostDispatchError
)

var hostFailureNames = [...]string{
	"not-granted",
	"sensitivity-exceeded",
	"approval-required",
	"serialization-error",
	"rate-limited",
	"not-implemented",
	"dispatch-error",
}

func (f HostFailure) String() string {
	if int(f) < len(hostFailureNames) {
		return hostFailureNames[f]
	}
	return fmt.Sprintf("host-failure(%d)", uint8(f))
}

// ErrorCode is the tool-contract code a refused host call surfaces as.
func (f HostFailure) ErrorCode() ErrorCode {
	switch f {
	case HostNotGranted, HostApprovalRequired:
		return ErrorPermissionDenied
	case HostRateLimited:
		return ErrorRateLimited
	case HostSerializationError:
		return ErrorProviderSchemaChanged
	default:
		return ErrorProviderUnavailable
	}
}

// HostCallError reports a host call the host refused.
type HostCallError struct {
	Failure HostFailure
}

func (e *HostCallError) Error() string {
	return "connector host call refused: " + e.Failure.String()
}

// Request is one tool invocation: the tool's name and its arguments, which
// Export has checked are JSON within MaxInvocationBytes.
type Request struct {
	Tool string
	Arguments json.RawMessage
}

// AttributedItem is one normalized item and the provenance tokens of the
// delegated results it was derived from.
type AttributedItem struct {
	Content          any
	ProvenanceTokens []string
}

type responseKind uint8

const (
	responseSuccess responseKind = iota
	responseAttributed
	responseError
)

// Response is a handled invocation's outcome. Build it with Success,
// AttributedResponse or ErrorResponse.
type Response struct {
	kind    responseKind
	result  any
	items   []AttributedItem
	summary any
	code    ErrorCode
	message string
}

// Success is the tool's result, as its tool contract defines it.
func Success(result any) Response {
	return Response{kind: responseSuccess, result: result}
}

// AttributedResponse returns items attributed to the delegated calls they
// came from, each carrying the provenance tokens InvokeTool returned, and an
// optional summary of the service's own (nil for none).
func AttributedResponse(items []AttributedItem, summary any) Response {
	return Response{kind: responseAttributed, items: items, summary: summary}
}

// ErrorResponse is a tool-contract error. The message stays in the guest.
func ErrorResponse(code ErrorCode, message string) Response {
	return Response{kind: responseError, code: code, message: message}
}

// Handler handles one invocation. A returned error crosses as its code:
// ErrUnknownTool as not_found, a *ToolError as its Code, a *HostCallError as
// its Failure's code, and any other error as provider_unavailable.
type Handler func(Request) (Response, error)

type invocationResult = cm.Result[handler.InvocationResultShape, handler.InvocationResult, types.HostError]

// Export registers handler as this component's local-mcp-service-handler.
// Call it from an init function.
func Export(h Handler) {
	handler.Exports.Invoke = func(request handler.InvocationRequest) invocationResult {
		return invoke(h, request)
	}
}

func invoke(h Handler, invocation handler.InvocationRequest) invocationResult {
	request, err := decodeInvocation(invocation)
	if err != nil {
		return encodeInvocation(Response{}, err)
	}
	return encodeInvocation(h(request))
}

func decodeInvocation(invocation handler.InvocationRequest) (Request, error) {
	if len(invocation.Arguments) > MaxInvocationBytes {
		return Request{}, Error(ErrorInvalidArguments, "tool arguments exceed the adapter input limit")
	}
	if !json.Valid([]byte(invocation.Arguments)) {
		return Request{}, Error(ErrorInvalidArguments, "tool arguments are not JSON")
	}
	return Request{Tool: invocation.Tool, Arguments: json.RawMessage(invocation.Arguments)}, nil
}

func encodeInvocation(response Response, err error) invocationResult {
	if err != nil {
		return ok(handler.InvocationResultError(toolError(errorCode(err))))
	}
	switch response.kind {
	case responseSuccess:
		result, err := json.Marshal(response.result)
		if err != nil {
			return cm.Err[invocationResult](types.HostErrorSerializationError)
		}
		return ok(handler.InvocationResultResultJSON(string(result)))
	case responseAttributed:
		items := make([]handler.AttributedItem, 0, len(response.items))
		for _, item := range response.items {
			content, err := json.Marshal(item.Content)
			if err != nil {
				return cm.Err[invocationResult](types.HostErrorSerializationError)
			}
			items = append(items, handler.AttributedItem{
				ContentJSON:      string(content),
				ProvenanceTokens: cm.ToList(item.ProvenanceTokens),
			})
		}
		summary := cm.None[string]()
		if response.summary != nil {
			encoded, err := json.Marshal(response.summary)
			if err != nil {
				return cm.Err[invocationResult](types.HostErrorSerializationError)
			}
			summary = cm.Some(string(encoded))
		}
		return ok(handler.InvocationResultAttributed(handler.AttributedResult{
			Items:       cm.ToList(items),
			SummaryJSON: summary,
		}))
	default:
		return ok(handler.InvocationResultError(toolError(response.code)))
	}
}

func ok(result handler.InvocationResult) invocationResult {
	return cm.OK[invocationResult](result)
}

func errorCode(err error) ErrorCode {
	var toolErr *ToolError
	var hostErr *HostCallError
	switch {
	case errors.Is(err, ErrUnknownTool):
		return ErrorNotFound
	case errors.As(err, &toolErr):
		return toolErr.Code
	case errors.As(err, &hostErr):
		return hostErr.Failure.ErrorCode()
	default:
		return ErrorProviderUnavailable
	}
}

func toolError(code ErrorCode) types.ToolError {
	for index, candidate := range ErrorCodes {
		if candidate == code {
			return types.ToolError(index)
		}
	}
	return types.ToolErrorProviderUnavailable
}

func errorCodeOf(code types.ToolError) ErrorCode {
	if int(code) < len(ErrorCodes) {
		return ErrorCodes[code]
	}
	return ErrorProviderUnavailable
}

func hostCallError(failure types.HostError) error {
	return &HostCallError{Failure: HostFailure(failure)}
}

// HTTPRequest is one provider HTTP request. Headers keep their order and
// repeated names.
type HTTPRequest struct {
	Method    string
	URL       string
	Headers   [][2]string
	Body      *string
	TimeoutMS uint32
}

func NewHTTPRequest(method, url string) HTTPRequest {
	return HTTPRequest{
		Method:    method,
		URL:       url,
		Headers:   [][2]string{},
		TimeoutMS: 5000,
	}
}

func (r HTTPRequest) Header(name, value string) HTTPRequest {
	r.Headers = append(r.Headers, [2]string{name, value})
	return r
}

func (r HTTPRequest) WithBody(body string) HTTPRequest {
	r.Body = &body
	return r
}

func (r HTTPRequest) WithTimeoutMS(timeoutMS uint32) HTTPRequest {
	r.TimeoutMS = timeoutMS
	return r
}

// HTTPResponse is a provider response with a text body.
type HTTPResponse struct {
	Status  uint16
	Headers [][2]string
	Body    string
}

// HTTPBinaryResponse is a provider response with a byte body.
type HTTPBinaryResponse struct {
	Status  uint16
	Headers [][2]string
	Body    []byte
}

// DelegatedToolSuccess is a delegated tool's result and the provenance token
// that attributes items derived from it.
type DelegatedToolSuccess struct {
	ToolResult      json.RawMessage
	ProvenanceToken string
}

// delegatedSuccess checks a delegated tool's typed response against the
// source-response budget.
func delegatedSuccess(toolResult, provenanceToken string, maxSourceResponseBytes int) (DelegatedToolSuccess, error) {
	if len(toolResult) > maxSourceResponseBytes {
		return DelegatedToolSuccess{}, Error(ErrorResponseTooLarge, "delegated tool result exceeds the source-response budget")
	}
	if provenanceToken == "" {
		return DelegatedToolSuccess{}, Error(ErrorProviderSchemaChanged, "delegated tool result has no provenance token")
	}
	if !json.Valid([]byte(toolResult)) {
		return DelegatedToolSuccess{}, Error(ErrorProviderSchemaChanged, "delegated tool result is not JSON")
	}
	return DelegatedToolSuccess{ToolResult: json.RawMessage(toolResult), ProvenanceToken: provenanceToken}, nil
}

func delegatedFailure(code types.ToolError) error {
	errorCode := errorCodeOf(code)
	return Error(errorCode, string(errorCode))
}
