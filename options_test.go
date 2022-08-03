package stormrpc

import (
	"reflect"
	"testing"
)

func TestWithHeaders(t *testing.T) {
	type args struct {
		h map[string]string
	}
	tests := []struct {
		name string
		args args
		want CallOption
	}{
		// TODO: Add test cases.
		{
			name: "add some headers",
			args: args{
				h: map[string]string{
					"Authorization": "Bearer ey.xyz",
					"X-Request-Id":  "abc",
				},
			},
			want: &HeaderCallOption{
				Headers: map[string]string{
					"Authorization": "Bearer ey.xyz",
					"X-Request-Id":  "abc",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithHeaders(tt.args.h); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}
