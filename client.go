// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"
)

// Client represents a stormRPC client. It contains all functionality for making RPC requests
// to stormRPC servers.
type Client struct {
	nc *nats.Conn
}

// NewClient returns a new instance of a Client.
func NewClient(natsURL string, opts ...ClientOption) (*Client, error) {
	options := clientOptions{}

	for _, o := range opts {
		o.applyClient(&options)
	}

	if options.nc == nil {
		var err error
		options.nc, err = nats.Connect(natsURL)
		if err != nil {
			return nil, err
		}
	}

	return &Client{
		nc: options.nc,
	}, nil
}

// Close closes the underlying nats connection.
func (c *Client) Close() {
	c.nc.Close()
}

// Do completes a request to a stormRPC Server.
func (c *Client) Do(ctx context.Context, r Request, opts ...CallOption) Response {
	options := callOptions{
		headers: make(map[string]string),
	}
	for _, o := range opts {
		err := o.before(&options)
		if err != nil {
			return NewErrorResponse("", err)
		}
	}

	applyOptions(&r, &options)

	dl, ok := ctx.Deadline()
	if ok {
		setDeadlineHeader(r.Header, dl)
	}

	msg, err := c.nc.RequestMsgWithContext(ctx, r.Msg)
	if errors.Is(err, nats.ErrNoResponders) {
		return Response{
			Msg: msg,
			Err: Errorf(ErrorCodeInternal, "no servers available for subject: %s", r.Subject()),
		}
	}
	if err != nil {
		return Response{
			Msg: msg,
			Err: err, // TODO: probably use errorf and inspect different error types from nats.
		}
	}

	// Inspect headers and set error if appropriate
	rpcErr := parseErrorHeader(msg.Header)
	if rpcErr != nil {
		return Response{
			Msg: msg,
			Err: rpcErr,
		}
	}

	return Response{
		Msg: msg,
		Err: nil,
	}
}

func applyOptions(r *Request, options *callOptions) {
	for k, v := range options.headers {
		r.Header.Set(k, v)
	}
}
