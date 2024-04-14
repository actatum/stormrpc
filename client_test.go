// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestNewClient(t *testing.T) {
	clientURL := startNatsServer(t)
	type args struct {
		natsURL string
		opts    func() []ClientOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "defaults",
			args: args{
				natsURL: clientURL,
				opts: func() []ClientOption {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "with nats conn opt",
			args: args{
				natsURL: "",
				opts: func() []ClientOption {
					nc, err := nats.Connect(clientURL)
					if err != nil {
						t.Fatal(err)
					}
					return []ClientOption{WithNatsConn(nc)}
				},
			},
			wantErr: false,
		},
		{
			name: "no nats running",
			args: args{
				natsURL: nats.DefaultURL,
				opts: func() []ClientOption {
					return nil
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.args.natsURL, tt.args.opts()...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.nc == nil {
				t.Errorf("NewClient() nats conn is not expected to be nil")
			}
		})
	}
}

func TestClient_Do(t *testing.T) {
	// t.Parallel()

	ns, err := server.NewServer(&server.Options{
		Port: 41397,
	})
	if err != nil {
		t.Fatal(err)
	}
	go ns.Start()
	t.Cleanup(func() {
		ns.Shutdown()
		ns.WaitForShutdown()
	})

	if !ns.ReadyForConnections(1 * time.Second) {
		t.Error("timeout waiting for nats server")
		return
	}

	clientURL := ns.ClientURL()

	t.Run("deadline exceeded", func(t *testing.T) {
		timeout := 50 * time.Millisecond
		subject := strconv.Itoa(rand.Int())
		srv, err := NewServer(&ServerConfig{
			NatsURL: clientURL,
			Name:    "test",
		})
		if err != nil {
			t.Fatal(err)
		}
		srv.Handle(subject, func(ctx context.Context, r Request) Response {
			time.Sleep(timeout + 10*time.Millisecond)
			return Response{Msg: &nats.Msg{Subject: r.Reply}}
		})
		go func() {
			_ = srv.Run()
		}()
		t.Cleanup(func() {
			_ = srv.Shutdown(context.Background())
		})

		client, err := NewClient(clientURL)
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		r, err := NewRequest(subject, map[string]string{"howdy": "partner"})
		if err != nil {
			t.Fatal(err)
		}

		resp := client.Do(ctx, r)
		if resp.Err == nil {
			t.Fatal("expected error got nil")
		}

		if !errors.Is(resp.Err, context.DeadlineExceeded) {
			t.Fatalf("got = %v, want %v", resp.Err, context.DeadlineExceeded)
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		timeout := 50 * time.Millisecond
		subject := strconv.Itoa(rand.Int())
		srv, err := NewServer(&ServerConfig{
			NatsURL: clientURL,
			Name:    "test",
		})
		if err != nil {
			t.Fatal(err)
		}
		srv.Handle(subject, func(ctx context.Context, r Request) Response {
			return NewErrorResponse(r.Reply, Errorf(ErrorCodeNotFound, "thingy not found"))
		})
		go func() {
			_ = srv.Run()
		}()
		t.Cleanup(func() {
			_ = srv.Shutdown(context.Background())
		})

		client, err := NewClient(clientURL)
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		r, err := NewRequest(subject, map[string]string{"howdy": "partner"})
		if err != nil {
			t.Fatal(err)
		}

		resp := client.Do(ctx, r)
		if resp.Err == nil {
			t.Fatal("expected error got nil")
		}

		code := CodeFromErr(resp.Err)
		if code != ErrorCodeNotFound {
			t.Fatalf("got = %v, want %v", code, ErrorCodeNotFound)
		}
		msg := MessageFromErr(resp.Err)
		if msg != "thingy not found" {
			t.Fatalf("got = %v, want %v", msg, "thingy not found")
		}
	})

	t.Run("no servers", func(t *testing.T) {
		subject := strconv.Itoa(rand.Int())

		client, err := NewClient(clientURL)
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		req, err := NewRequest(subject, map[string]string{"x": "D"})
		if err != nil {
			t.Fatal(err)
		}
		resp := client.Do(ctx, req)
		if resp.Err == nil {
			t.Fatal("expected error got nil")
		}

		code := CodeFromErr(resp.Err)
		if code != ErrorCodeInternal {
			t.Fatalf("got = %v, want %v", code, ErrorCodeInternal)
		}
		msg := MessageFromErr(resp.Err)
		if msg != fmt.Sprintf("no servers available for subject: %s", subject) {
			t.Fatalf(
				"got = %v, want %v",
				msg,
				fmt.Sprintf("no servers available for subject: %s", subject),
			)
		}
	})

	t.Run("request option errors", func(t *testing.T) {
		client, err := NewClient(clientURL)
		if err != nil {
			t.Fatal(err)
		}

		subject := strconv.Itoa(rand.Int())
		r, err := NewRequest(subject, map[string]string{"howdy": "partner"})
		if err != nil {
			t.Fatal(err)
		}

		opts := []CallOption{
			&optWithError{},
		}

		resp := client.Do(context.Background(), r, opts...)

		if resp.Err == nil {
			t.Fatal("expected error got nil")
		}

		if CodeFromErr(resp.Err) != ErrorCodeUnknown {
			t.Errorf("CodeFromErr() got = %v, want %v", CodeFromErr(err), ErrorCodeUnknown)
		} else if MessageFromErr(resp.Err) != "unknown error" {
			t.Errorf("MessageFromErr() got = %v, want %v", MessageFromErr(resp.Err), "unknown error")
		}
	})

	t.Run("successful request", func(t *testing.T) {
		timeout := 50 * time.Millisecond
		subject := strconv.Itoa(rand.Int())
		srv, err := NewServer(&ServerConfig{
			NatsURL: clientURL,
			Name:    "test",
		})
		if err != nil {
			t.Fatal(err)
		}
		srv.Handle(subject, func(ctx context.Context, r Request) Response {
			var resp Response
			resp, err = NewResponse(r.Reply, map[string]string{"hello": "world"})
			if err != nil {
				return NewErrorResponse(r.Reply, err)
			}
			return resp
		})
		go func() {
			_ = srv.Run()
		}()
		t.Cleanup(func() {
			_ = srv.Shutdown(context.Background())
		})

		client, err := NewClient(clientURL)
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		r, err := NewRequest(subject, map[string]string{"howdy": "partner"})
		if err != nil {
			t.Fatal(err)
		}

		resp := client.Do(ctx, r)
		if resp.Err != nil {
			t.Fatal(resp.Err)
		}

		var result map[string]string
		if err = resp.Decode(&result); err != nil {
			t.Fatal(err)
		}

		if result["hello"] != "world" {
			t.Fatalf("got = %v, want %v", result["hello"], "world")
		}
	})

	t.Run("successful request w/headers option", func(t *testing.T) {
		apiKey := uuid.NewString()
		timeout := 50 * time.Millisecond
		subject := strconv.Itoa(rand.Int())
		srv, err := NewServer(&ServerConfig{
			NatsURL: clientURL,
			Name:    "test",
		})
		if err != nil {
			t.Fatal(err)
		}
		srv.Handle(subject, func(ctx context.Context, r Request) Response {
			if r.Header.Get("X-API-Key") != apiKey {
				t.Errorf("X-API-Key got = %v, want %v", r.Header.Get("X-API-Key"), apiKey)
			}
			var resp Response
			resp, err = NewResponse(r.Reply, map[string]string{"hello": "world"})
			if err != nil {
				return NewErrorResponse(r.Reply, err)
			}
			return resp
		})
		go func() {
			_ = srv.Run()
		}()
		t.Cleanup(func() {
			_ = srv.Shutdown(context.Background())
		})

		client, err := NewClient(clientURL)
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		r, err := NewRequest(subject, map[string]string{"howdy": "partner"})
		if err != nil {
			t.Fatal(err)
		}

		resp := client.Do(ctx, r, WithHeaders(map[string]string{
			"X-API-Key": apiKey,
		}))
		if resp.Err != nil {
			t.Fatal(resp.Err)
		}

		var result map[string]string
		if err = resp.Decode(&result); err != nil {
			t.Fatal(err)
		}

		if result["hello"] != "world" {
			t.Fatalf("got = %v, want %v", result["hello"], "world")
		}
	})
}

type optWithError struct{}

func (o *optWithError) before(*callOptions) error {
	return fmt.Errorf("before call errored")
}

func (o *optWithError) after(*callOptions) {}
