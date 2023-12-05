// Package middleware provides some useful and commonly implemented middleware functions for StormRPC servers.
package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/actatum/stormrpc"
	"go.opentelemetry.io/otel/trace"
)

// Logger logs request scoped information such as request id, trace information, and request duration.
// This middleware should be applied after RequestID, and Tracing.
func Logger(l *slog.Logger) func(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
	return func(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
		return func(ctx context.Context, r stormrpc.Request) stormrpc.Response {
			span := trace.SpanFromContext(ctx)
			id := RequestIDFromContext(ctx)

			start := time.Now()

			resp := next(ctx, r)

			attrs := make([]slog.Attr, 0)

			level := slog.LevelInfo
			msg := "Success"
			if resp.Err != nil {
				msg = "Server Error"
				level = slog.LevelError
				code := stormrpc.CodeFromErr(resp.Err)
				attrs = append(attrs, slog.Group(
					"error",
					slog.String("message", resp.Err.Error()),
					slog.String("code", code.String()),
				))
			}

			attrs = append(attrs, slog.Group("request",
				slog.String("id", id),
				slog.String("trace_id", span.SpanContext().TraceID().String()),
				slog.String("duration", time.Since(start).String()),
			))

			l.LogAttrs(
				ctx,
				level,
				msg,
				attrs...,
			)

			return resp
		}
	}
}
