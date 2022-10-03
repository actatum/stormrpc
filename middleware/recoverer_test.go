// Package middleware provides some useful and commonly implemented middleware functions for StormRPC servers.
package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/actatum/stormrpc"
)

func TestRecoverer(t *testing.T) {
	t.Run("no panic", func(t *testing.T) {
		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "there"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			return stormrpc.NewErrorResponse("test", fmt.Errorf("new"))
		})
		h := Recoverer(handler)
		resp := h(context.Background(), req)
		if resp.Err.Error() != fmt.Errorf("new").Error() {
			t.Fatalf("got = %v, want %v", resp.Err, fmt.Errorf("new"))
		}
	})

	t.Run("panic", func(t *testing.T) {
		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "there"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			panic(fmt.Errorf("panic"))
		})
		h := Recoverer(handler)
		resp := h(context.Background(), req)

		code := stormrpc.CodeFromErr(resp.Err)
		if code != stormrpc.ErrorCodeInternal {
			t.Fatalf("got = %v, want %v", code, stormrpc.ErrorCodeInternal)
		}

		msg := stormrpc.MessageFromErr(resp.Err)
		if msg != "panic" {
			t.Fatalf("got = %v, want %v", resp.Err.Error(), "panic")
		}
	})
}
