// Package middleware provides some useful and commonly implemented middleware functions for StormRPC servers.
package middleware

import (
	"context"

	"github.com/actatum/stormrpc"
)

// Recoverer handles recovering from a panic in other HandlerFunc's.
func Recoverer(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
	return func(ctx context.Context, r stormrpc.Request) (resp stormrpc.Response) {
		defer func() {
			err := recover()
			if err != nil {
				resp = stormrpc.NewErrorResponse(
					r.Reply,
					stormrpc.Errorf(stormrpc.ErrorCodeInternal, "%v", err),
				)
			}
		}()

		resp = next(ctx, r)

		return resp
	}
}
