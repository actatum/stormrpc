package middleware

import (
	"time"

	"github.com/actatum/stormrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func Logger(l *zap.Logger) func(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
	return func(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
		return func(r stormrpc.Request) stormrpc.Response {
			span := trace.SpanFromContext(r.Context)
			id := RequestIDFromContext(r.Context)

			start := time.Now()

			resp := next(r)

			fields := []zap.Field{
				zap.String("id", id),
				zap.String("trace_id", span.SpanContext().TraceID().String()),
				zap.String("span_id", span.SpanContext().SpanID().String()),
				zap.String("duration", time.Since(start).String()),
			}

			if resp.Err != nil {
				code := stormrpc.CodeFromErr(resp.Err)
				fields = append(fields, zap.String("code", code.String()))
				l.Error("Server Error", fields...)
			} else {
				l.Info("Success", fields...)
			}

			return resp
		}
	}
}
