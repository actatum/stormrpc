package stormrpc

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

// Request is stormRPC's wrapper around a nats.Msg and is used by both clients and servers.
type Request struct {
	*nats.Msg
}

// NewRequest constructs a new request with the given parameters. It also handles encoding the request body.
func NewRequest(subject string, body any, opts ...RequestOption) (Request, error) {
	options := requestOptions{
		encodeProto:   false,
		encodeMsgpack: false,
	}

	for _, o := range opts {
		o.apply(&options)
	}

	var data []byte
	var err error
	var contentType string

	switch {
	case options.encodeProto:
		switch m := body.(type) {
		case proto.Message:
			data, err = proto.Marshal(m)
			contentType = "application/protobuf"
		default:
			return Request{}, fmt.Errorf("failed to encode proto message: invalid type: %T", m)
		}
	case options.encodeMsgpack:
		data, err = msgpack.Marshal(body)
		contentType = "application/msgpack"
	default:
		data, err = json.Marshal(body)
		contentType = "application/json"
	}
	if err != nil {
		return Request{}, err
	}

	headers := nats.Header{}
	headers.Set("Content-Type", contentType)
	msg := &nats.Msg{
		Data:    data,
		Subject: subject,
		Header:  headers,
	}

	return Request{
		Msg: msg,
	}, nil
}

type requestOptions struct {
	encodeProto   bool
	encodeMsgpack bool
}

// RequestOption represents functional options for configuring a request.
type RequestOption interface {
	apply(options *requestOptions)
}

type encodeProtoOption bool

func (p encodeProtoOption) apply(opts *requestOptions) {
	opts.encodeProto = bool(p)
}

// WithEncodeProto is a RequestOption to encode the request body using the proto.Marshal method.
func WithEncodeProto() RequestOption {
	return encodeProtoOption(true)
}

type encodeMsgpackOption bool

func (p encodeMsgpackOption) apply(opts *requestOptions) {
	opts.encodeMsgpack = bool(p)
}

// WithEncodeMsgpack is a RequestOption to encode the request body using the msgpack.Marshal method.
func WithEncodeMsgpack() RequestOption {
	return encodeMsgpackOption(true)
}

// Decode de-serializes the body into the passed in object. The de-serialization method is based on
// the request's Content-Type header.
func (r *Request) Decode(v any) error {
	var err error

	switch r.Header.Get("Content-Type") {
	case "application/msgpack":
		err = msgpack.Unmarshal(r.Data, v)
	case "application/protobuf":
		switch m := v.(type) {
		case proto.Message:
			err = proto.Unmarshal(r.Data, m)
		default:
			return fmt.Errorf("failed to decode proto message: invalid type :%T", v)
		}
	default:
		err = json.Unmarshal(r.Data, v)
	}

	if err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	return nil
}

// Subject returns the underlying nats.Msg subject.
func (r *Request) Subject() string {
	return r.Msg.Subject
}
