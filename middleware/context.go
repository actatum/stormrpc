package middleware

import "context"

type contextKey int

const (
	requestIDContextKey contextKey = iota
)

// NewContextWithRequestID will take the original context and use it as the parent context for the returned context.
// The passed in id will be added to this new context.
func NewContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, id)
}

// RequestIDFromContext extracts the request id from the context if one is present. If no request id is present
// an empty string will be returned.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey).(string)
	return id
}
