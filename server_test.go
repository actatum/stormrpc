package stormrpc

import (
	"context"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestServer_RunAndShutdown(t *testing.T) {
	ns, err := server.NewServer(&server.Options{
		Port: 40897,
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

	srv, err := NewServer("test", ns.ClientURL())
	if err != nil {
		t.Fatal(err)
	}

	runCh := make(chan error)
	go func(ch chan error) {
		runErr := srv.Run()
		runCh <- runErr
	}(runCh)
	time.Sleep(250 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}

	err = <-runCh
	if err != nil {
		t.Fatal(err)
	}
}

func TestServer_handler(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	ns, err := server.NewServer(&server.Options{
		Port: 40897,
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

	t.Run("successful handle", func(t *testing.T) {
		t.Parallel()

		srv, err := NewServer("test", ns.ClientURL())
		if err != nil {
			t.Fatal(err)
		}

		subject := strconv.Itoa(rand.Int())
		srv.Handle(subject, func(r Request) Response {
			return Response{
				Msg: &nats.Msg{
					Subject: r.Reply,
					Data:    []byte(`{"response":"1"}`),
				},
				Err: nil,
			}
		})

		runCh := make(chan error)
		go func(ch chan error) {
			runErr := srv.Run()
			runCh <- runErr
		}(runCh)
		time.Sleep(250 * time.Millisecond)

		client, err := NewClient(ns.ClientURL())
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		req, err := NewRequest(ctx, subject, map[string]string{"x": "D"})
		if err != nil {
			t.Fatal(err)
		}
		resp := client.Do(req)
		if resp.Err != nil {
			t.Fatal(resp.Err)
		}

		var result map[string]string
		if err = resp.Decode(&result); err != nil {
			t.Fatal(err)
		}

		if result["response"] != "1" {
			t.Fatalf("got = %v, want %v", result["response"], "1")
		}

		if err = srv.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}

		err = <-runCh
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestServer_Handle(t *testing.T) {
	s := Server{
		handlerFuncs: make(map[string]HandlerFunc),
	}

	t.Run("OK", func(t *testing.T) {
		s.Handle("testing", func(r Request) Response { return Response{} })

		if _, ok := s.handlerFuncs["testing"]; !ok {
			t.Fatal("expected key testing to contain a handler func")
		}
	})
}

func TestServer_Subjects(t *testing.T) {
	s := Server{
		handlerFuncs: make(map[string]HandlerFunc),
	}

	s.Handle("testing", func(r Request) Response { return Response{} })
	s.Handle("testing", func(r Request) Response { return Response{} })
	s.Handle("1, 2, 3", func(r Request) Response { return Response{} })

	expected := []string{"testing", "1, 2, 3"}

	got := s.Subjects()

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("got = %v, want %v", got, expected)
	}
}
