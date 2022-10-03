// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/actatum/stormrpc/prototest"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

func TestResponse_Decode(t *testing.T) {
	t.Run("decode json", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		data, _ := json.Marshal(body)
		resp := &Response{
			Msg: &nats.Msg{
				Header: nats.Header{
					"Content-Type": []string{"application/json"},
				},
				Data: data,
			},
			Err: nil,
		}

		var got map[string]string
		if err := resp.Decode(&got); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, body) {
			t.Fatalf("NewRequest() got = %v, want %v", got, body)
		}
	})

	t.Run("decode msgpack", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		data, _ := msgpack.Marshal(body)
		resp := &Response{
			Msg: &nats.Msg{
				Header: nats.Header{
					"Content-Type": []string{"application/msgpack"},
				},
				Data: data,
			},
			Err: nil,
		}

		var got map[string]string
		if err := resp.Decode(&got); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, body) {
			t.Fatalf("NewRequest() got = %v, want %v", got, body)
		}
	})

	t.Run("decode proto", func(t *testing.T) {
		body := &prototest.HelloReply{Message: "hi"}
		data, _ := proto.Marshal(body)
		resp := &Response{
			Msg: &nats.Msg{
				Header: nats.Header{
					"Content-Type": []string{"application/protobuf"},
				},
				Data: data,
			},
			Err: nil,
		}

		var got prototest.HelloReply
		if err := resp.Decode(&got); err != nil {
			t.Fatal(err)
		}

		if got.GetMessage() != body.GetMessage() {
			t.Fatalf("got = %v, want %v", got.GetMessage(), body.GetMessage())
		}
	})

	t.Run("decode proto w/non proto message", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		data, _ := json.Marshal(body)
		resp := &Response{
			Msg: &nats.Msg{
				Header: nats.Header{
					"Content-Type": []string{"application/protobuf"},
				},
				Data: data,
			},
			Err: nil,
		}

		var got prototest.HelloReply
		err := resp.Decode(&got)
		if err == nil {
			t.Fatal("expected error got nil")
		}
	})
}

func TestNewErrorResponse(t *testing.T) {
	type args struct {
		reply string
		err   error
	}
	tests := []struct {
		name string
		args args
		want Response
	}{
		{
			name: "non stormrpc error",
			args: args{
				reply: "test",
				err:   fmt.Errorf("10"),
			},
			want: Response{
				Msg: &nats.Msg{
					Subject: "test",
				},
				Err: fmt.Errorf("10"),
			},
		},
		{
			name: "stormrpc error",
			args: args{
				reply: "test",
				err:   Errorf(ErrorCodeNotFound, "hi"),
			},
			want: Response{
				Msg: &nats.Msg{
					Subject: "test",
				},
				Err: Errorf(ErrorCodeNotFound, "hi"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewErrorResponse(tt.args.reply, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewErrorResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		data, _ := json.Marshal(body)

		want := Response{
			Msg: &nats.Msg{
				Subject: "test",
				Header: nats.Header{
					"Content-Type": []string{"application/json"},
				},
				Data: data,
			},
			Err: nil,
		}

		got, err := NewResponse("test", body)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("NewResponse() got = %v, want %v", got, want)
		}
	})

	t.Run("with msgpack", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		data, _ := msgpack.Marshal(body)

		want := Response{
			Msg: &nats.Msg{
				Subject: "test",
				Header: nats.Header{
					"Content-Type": []string{"application/msgpack"},
				},
				Data: data,
			},
			Err: nil,
		}

		got, err := NewResponse("test", body, WithEncodeMsgpack())
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("NewResponse() got = %v, want %v", got, want)
		}
	})

	t.Run("proto option", func(t *testing.T) {
		body := &prototest.HelloReply{Message: "hello"}
		data, _ := proto.Marshal(body)

		want := Response{
			Msg: &nats.Msg{
				Subject: "test",
				Header: nats.Header{
					"Content-Type": []string{"application/protobuf"},
				},
				Data: data,
			},
			Err: nil,
		}

		got, err := NewResponse("test", body, WithEncodeProto())
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("NewResponse() got = %v, want %v", got, want)
		}
	})

	t.Run("proto option w/non proto message", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		_, err := NewResponse("test", body, WithEncodeProto())
		if err == nil {
			t.Fatal("expected error got nil")
		}
	})
}
