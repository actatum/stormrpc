// Package stormrpc provides the functionality for creating RPC servers/clients that communicate via NATS.
package stormrpc

import (
	"fmt"
	"testing"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name string
		c    ErrorCode
		want string
	}{
		{
			name: "unknown",
			c:    ErrorCodeUnknown,
			want: "STORMRPC_CODE_UNKNOWN",
		},
		{
			name: "internal",
			c:    ErrorCodeInternal,
			want: "STORMRPC_CODE_INTERNAL",
		},
		{
			name: "not found",
			c:    ErrorCodeNotFound,
			want: "STORMRPC_CODE_NOT_FOUND",
		},
		{
			name: "invalid argument",
			c:    ErrorCodeInvalidArgument,
			want: "STORMRPC_CODE_INVALID_ARGUMENT",
		},
		{
			name: "unimplemented",
			c:    ErrorCodeUnimplemented,
			want: "STORMRPC_CODE_UNIMPLEMENTED",
		},
		{
			name: "unauthenticated",
			c:    ErrorCodeUnauthenticated,
			want: "STORMRPC_CODE_UNAUTHENTICATED",
		},
		{
			name: "permission denied",
			c:    ErrorCodePermissionDenied,
			want: "STORMRPC_CODE_PERMISSION_DENIED",
		},
		{
			name: "already exists",
			c:    ErrorCodeAlreadyExists,
			want: "STORMRPC_CODE_ALREADY_EXISTS",
		},
		{
			name: "default",
			c:    10000,
			want: "STORMRPC_CODE_UNKNOWN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	type fields struct {
		Code    ErrorCode
		Message string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "print error message",
			fields: fields{
				Code:    ErrorCodeNotFound,
				Message: "thing not found",
			},
			want: "STORMRPC_CODE_NOT_FOUND: thing not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Error{
				Code:    tt.fields.Code,
				Message: tt.fields.Message,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeFromErr(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want ErrorCode
	}{
		{
			name: "non stormrpc error",
			args: args{
				err: fmt.Errorf("howdy"),
			},
			want: ErrorCodeUnknown,
		},
		{
			name: "stormrpc error",
			args: args{
				err: Errorf(ErrorCodeNotFound, "hi"),
			},
			want: ErrorCodeNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeFromErr(tt.args.err); got != tt.want {
				t.Errorf("CodeFromErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageFromErr(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "non stormrpc error",
			args: args{
				err: fmt.Errorf("hi"),
			},
			want: "unknown error",
		},
		{
			name: "stormrpc error",
			args: args{
				err: Errorf(ErrorCodeNotFound, "err"),
			},
			want: "err",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MessageFromErr(tt.args.err); got != tt.want {
				t.Errorf("MessageFromErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_codeFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want ErrorCode
	}{
		{
			name: "default",
			args: args{
				s: "asijdfoaijdsfoaijdf",
			},
			want: ErrorCodeUnknown,
		},
		{
			name: "internal",
			args: args{
				s: "STORMRPC_CODE_INTERNAL",
			},
			want: ErrorCodeInternal,
		},
		{
			name: "not found",
			args: args{
				s: "STORMRPC_CODE_NOT_FOUND",
			},
			want: ErrorCodeNotFound,
		},
		{
			name: "invalid argument",
			args: args{
				s: "STORMRPC_CODE_INVALID_ARGUMENT",
			},
			want: ErrorCodeInvalidArgument,
		},
		{
			name: "unimplemented",
			args: args{
				s: "STORMRPC_CODE_UNIMPLEMENTED",
			},
			want: ErrorCodeUnimplemented,
		},
		{
			name: "unauthenticated",
			args: args{
				s: "STORMRPC_CODE_UNAUTHENTICATED",
			},
			want: ErrorCodeUnauthenticated,
		},
		{
			name: "permission denied",
			args: args{
				s: "STORMRPC_CODE_PERMISSION_DENIED",
			},
			want: ErrorCodePermissionDenied,
		},
		{
			name: "already exists",
			args: args{
				s: "STORMRPC_CODE_ALREADY_EXISTS",
			},
			want: ErrorCodeAlreadyExists,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeFromString(tt.args.s); got != tt.want {
				t.Errorf("codeFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
