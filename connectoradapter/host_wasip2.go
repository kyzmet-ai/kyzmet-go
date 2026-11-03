// protobuf-policy-file: allowed external (1 site) [df6a2ff13c2a] — delegated tool arguments, defined by the tool contract.

//go:build wasip2

package connectoradapter

import (
	"encoding/json"

	binaryhttp "kyzmet.ai/go/world/local-mcp-service/kyzmet/host/binary-http"
	"kyzmet.ai/go/world/local-mcp-service/kyzmet/host/connector"
	"kyzmet.ai/go/world/local-mcp-service/kyzmet/host/http"
	"kyzmet.ai/go/world/local-mcp-service/kyzmet/host/types"
	"go.bytecodealliance.org/cm"
)

// HTTPRequestDo sends a provider request through the host's http import and
// returns the text response.
func HTTPRequestDo(request HTTPRequest) (HTTPResponse, error) {
	response, failure, failed := http.HTTPRequest(
		request.URL, request.Method, headers(request), body(request), request.TimeoutMS,
	).Result()
	if failed {
		return HTTPResponse{}, hostCallError(failure)
	}
	return HTTPResponse{Status: response.Status, Headers: pairs(response.Headers), Body: response.Body}, nil
}

// HTTPRequestBytes sends a provider request through the host's binary-http
// import and returns the byte response.
func HTTPRequestBytes(request HTTPRequest) (HTTPBinaryResponse, error) {
	response, failure, failed := binaryhttp.HTTPRequest(
		request.URL, request.Method, headers(request), body(request), request.TimeoutMS,
	).Result()
	if failed {
		return HTTPBinaryResponse{}, hostCallError(failure)
	}
	return HTTPBinaryResponse{
		Status:  response.Status,
		Headers: pairs(response.Headers),
		Body:    append([]byte(nil), response.Body.Slice()...),
	}, nil
}

// InvokeTool calls a delegated tool of source with arguments, encoded as JSON.
// A result larger than maxSourceResponseBytes fails as response_too_large.
func InvokeTool(source string, arguments any, maxSourceResponseBytes int) (DelegatedToolSuccess, error) {
	encoded, err := json.Marshal(arguments)
	if err != nil {
		return DelegatedToolSuccess{}, hostCallError(types.HostErrorSerializationError)
	}
	response, failure, failed := connector.InvokeTool(source, string(encoded)).Result()
	if failed {
		return DelegatedToolSuccess{}, hostCallError(failure)
	}
	if success := response.Success(); success != nil {
		return delegatedSuccess(success.ToolResult, success.ProvenanceToken, maxSourceResponseBytes)
	}
	return DelegatedToolSuccess{}, delegatedFailure(response.Failure().Code)
}

func headers(request HTTPRequest) cm.List[http.HTTPHeader] {
	list := make([]http.HTTPHeader, 0, len(request.Headers))
	for _, header := range request.Headers {
		list = append(list, http.HTTPHeader{Name: header[0], Value: header[1]})
	}
	return cm.ToList(list)
}

func body(request HTTPRequest) cm.Option[string] {
	if request.Body == nil {
		return cm.None[string]()
	}
	return cm.Some(*request.Body)
}

func pairs(headers cm.List[http.HTTPHeader]) [][2]string {
	list := make([][2]string, 0, headers.Len())
	for _, header := range headers.Slice() {
		list = append(list, [2]string{header.Name, header.Value})
	}
	return list
}
