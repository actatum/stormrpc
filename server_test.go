package stormrpc

import (
	"context"
	"errors"
	"fmt"
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
		srv.Handle(subject, func(ctx context.Context, r Request) Response {
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

		req, err := NewRequest(subject, map[string]string{"x": "D"})
		if err != nil {
			t.Fatal(err)
		}
		resp := client.Do(ctx, req)
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

	t.Run("context deadline exceeded", func(t *testing.T) {
		t.Parallel()

		srv, err := NewServer("test", ns.ClientURL())
		if err != nil {
			t.Fatal(err)
		}

		subject := strconv.Itoa(rand.Int())
		srv.Handle(subject, func(ctx context.Context, r Request) Response {
			for {
				select {
				case <-ctx.Done():
					return NewErrorResponse(r.Reply, Error{
						Code:    ErrorCodeDeadlineExceeded,
						Message: ctx.Err().Error(),
					})
				}
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

		req, err := NewRequest(subject, map[string]string{"x": "D"})
		if err != nil {
			t.Fatal(err)
		}
		resp := client.Do(ctx, req)
		var e *Error
		ok := errors.As(resp.Err, &e)
		if !ok {
			t.Fatalf("expected error to be of type Error, got %T", resp.Err)
		}
		if e.Code != ErrorCodeDeadlineExceeded {
			t.Fatalf("e.Code got = %v, want %v", e.Code, ErrorCodeDeadlineExceeded)
		} else if e.Message != context.DeadlineExceeded.Error() {
			t.Fatalf("e.Message got = %v, want %v", e.Message, context.DeadlineExceeded.Error())
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
		s.Handle("testing", func(ctx context.Context, r Request) Response { return Response{} })

		if _, ok := s.handlerFuncs["testing"]; !ok {
			t.Fatal("expected key testing to contain a handler func")
		}
	})
}

func TestServer_Subjects(t *testing.T) {
	s := Server{
		handlerFuncs: make(map[string]HandlerFunc),
	}

	s.Handle("testing", func(ctx context.Context, r Request) Response { return Response{} })
	s.Handle("testing", func(ctx context.Context, r Request) Response { return Response{} })
	s.Handle("1, 2, 3", func(ctx context.Context, r Request) Response { return Response{} })

	want := []string{"testing", "1, 2, 3"}

	got := s.Subjects()

	if !sameStringSlice(got, want) {
		t.Fatalf("got = %v, want %v", got, want)
	}
}

func TestServer_Use(t *testing.T) {
	type fields struct {
		nc             *nats.Conn
		name           string
		shutdownSignal chan struct{}
		handlerFuncs   map[string]HandlerFunc
		errorHandler   ErrorHandler
		timeout        time.Duration
		mw             []Middleware
	}
	type args struct {
		mw []Middleware
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "add middlewares",
			fields: fields{
				mw: make([]Middleware, 0),
			},
			args: args{
				mw: []Middleware{
					func(next HandlerFunc) HandlerFunc {
						return func(ctx context.Context, request Request) Response {
							return NewErrorResponse("test", fmt.Errorf("hi"))
						}
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				nc:             tt.fields.nc,
				name:           tt.fields.name,
				shutdownSignal: tt.fields.shutdownSignal,
				handlerFuncs:   tt.fields.handlerFuncs,
				errorHandler:   tt.fields.errorHandler,
				timeout:        tt.fields.timeout,
				mw:             tt.fields.mw,
			}
			s.Use(tt.args.mw...)

			if !reflect.DeepEqual(tt.args.mw, s.mw) {
				t.Fatalf("got = %v, want %v", s.mw, tt.args.mw)
			}
		})
	}
}

func TestServer_applyMiddlewares(t *testing.T) {
	type fields struct {
		nc             *nats.Conn
		name           string
		shutdownSignal chan struct{}
		handlerFuncs   map[string]HandlerFunc
		errorHandler   ErrorHandler
		timeout        time.Duration
		mw             []Middleware
	}
	tests := []struct {
		name   string
		fields fields
		base   HandlerFunc
		want   HandlerFunc
	}{
		{
			name: "single middleware, single handler",
			fields: fields{
				handlerFuncs: make(map[string]HandlerFunc),
				mw: []Middleware{
					func(next HandlerFunc) HandlerFunc {
						return func(ctx context.Context, r Request) Response {
							return NewErrorResponse("test", fmt.Errorf("hi"))
						}
					},
				},
			},
			base: func(ctx context.Context, r Request) Response {
				return NewErrorResponse("bob", fmt.Errorf("now"))
			},
			want: func(next HandlerFunc) HandlerFunc {
				return func(ctx context.Context, r Request) Response {
					return NewErrorResponse("test", fmt.Errorf("hi"))
				}
			}(func(ctx context.Context, r Request) Response {
				return NewErrorResponse("bob", fmt.Errorf("now"))
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				nc:             tt.fields.nc,
				name:           tt.fields.name,
				shutdownSignal: tt.fields.shutdownSignal,
				handlerFuncs:   tt.fields.handlerFuncs,
				errorHandler:   tt.fields.errorHandler,
				timeout:        tt.fields.timeout,
				mw:             tt.fields.mw,
			}
			s.Handle("base", tt.base)
			s.applyMiddlewares()

			resp := s.handlerFuncs["base"](context.Background(), Request{})
			if resp.Err == nil {
				t.Fatalf("expected error got nil")
			}

			if resp.Err.Error() != "hi" {
				t.Fatalf("got = %v, want %v", resp.Err.Error(), "hi")
			}
		})
	}
}

func sameStringSlice(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(x))
	for _, _x := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	return len(diff) == 0
}
