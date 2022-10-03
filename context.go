// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"context"

	"github.com/nats-io/nats.go"
)

type ctxKey int

const (
	headerContextKey ctxKey = iota
)

// HeadersFromContext retrieves RPC headers from the given context.
func HeadersFromContext(ctx context.Context) nats.Header {
	h, ok := ctx.Value(headerContextKey).(nats.Header)
	if !ok {
		return make(nats.Header)
	}

	return h
}

// newContextWithHeaders creates a new context with Header information stored in it.
func newContextWithHeaders(ctx context.Context, headers nats.Header) context.Context {
	return context.WithValue(ctx, headerContextKey, headers)
}
