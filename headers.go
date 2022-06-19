package stormrpc

import (
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	errorHeader    = "stormrpc-error"
	deadlineHeader = "stormrpc-deadline"
)

func parseDeadlineHeader(header nats.Header) time.Time {
	dh := header.Get(deadlineHeader)
	if dh == "" {
		return time.Time{}
	}

	i, err := strconv.ParseInt(dh, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(i, 0)
}

func parseErrorHeader(header nats.Header) *Error {
	eh := header.Get(errorHeader)
	if eh == "" {
		return nil
	}

	sp := strings.Split(eh, ":")

	if len(sp) < 2 {
		return &Error{
			Code:    ErrorCodeUnknown,
			Message: "unknown error",
		}
	}

	code := codeFromString(strings.TrimSpace(sp[0]))
	msg := strings.TrimSpace(sp[1])

	if code == ErrorCodeUnknown {
		msg = "unknown error"
	}

	return &Error{
		Code:    code,
		Message: msg,
	}
}
