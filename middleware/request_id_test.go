// Package middleware provides some useful and commonly implemented middleware functions for StormRPC servers.
package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/actatum/stormrpc"
	"github.com/google/uuid"
)

func TestRequestID(t *testing.T) {
	t.Run("header present", func(t *testing.T) {
		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "bye"})
		req.Header.Set(RequestIDHeader, "testing")
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			if RequestIDFromContext(ctx) != "testing" {
				t.Fatalf("got = %v, want %v", RequestIDFromContext(ctx), "testing")
			}

			return stormrpc.NewErrorResponse("test", fmt.Errorf("x"))
		})
		h := RequestID(handler)
		h(context.Background(), req)
	})

	t.Run("header missing", func(t *testing.T) {
		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "bye"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			_, err := uuid.Parse(RequestIDFromContext(ctx))
			if err != nil {
				t.Fatal("expected request id to be a uuid")
			}

			return stormrpc.NewErrorResponse("test", fmt.Errorf("x"))
		})
		h := RequestID(handler)
		h(context.Background(), req)
	})

	t.Run("header is on response", func(t *testing.T) {
		var want string
		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "bye"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			want = RequestIDFromContext(ctx)
			return stormrpc.NewErrorResponse("test", fmt.Errorf("x"))
		})
		h := RequestID(handler)
		resp := h(context.Background(), req)

		got := resp.Header.Get(RequestIDHeader)
		if got != want {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})
}
