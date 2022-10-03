// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func Test_parseErrorHeader(t *testing.T) {
	type args struct {
		header nats.Header
	}
	tests := []struct {
		name string
		args args
		want *Error
	}{
		{
			name: "no error header",
			args: args{
				header: nats.Header{},
			},
			want: nil,
		},
		{
			name: "weirdly formatted error header",
			args: args{
				header: nats.Header{
					errorHeader: []string{"BIG HEADER", "NICE ERROR"},
				},
			},
			want: &Error{
				Code:    ErrorCodeUnknown,
				Message: "unknown error",
			},
		},
		{
			name: "not found error",
			args: args{
				header: nats.Header{
					errorHeader: []string{"STORMRPC_CODE_NOT_FOUND: new error"},
				},
			},
			want: &Error{
				Code:    ErrorCodeNotFound,
				Message: "new error",
			},
		},
		{
			name: "unknown error",
			args: args{
				header: nats.Header{
					errorHeader: []string{"STORMRPC_CODE_UNKNOWN: xD"},
				},
			},
			want: &Error{
				Code:    ErrorCodeUnknown,
				Message: "unknown error",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseErrorHeader(tt.args.header); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseErrorHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseDeadlineHeader(t *testing.T) {
	type args struct {
		header nats.Header
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		// TODO: Add test cases.
		{
			name: "no header",
			args: args{
				header: nats.Header{},
			},
			want: time.Time{},
		},
		{
			name: "header non int",
			args: args{
				header: nats.Header{
					deadlineHeader: []string{"bob"},
				},
			},
			want: time.Time{},
		},
		{
			name: "header with unix time",
			args: args{
				header: nats.Header{
					deadlineHeader: []string{strconv.FormatInt(time.Now().Round(1*time.Minute).Unix(), 10)},
				},
			},
			want: time.Now().Round(1 * time.Minute),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDeadlineHeader(tt.args.header); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDeadlineHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}
