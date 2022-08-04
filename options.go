package stormrpc

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

func (o *HeaderCallOption) after(c *callOptions) {}

// WithHeaders returns a CallOption that appends the given headers to the request.
func WithHeaders(h map[string]string) CallOption {
	return &HeaderCallOption{Headers: h}
}
