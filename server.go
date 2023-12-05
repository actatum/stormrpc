// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

var defaultServerTimeout = 5 * time.Second

type ServerConfig struct {
	NatsURL string
	Name    string
	Version string

	errorHandler ErrorHandler
}

func (s *ServerConfig) setDefaults() {
	if s.NatsURL == "" {
		s.NatsURL = nats.DefaultURL
	}
	if s.Name == "" {
		s.Name = "service"
	}
	if s.Version == "" {
		s.Version = "0.1.0"
	}
	if s.errorHandler == nil {
		s.errorHandler = func(ctx context.Context, err error) {}
	}
}

// Server represents a stormRPC server. It contains all functionality for handling RPC requests.
type Server struct {
	nc             *nats.Conn
	shutdownSignal chan struct{}
	handlerFuncs   map[string]HandlerFunc
	errorHandler   ErrorHandler
	timeout        time.Duration
	mw             []Middleware

	svc micro.Service
}

// NewServer returns a new instance of a Server.
func NewServer(cfg *ServerConfig, opts ...ServerOption) (*Server, error) {
	cfg.setDefaults()

	for _, o := range opts {
		o.apply(cfg)
	}

	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		return nil, err
	}

	mc := micro.Config{
		Name:    cfg.Name,
		Version: cfg.Version,
	}
	if cfg.errorHandler != nil {
		mc.ErrorHandler = func(s micro.Service, n *micro.NATSError) {
			ctx, cancel := context.WithTimeout(context.Background(), defaultServerTimeout)
			defer cancel()
			cfg.errorHandler(ctx, n)
		}
	}

	svc, err := micro.AddService(nc, mc)
	if err != nil {
		return nil, err
	}

	return &Server{
		nc:             nc,
		shutdownSignal: make(chan struct{}),
		handlerFuncs:   make(map[string]HandlerFunc),
		timeout:        defaultServerTimeout,
		errorHandler:   cfg.errorHandler,
		svc:            svc,
	}, nil
}

// ServerOption represents functional options for configuring a stormRPC Server.
type ServerOption interface {
	apply(*ServerConfig)
}

type errorHandlerOption ErrorHandler

func (h errorHandlerOption) apply(opts *ServerConfig) {
	opts.errorHandler = ErrorHandler(h)
}

// WithErrorHandler is a ServerOption that allows for registering a function for handling server errors.
func WithErrorHandler(fn ErrorHandler) ServerOption {
	return errorHandlerOption(fn)
}

// HandlerFunc is the function signature for handling of a single request to a stormRPC server.
type HandlerFunc func(ctx context.Context, r Request) Response

// Middleware is the function signature for wrapping HandlerFunc's to extend their functionality.
type Middleware func(next HandlerFunc) HandlerFunc

// ErrorHandler is the function signature for handling server errors.
type ErrorHandler func(context.Context, error)

// Handle registers a new HandlerFunc on the server.
func (s *Server) Handle(subject string, fn HandlerFunc) {
	s.handlerFuncs[subject] = fn
}

// Run listens on the configured subjects.
func (s *Server) Run() error {
	s.applyMiddlewares()
	for sub, fn := range s.handlerFuncs {
		if err := s.createMicroEndpoint(sub, fn); err != nil {
			return err
		}
	}

	<-s.shutdownSignal
	return nil
}

// Shutdown stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.nc.FlushWithContext(ctx); err != nil {
		return err
	}

	s.nc.Close()
	s.shutdownSignal <- struct{}{}
	return nil
}

// Subjects returns a list of all subjects with registered handler funcs.
func (s *Server) Subjects() []string {
	subs := make([]string, 0, len(s.handlerFuncs))
	for k := range s.handlerFuncs {
		subs = append(subs, k)
	}

	return subs
}

// Use applies all given middleware globally across all handlers.
func (s *Server) Use(mw ...Middleware) {
	s.mw = mw
}

func (s *Server) applyMiddlewares() {
	for k, hf := range s.handlerFuncs {
		for i := len(s.mw) - 1; i >= 0; i-- {
			hf = s.mw[i](hf)
		}

		s.handlerFuncs[k] = hf
	}
}

// handler serves the request to the specific request handler based on subject.
// wildcard subjects are not supported as you'll need to register a handler func for each
// rpc the server supports.
func (s *Server) handler(msg *nats.Msg) {
	fn := s.handlerFuncs[msg.Subject]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dl := parseDeadlineHeader(msg.Header)
	if !dl.IsZero() { // if deadline is present use it
		ctx, cancel = context.WithDeadline(context.Background(), dl)
		defer cancel()
	} else {
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	req := Request{
		Msg: msg,
	}
	// pass headers into context for use in protobuf generated servers.
	ctx = newContextWithHeaders(ctx, req.Header)

	resp := fn(ctx, req)

	if resp.Err != nil {
		if resp.Header == nil {
			resp.Header = nats.Header{}
		}
		setErrorHeader(resp.Header, resp.Err)
		err := msg.RespondMsg(resp.Msg)
		if err != nil {
			s.errorHandler(ctx, err)
		}
	}

	err := msg.RespondMsg(resp.Msg)
	if err != nil {
		s.errorHandler(ctx, err)
	}
}

func (s *Server) createMicroEndpoint(subject string, handlerFunc HandlerFunc) error {
	err := s.svc.AddEndpoint(subject, micro.ContextHandler(context.Background(), func(ctx context.Context, r micro.Request) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		dl := parseDeadlineHeader(nats.Header(r.Headers()))
		if !dl.IsZero() { // if deadline is present use it
			ctx, cancel = context.WithDeadline(context.Background(), dl)
			defer cancel()
		} else {
			ctx, cancel = context.WithTimeout(ctx, s.timeout)
			defer cancel()
		}

		resp := handlerFunc(ctx, Request{
			Msg: &nats.Msg{
				Subject: r.Subject(),
				Reply:   "",
				Header:  nats.Header(r.Headers()),
				Data:    r.Data(),
				Sub:     &nats.Subscription{},
			},
		})

		if resp.Err != nil {
			if resp.Header == nil {
				resp.Header = nats.Header{}
			}
			setErrorHeader(resp.Header, resp.Err)
		}

		err := r.Respond(resp.Data, micro.WithHeaders(micro.Headers(resp.Header)))
		if err != nil {
			s.errorHandler(ctx, err)
		}
	}))
	if err != nil {
		return err
	}

	return nil
}
