// Package middleware provides some useful and commonly implemented middleware functions for StormRPC servers.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/actatum/stormrpc"
)

type logOutput struct {
	Time    time.Time  `json:"time"`
	Level   slog.Level `json:"level"`
	Msg     string     `json:"msg"`
	Request struct {
		ID       string `json:"id"`
		TraceID  string `json:"trace_id"`
		Duration string `json:"duration"`
	} `json:"request"`
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func TestLogger(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewJSONHandler(buf, nil))

		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "there"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			resp, err := stormrpc.NewResponse(r.Reply, map[string]string{"hello": "world"})
			if err != nil {
				return stormrpc.NewErrorResponse(r.Reply, err)
			}
			return resp
		})

		h := RequestID(Logger(logger)(handler))
		_ = h(context.Background(), req)

		var out logOutput
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatal(err)
		}

		if out.Level != slog.LevelInfo {
			t.Errorf("got level = %v, want %v", out.Level, slog.LevelInfo)
		} else if out.Msg != "Success" {
			t.Errorf("got msg = %v, want %v", out.Msg, "Success")
		}
	})

	t.Run("error response", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewJSONHandler(buf, nil))

		req, _ := stormrpc.NewRequest("test", map[string]string{"hi": "there"})
		handler := stormrpc.HandlerFunc(func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			return stormrpc.NewErrorResponse(r.Reply, fmt.Errorf("some error"))
		})

		h := RequestID(Logger(logger)(handler))
		_ = h(context.Background(), req)

		var out logOutput
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatal(err)
		}

		if out.Level != slog.LevelError {
			t.Errorf("got level = %v, want %v", out.Level, slog.LevelError)
		} else if out.Msg != "Server Error" {
			t.Errorf("got msg = %v, want %v", out.Msg, "Server Error")
		} else if out.Error.Code != stormrpc.ErrorCodeUnknown.String() {
			t.Errorf("got error code = %v, want %v", out.Error.Code, stormrpc.ErrorCodeUnknown.String())
		}
	})
}
