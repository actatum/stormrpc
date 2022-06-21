package stormrpc

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/actatum/stormrpc/prototest"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

func TestNewRequest(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		data, _ := json.Marshal(body)

		want := Request{
			Msg: &nats.Msg{
				Subject: "test",
				Header: nats.Header{
					"Content-Type": []string{"application/json"},
				},
				Data: data,
			},
		}

		got, err := NewRequest("test", body)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("NewRequest() got = %v, want %v", got, want)
		}
	})

	t.Run("msgpack option", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		data, _ := msgpack.Marshal(body)

		want := Request{
			Msg: &nats.Msg{
				Subject: "test",
				Header: nats.Header{
					"Content-Type": []string{"application/msgpack"},
				},
				Data: data,
			},
		}

		got, err := NewRequest("test", body, WithEncodeMsgpack())
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("NewRequest() got = %v, want %v", got, want)
		}
	})

	t.Run("proto option", func(t *testing.T) {
		body := &prototest.Greeting{Message: "hello"}
		data, _ := proto.Marshal(body)

		want := Request{
			Msg: &nats.Msg{
				Subject: "test",
				Header: nats.Header{
					"Content-Type": []string{"application/protobuf"},
				},
				Data: data,
			},
		}

		got, err := NewRequest("test", body, WithEncodeProto())
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("NewRequest() got = %v, want %v", got, want)
		}
	})

	t.Run("proto option w/non proto message", func(t *testing.T) {
		body := map[string]string{"hello": "world"}

		_, err := NewRequest("test", body, WithEncodeProto())
		if err == nil {
			t.Fatal("expected error got nil")
		}
	})
}

func TestRequest_Decode(t *testing.T) {
	t.Run("decode json", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		r, err := NewRequest("test", body)
		if err != nil {
			t.Fatal(err)
		}

		var got map[string]string
		if err = r.Decode(&got); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, body) {
			t.Fatalf("NewRequest() got = %v, want %v", got, body)
		}
	})

	t.Run("decode msgpack", func(t *testing.T) {
		body := map[string]string{"hello": "world"}
		r, err := NewRequest("test", body, WithEncodeMsgpack())
		if err != nil {
			t.Fatal(err)
		}

		var got map[string]string
		if err = r.Decode(&got); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, body) {
			t.Fatalf("NewRequest() got = %v, want %v", got, body)
		}
	})

	t.Run("decode proto", func(t *testing.T) {
		body := &prototest.Greeting{Message: "hi"}
		r, err := NewRequest("test", body, WithEncodeProto())
		if err != nil {
			t.Fatal(err)
		}

		var got prototest.Greeting
		if err = r.Decode(&got); err != nil {
			t.Fatal(err)
		}

		if got.GetMessage() != body.GetMessage() {
			t.Fatalf("got = %v, want %v", got.GetMessage(), body.GetMessage())
		}
	})

	t.Run("decode proto w/non proto message", func(t *testing.T) {
		body := &prototest.Greeting{Message: "hello"}
		r, err := NewRequest("test", body, WithEncodeProto())
		if err != nil {
			t.Fatal(err)
		}

		var got map[string]string
		err = r.Decode(&got)
		if err == nil {
			t.Fatal(err)
		}
	})
}

func TestRequest_Subject(t *testing.T) {
	type fields struct {
		Msg *nats.Msg
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "return msg subject",
			fields: fields{
				Msg: &nats.Msg{
					Subject: "me",
				},
			},
			want: "me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				Msg: tt.fields.Msg,
			}
			if got := r.Subject(); got != tt.want {
				t.Errorf("Subject() = %v, want %v", got, tt.want)
			}
		})
	}
}
