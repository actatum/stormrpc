// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import "github.com/nats-io/nats.go"

// Option represents functional options used to configure stormRPC clients and servers.
type Option interface {
	ClientOption
	ServerOption
}

// ClientOption represents functional options for configuring a stormRPC Client.
type ClientOption interface {
	applyClient(*clientOptions)
}

type clientOptions struct {
	nc *nats.Conn
}

type natsConnOption struct {
	nc *nats.Conn
}

func (n *natsConnOption) applyClient(c *clientOptions) {
	c.nc = n.nc
}

func (n *natsConnOption) applyServer(c *ServerConfig) {
	c.nc = n.nc
}

// WithNatsConn is an Option that allows for using an existing nats client connection.
func WithNatsConn(nc *nats.Conn) Option {
	return &natsConnOption{nc: nc}
}

// ServerOption represents functional options for configuring a stormRPC Server.
type ServerOption interface {
	applyServer(*ServerConfig)
}

type errorHandlerOption ErrorHandler

func (h errorHandlerOption) applyServer(opts *ServerConfig) {
	opts.errorHandler = ErrorHandler(h)
}

// WithErrorHandler is a ServerOption that allows for registering a function for handling server errors.
func WithErrorHandler(fn ErrorHandler) ServerOption {
	return errorHandlerOption(fn)
}

// CallOption configures an RPC to perform actions before it starts or after
// the RPC has completed.
type CallOption interface {
	// before is called before the RPC is sent to any server.
	// If before returns a non-nil error, the RPC fails with that error.
	before(*callOptions) error

	// after is called after the RPC has completed after cannot return an error.
	after(*callOptions)
}

// callOptions contains all configuration for an RPC.
type callOptions struct {
	headers map[string]string
}

// HeaderCallOption is used to configure which headers to append to the outgoing RPC.
type HeaderCallOption struct {
	Headers map[string]string
}

func (o *HeaderCallOption) before(c *callOptions) error {
	c.headers = o.Headers
	return nil
}

func (o *HeaderCallOption) after(_ *callOptions) {}

// WithHeaders returns a CallOption that appends the given headers to the request.
func WithHeaders(h map[string]string) CallOption {
	return &HeaderCallOption{Headers: h}
}
