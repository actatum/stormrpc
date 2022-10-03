// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

// Response is stormRPC's wrapper around a nats.Msg and is used by both clients and servers.
type Response struct {
	*nats.Msg
	Err error
}

// NewResponse constructs a new response with the given parameters. It also handles encoding the response body.
func NewResponse(reply string, body any, opts ...ResponseOption) (Response, error) {
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
			return Response{}, fmt.Errorf("failed to encode proto message: invalid type: %T", m)
		}
	case options.encodeMsgpack:
		data, err = msgpack.Marshal(body)
		contentType = "application/msgpack"
	default:
		data, err = json.Marshal(body)
		contentType = "application/json"
	}
	if err != nil {
		return Response{}, err
	}

	headers := nats.Header{}
	headers.Set("Content-Type", contentType)

	msg := &nats.Msg{
		Subject: reply,
		Header:  headers,
		Data:    data,
	}

	return Response{
		Msg: msg,
		Err: nil,
	}, nil
}

// NewErrorResponse constructs a new error response with the given parameters.
func NewErrorResponse(reply string, err error) Response {
	return Response{
		Msg: &nats.Msg{
			Subject: reply,
		},
		Err: err,
	}
}

// ResponseOption represents functional options for configuring a response.
type ResponseOption RequestOption

// Decode de-serializes the body into the passed in object. The de-serialization method is based on
// the response's Content-Type header.
func (r *Response) Decode(v any) error {
	var err error

	switch r.Header.Get("Content-Type") {
	case "application/msgpack":
		err = msgpack.Unmarshal(r.Data, v)
	case "application/protobuf":
		switch m := v.(type) {
		case proto.Message:
			err = proto.Unmarshal(r.Data, m)
		default:
			return fmt.Errorf("failed to decode proto message: invalid type: %T", m)
		}
	default:
		err = json.Unmarshal(r.Data, v)
	}

	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
