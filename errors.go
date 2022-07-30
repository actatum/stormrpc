package stormrpc

import (
	"errors"
	"fmt"
)

// ErrorCode represents an enum type for stormRPC error codes.
type ErrorCode int

// RPC ErrorCodes.
const (
	ErrorCodeUnknown          ErrorCode = 0
	ErrorCodeInternal         ErrorCode = 1
	ErrorCodeNotFound         ErrorCode = 2
	ErrorCodeInvalidArgument  ErrorCode = 3
	ErrorCodeUnimplemented    ErrorCode = 4
	ErrorCodeUnauthenticated  ErrorCode = 5
	ErrorCodePermissionDenied ErrorCode = 6
	ErrorCodeAlreadyExists    ErrorCode = 7
)

func (c ErrorCode) String() string {
	switch c {
	case ErrorCodeInternal:
		return "STORMRPC_CODE_INTERNAL"
	case ErrorCodeNotFound:
		return "STORMRPC_CODE_NOT_FOUND"
	case ErrorCodeInvalidArgument:
		return "STORMRPC_CODE_INVALID_ARGUMENT"
	case ErrorCodeUnimplemented:
		return "STORMRPC_CODE_UNIMPLEMENTED"
	case ErrorCodeUnauthenticated:
		return "STORMRPC_CODE_UNAUTHENTICATED"
	case ErrorCodePermissionDenied:
		return "STORMRPC_CODE_PERMISSION_DENIED"
	case ErrorCodeAlreadyExists:
		return "STORMRPC_CODE_ALREADY_EXISTS"
	default:
		return "STORMRPC_CODE_UNKNOWN"
	}
}

// Error represents an RPC error.
type Error struct {
	Code    ErrorCode
	Message string
}

// Error allows for the Error type to conform to the built-in error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code.String(), e.Message)
}

// Errorf constructs a new RPC Error.
func Errorf(code ErrorCode, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// CodeFromErr retrieves the ErrorCode from a given error.
// If the error is not of type Error, ErrorCodeUnknown is returned.
func CodeFromErr(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ErrorCodeUnknown
}

// MessageFromErr retrieves the message from a given error.
// If the error is not of type Error, "unknown error" is returned.
func MessageFromErr(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Message
	}
	return "unknown error"
}

func codeFromString(s string) ErrorCode {
	switch s {
	case "STORMRPC_CODE_INTERNAL":
		return ErrorCodeInternal
	case "STORMRPC_CODE_NOT_FOUND":
		return ErrorCodeNotFound
	default:
		return ErrorCodeUnknown
	}
}
