package stormrpc

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

type Request struct {
	*nats.Msg
}

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

type RequestOption interface {
	apply(options *requestOptions)
}

type encodeProtoOption bool

func (p encodeProtoOption) apply(opts *requestOptions) {
	opts.encodeProto = bool(p)
}

func WithEncodeProto() RequestOption {
	return encodeProtoOption(true)
}

type encodeMsgpackOption bool

func (p encodeMsgpackOption) apply(opts *requestOptions) {
	opts.encodeMsgpack = bool(p)
}

func WithEncodeMsgpack() RequestOption {
	return encodeMsgpackOption(true)
}

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

func (r *Request) Subject() string {
	return r.Msg.Subject
}
