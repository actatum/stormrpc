// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

type testErrorHandler struct {
	cnt int
}

func (t *testErrorHandler) handle(_ context.Context, _ error) {
	t.cnt++
}

func (t *testErrorHandler) clear() {
	t.cnt = 0
}

func TestNewServer(t *testing.T) {
	teh := &testErrorHandler{}
	type args struct {
		cfg  *ServerConfig
		opts []ServerOption
	}
	tests := []struct {
		name    string
		args    args
		want    *Server
		runNats bool
		wantErr bool
	}{
		{
			name: "defaults",
			args: args{
				cfg: &ServerConfig{
					Name:    "name",
					NatsURL: "nats://localhost:40897",
				},
				opts: nil,
			},
			want: &Server{
				timeout:      defaultServerTimeout,
				mw:           nil,
				errorHandler: func(ctx context.Context, err error) {},
			},
			runNats: true,
			wantErr: false,
		},
		{
			name: "with error handler opt",
			args: args{
				cfg: &ServerConfig{
					Name:    "name",
					NatsURL: "nats://localhost:40897",
				},
				opts: []ServerOption{
					WithErrorHandler(teh.handle),
				},
			},
			want: &Server{
				timeout:      defaultServerTimeout,
				mw:           nil,
				errorHandler: teh.handle,
			},
			runNats: true,
			wantErr: false,
		},
		{
			name: "no nats running",
			args: args{
				cfg: &ServerConfig{
					Name:    "name",
					NatsURL: "nats://localhost:40897",
				},
				opts: nil,
			},
			want:    nil,
			runNats: false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(teh.clear)
			if tt.runNats {
				startNatsServer(t)
			}

			got, err := NewServer(tt.args.cfg, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.timeout != tt.want.timeout {
				t.Errorf("NewServer() timeout = %v, want %v", got.timeout, tt.want.timeout)
			} else if (got.errorHandler == nil) != (tt.want.errorHandler == nil) {
				t.Errorf("NewServer() errorHandler = %v, want %v", got.errorHandler == nil, tt.want.errorHandler == nil)
			} else if !reflect.DeepEqual(got.mw, tt.want.mw) {
				t.Errorf("NewServer() mw = %v, want %v", got.mw, tt.want.mw)
			}

			for _, opt := range tt.args.opts {
				_, ok := opt.(errorHandlerOption)
				if ok {
					got.errorHandler(context.Background(), fmt.Errorf("hi"))
					tt.want.errorHandler(context.Background(), fmt.Errorf("hi"))
					if teh.cnt != 2 {
						t.Errorf("NewServer() errorHandler expected 2 calls got %v", teh.cnt)
					}
				}
			}
		})
	}
}

func TestServer_RunAndShutdown(t *testing.T) {
	clientURL := startNatsServer(t)

	srv, err := NewServer(&ServerConfig{
		NatsURL: clientURL,
		Name:    "test",
	})
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

func TestServer_Run(t *testing.T) {
	type args struct {
		ctx context.Context
		req Request
	}
	type endpoint struct {
		name    string
		handler HandlerFunc
	}
	tests := []struct {
		name      string
		endpoints []endpoint
		args      args
		wantErr   bool
	}{
		{
			name: "ok",
			endpoints: []endpoint{
				{
					name: "test",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response {
						_, ok := ctx.Deadline()
						if !ok {
							t.Error("context should have deadline")
						}
						return Response{
							Msg: &nats.Msg{
								Subject: r.Reply,
								Data:    []byte(`{"response":"1"}`),
							},
							Err: nil,
						}
					}),
				},
			},
			args: args{
				ctx: ctxWithTimeout(t, 5*time.Second),
				req: mustNewRequest(t, "test", map[string]string{"hello": "world"}),
			},
			wantErr: false,
		},
		{
			name: "context deadline exceeded",
			endpoints: []endpoint{
				{
					name: "test",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response {
						_, ok := ctx.Deadline()
						if !ok {
							t.Error("context should have deadline")
						}

						ticker := time.NewTicker(2 * time.Second)
						defer ticker.Stop()
						for {
							select {
							case <-ctx.Done():
								return NewErrorResponse(r.Reply, Error{
									Code:    ErrorCodeDeadlineExceeded,
									Message: ctx.Err().Error(),
								})
							case <-ticker.C:
								return NewErrorResponse(r.Reply, fmt.Errorf("somethings wrong"))
							}
						}
					}),
				},
			},
			args: args{
				ctx: ctxWithTimeout(t, 2*time.Second),
				req: mustNewRequest(t, "test", map[string]string{"hello": "world"}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientURL := startNatsServer(t)

			srv, err := NewServer(&ServerConfig{
				NatsURL: clientURL,
				Name:    "test",
			})
			if err != nil {
				t.Fatal(err)
			}

			for _, ep := range tt.endpoints {
				srv.Handle(ep.name, ep.handler)
			}

			errs := make(chan error)
			go func(srv *Server, errs chan error) {
				errs <- srv.Run()
			}(srv, errs)

			client, err := NewClient(clientURL)
			if err != nil {
				t.Fatal(err)
			}

			resp := client.Do(tt.args.ctx, tt.args.req)
			if (resp.Err != nil) != tt.wantErr {
				t.Errorf("Client.Do() error = %v, wantErr %v", resp.Err, tt.wantErr)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if err = srv.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}

			err = <-errs
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestServer_Handle(t *testing.T) {
	clientURL := startNatsServer(t)

	s, err := NewServer(&ServerConfig{
		Name:    "test",
		NatsURL: clientURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Handle("testing", func(ctx context.Context, r Request) Response { return Response{} })
	if err != nil {
		t.Fatal(err)
	}

	_, ok := s.handlerFuncs["testing"]
	if !ok {
		t.Errorf("expected handler to exist for subject %s", "testing")
	}
}

func TestServer_Subjects(t *testing.T) {
	type endpoint struct {
		name    string
		handler HandlerFunc
	}
	tests := []struct {
		name      string
		endpoints []endpoint
		want      []string
	}{
		{
			name: "ok",
			endpoints: []endpoint{
				{
					name:    "test",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response { return Response{} }),
				},
			},
			want: []string{"test"},
		},
		{
			name: "multiple endpoints",
			endpoints: []endpoint{
				{
					name:    "1",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response { return Response{} }),
				},
				{
					name:    "2",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response { return Response{} }),
				},
				{
					name:    "3",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response { return Response{} }),
				},
			},
			want: []string{"1", "2", "3"},
		},
		{
			name: "duplicate endpoints",
			endpoints: []endpoint{
				{
					name:    "1",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response { return Response{} }),
				},
				{
					name:    "1",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response { return Response{} }),
				},
				{
					name:    "2",
					handler: HandlerFunc(func(ctx context.Context, r Request) Response { return Response{} }),
				},
			},
			want: []string{"1", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientURL := startNatsServer(t)

			srv, err := NewServer(&ServerConfig{
				Name:    "test",
				NatsURL: clientURL,
			})
			if err != nil {
				t.Fatal(err)
			}

			for _, ep := range tt.endpoints {
				srv.Handle(ep.name, ep.handler)
			}

			got := srv.Subjects()

			if !sameStringSlice(got, tt.want) {
				t.Fatalf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServer_Use(t *testing.T) {
	type fields struct {
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
				shutdownSignal: tt.fields.shutdownSignal,
				handlerFuncs:   tt.fields.handlerFuncs,
				errorHandler:   tt.fields.errorHandler,
				timeout:        tt.fields.timeout,
				mw:             tt.fields.mw,
			}
			s.Use(tt.args.mw...)

			if len(tt.args.mw) != len(s.mw) {
				t.Fatalf("expected slices to be the same length got = %v, want %v", s.mw, tt.args.mw)
			}
		})
	}
}

func TestServer_applyMiddlewares(t *testing.T) {
	type fields struct {
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

func startNatsServer(tb testing.TB) string {
	tb.Helper()

	ns, err := server.NewServer(&server.Options{
		Port: 40897,
	})
	if err != nil {
		tb.Fatal(err)
	}

	ns.Start()

	tb.Cleanup(func() {
		ns.Shutdown()
		ns.WaitForShutdown()
	})

	if !ns.ReadyForConnections(1 * time.Second) {
		tb.Fatal("timeout waiting for nats server")
	}

	return ns.ClientURL()
}

func mustNewRequest(tb testing.TB, subject string, body any, opts ...RequestOption) Request {
	req, err := NewRequest(subject, body, opts...)
	if err != nil {
		tb.Fatal(err)
	}

	return req
}

func ctxWithTimeout(tb testing.TB, timeout time.Duration) context.Context {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	tb.Cleanup(cancel)

	return ctx
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
