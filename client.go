package stormrpc

import (
	"errors"

	"github.com/nats-io/nats.go"
)

type Client struct {
	nc *nats.Conn
}

func NewClient(natsURL string, opts ...ClientOption) (*Client, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		nc: nc,
	}, nil
}

type clientOptions struct{}

type ClientOption interface {
	apply(*clientOptions)
}

func (c *Client) Do(r *Request) Response {
	msg, err := c.nc.RequestMsgWithContext(r.Context, r.Msg)
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
