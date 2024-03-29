// Package middleware provides some useful and commonly implemented middleware functions for StormRPC servers.
package middleware

import (
	"context"

	"github.com/actatum/stormrpc"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// RequestIDHeader represents the key in the header map that stores the unique request id.
const RequestIDHeader = "X-Request-Id"

// RequestID extracts the value from the request header if its present. If none is present a uuid is generated
// to serve as the request id. This id is passed into the request context to be extracted later. It is also added
// to the response headers.
func RequestID(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
	return func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		ctx = NewContextWithRequestID(ctx, id)

		resp := next(ctx, r)

		if resp.Header == nil {
			resp.Header = nats.Header{}
		}
		resp.Header.Set(RequestIDHeader, id)

		return resp
	}
}
