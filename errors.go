package stormrpc

import (
	"errors"
	"fmt"
)

type ErrorCode int

const (
	ErrorCodeUnknown  ErrorCode = 0
	ErrorCodeInternal ErrorCode = 1
	ErrorCodeNotFound ErrorCode = 2
)

func (c ErrorCode) String() string {
	switch c {
	case ErrorCodeInternal:
		return "STORMRPC_CODE_INTERNAL"
	case ErrorCodeNotFound:
		return "STORMRPC_CODE_NOT_FOUND"
	default:
		return "STORMRPC_CODE_UNKNOWN"
	}
}

type Error struct {
	Code    ErrorCode
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code.String(), e.Message)
}

func Errorf(code ErrorCode, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

func CodeFromErr(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ErrorCodeUnknown
}

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
