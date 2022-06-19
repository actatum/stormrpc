package middleware

import (
	"stormrpc"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// RequestIDHeader represents the key in the header map that stores the unique request id.
const RequestIDHeader = "X-Request-Id"

// RequestID extracts the value from the request header if its present. If none is present a uuid is generated
// to serve as the request id. This id is passed into the request context to be extracted later. It is also added
// to the response headers.
func RequestID(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
	return func(r stormrpc.Request) stormrpc.Response {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		r.Context = NewContextWithRequestID(r.Context, id)

		resp := next(r)

		if resp.Header == nil {
			resp.Header = nats.Header{}
		}
		resp.Header.Set(RequestIDHeader, id)

		return resp
	}
}
