package stormrpc

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestNewClient(t *testing.T) {
	t.Run("no nats server running", func(t *testing.T) {
		_, err := NewClient(nats.DefaultURL)
		if err == nil {
			t.Fatal("expected error got nil")
		}
	})

	t.Run("nats server running", func(t *testing.T) {
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

		c, err := NewClient(ns.ClientURL())
		if err != nil {
			t.Fatal(err)
		}
		c.Close()
	})
}

func TestClient_Do(t *testing.T) {
	t.Parallel()

	rand.Seed(time.Now().UnixNano())
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

	t.Run("deadline exceeded", func(t *testing.T) {
		t.Parallel()

		timeout := 50 * time.Millisecond
		subject := strconv.Itoa(rand.Int())
		srv, err := NewServer("test", ns.ClientURL())
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

		client, err := NewClient(ns.ClientURL())
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
		t.Parallel()

		timeout := 50 * time.Millisecond
		subject := strconv.Itoa(rand.Int())
		srv, err := NewServer("test", ns.ClientURL())
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

		client, err := NewClient(ns.ClientURL())
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
		t.Parallel()

		subject := strconv.Itoa(rand.Int())

		client, err := NewClient(ns.ClientURL())
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

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		timeout := 50 * time.Millisecond
		subject := strconv.Itoa(rand.Int())
		srv, err := NewServer("test", ns.ClientURL())
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

		client, err := NewClient(ns.ClientURL())
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
}
