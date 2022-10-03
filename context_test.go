// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"context"
	"reflect"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestHeadersFromContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want nats.Header
	}{
		{
			name: "ok",
			args: args{
				ctx: newContextWithHeaders(context.Background(), nats.Header{
					"name": []string{"Gon Freecs"},
				}),
			},
			want: nats.Header{
				"name": []string{"Gon Freecs"},
			},
		},
		{
			name: "no headers",
			args: args{
				ctx: context.Background(),
			},
			want: make(nats.Header),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HeadersFromContext(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HeadersFromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
