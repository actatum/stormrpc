// Package middleware provides some useful and commonly implemented middleware functions for StormRPC servers.
package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/actatum/stormrpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestLogger(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "there"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			resp, err := stormrpc.NewResponse(r.Reply, map[string]string{"hello": "world"})
			if err != nil {
				return stormrpc.NewErrorResponse(r.Reply, err)
			}
			return resp
		})
		l := zaptest.NewLogger(t, zaptest.WrapOptions(zap.Hooks(func(e zapcore.Entry) error {
			if e.Level != zap.InfoLevel {
				t.Errorf("e.Level got = %v, want %v", e.Level, zap.InfoLevel)
			}
			if e.Message != "Success" {
				t.Errorf("e.Message got = %v, want %v", e.Message, "Success")
			}

			return nil
		})))
		h := Logger(l)(handler)
		resp := h(context.Background(), req)
		fmt.Println(resp)
	})

	t.Run("error response", func(t *testing.T) {
		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "there"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			return stormrpc.NewErrorResponse(r.Reply, fmt.Errorf("some error"))
		})
		l := zaptest.NewLogger(t, zaptest.WrapOptions(zap.Hooks(func(e zapcore.Entry) error {
			if e.Level != zap.ErrorLevel {
				t.Errorf("e.Level got = %v, want %v", e.Level, zap.ErrorLevel)
			}
			if e.Message != "Server Error" {
				t.Errorf("e.Message got = %v, want %v", e.Message, "Server Error")
			}

			return nil
		})))
		h := Logger(l)(handler)
		resp := h(context.Background(), req)
		fmt.Println(resp)
	})
}
