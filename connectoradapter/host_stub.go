//go:build !wasip2

package connectoradapter

// Outside a wasip2 component there is no host: host calls fail with
// ErrHostUnavailable, so adapters and their tests still build.

func HTTPRequestDo(HTTPRequest) (HTTPResponse, error) {
	return HTTPResponse{}, ErrHostUnavailable
}

func HTTPRequestBytes(HTTPRequest) (HTTPBinaryResponse, error) {
	return HTTPBinaryResponse{}, ErrHostUnavailable
}

func InvokeTool(string, any, int) (DelegatedToolSuccess, error) {
	return DelegatedToolSuccess{}, ErrHostUnavailable
}
