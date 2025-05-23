// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

var defaultServerTimeout = 5 * time.Second

// ServerConfig is used to configure required fields for a StormRPC server.
// If any fields aren't present a default value will be used.
type ServerConfig struct {
	NatsURL string
	Name    string
	Version string

	nc           *nats.Conn
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
	mu             sync.RWMutex
	nc             *nats.Conn
	shutdownSignal chan struct{}
	handlerFuncs   map[string]HandlerFunc
	errorHandler   ErrorHandler
	timeout        time.Duration
	mw             []Middleware

	running bool

	svc micro.Service
}

// NewServer returns a new instance of a Server.
func NewServer(cfg *ServerConfig, opts ...ServerOption) (*Server, error) {
	cfg.setDefaults()

	for _, o := range opts {
		o.applyServer(cfg)
	}

	if cfg.nc == nil {
		var err error
		cfg.nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			return nil, err
		}
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

	svc, err := micro.AddService(cfg.nc, mc)
	if err != nil {
		return nil, err
	}

	return &Server{
		nc:             cfg.nc,
		shutdownSignal: make(chan struct{}),
		handlerFuncs:   make(map[string]HandlerFunc),
		timeout:        defaultServerTimeout,
		errorHandler:   cfg.errorHandler,
		running:        false,
		svc:            svc,
	}, nil
}

// HandlerFunc is the function signature for handling of a single request to a stormRPC server.
type HandlerFunc func(ctx context.Context, r Request) Response

// Middleware is the function signature for wrapping HandlerFunc's to extend their functionality.
type Middleware func(next HandlerFunc) HandlerFunc

// ErrorHandler is the function signature for handling server errors.
type ErrorHandler func(context.Context, error)

// Handle registers a new HandlerFunc on the server.
func (s *Server) Handle(subject string, fn HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlerFuncs[subject] = fn
}

// Run listens on the configured subjects.
func (s *Server) Run() error {
	s.mu.Lock()
	s.applyMiddlewares()

	for sub, fn := range s.handlerFuncs {
		if err := s.createMicroEndpoint(sub, fn); err != nil {
			return err
		}
	}

	s.running = true
	s.mu.Unlock()

	<-s.shutdownSignal
	return nil
}

// Shutdown stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.svc.Stop(); err != nil {
		return err
	}

	if err := s.nc.FlushWithContext(ctx); err != nil {
		return err
	}

	s.nc.Close()
	s.running = false
	s.shutdownSignal <- struct{}{}
	return nil
}

// Subjects returns a list of all subjects with registered handler funcs.
func (s *Server) Subjects() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := make([]string, 0, len(s.handlerFuncs))
	for k := range s.handlerFuncs {
		subs = append(subs, k)
	}

	return subs
}

// Use applies all given middleware globally across all handlers.
func (s *Server) Use(mw ...Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		s.mw = append(s.mw, mw...)
	}
}

func (s *Server) applyMiddlewares() {
	for k, hf := range s.handlerFuncs {
		for i := len(s.mw) - 1; i >= 0; i-- {
			hf = s.mw[i](hf)
		}

		s.handlerFuncs[k] = hf
	}
}

// createMicroEndpoint registers a HandlerFunc as a micro Endpoint
// allowing for automatic service discovery and observability.
func (s *Server) createMicroEndpoint(subject string, handlerFunc HandlerFunc) error {
	return s.svc.AddEndpoint(
		nameFromSubject(subject),
		micro.ContextHandler(context.Background(), func(ctx context.Context, r micro.Request) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			ctx = newContextWithHeaders(ctx, nats.Header(r.Headers()))

			dl := parseDeadlineHeader(nats.Header(r.Headers()))
			if !dl.IsZero() { // if deadline is present use it
				ctx, cancel = context.WithDeadline(ctx, dl)
				defer cancel()
			} else {
				ctx, cancel = context.WithTimeout(ctx, s.timeout)
				defer cancel()
			}

			resp := handlerFunc(ctx, Request{
				Msg: &nats.Msg{
					Subject: r.Subject(),
					Header:  nats.Header(r.Headers()),
					Data:    r.Data(),
				},
			})

			if resp.Err != nil {
				if resp.Header == nil {
					resp.Header = nats.Header{}
				}
				setErrorHeader(resp.Header, resp.Err)

				err := r.Error(
					CodeFromErr(resp.Err).String(),
					MessageFromErr(resp.Err),
					nil,
					micro.WithHeaders(micro.Headers(resp.Header)),
				)
				if err != nil {
					s.errorHandler(ctx, err)
				}
			}

			err := r.Respond(resp.Data, micro.WithHeaders(micro.Headers(resp.Header)))
			if err != nil {
				s.errorHandler(ctx, err)
			}
		}), micro.WithEndpointSubject(subject))
}

func (s *Server) ready(dur time.Duration) bool {
	deadline := time.Now().Add(dur)
	tick := 25 * time.Millisecond

	for time.Now().Before(deadline) {
		s.mu.RLock()
		running := s.running
		s.mu.RUnlock()

		if running {
			return true
		}
		time.Sleep(tick)
	}

	return false
}

// If a subject contains '.' delimiters replace them with '_' for the endpoint name.
func nameFromSubject(subj string) string {
	return strings.ReplaceAll(subj, ".", "_")
}
