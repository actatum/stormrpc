package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/actatum/stormrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTracing(t *testing.T) {
	tp := tracesdk.NewTracerProvider()
	tr := tp.Tracer("")

	t.Run("no header present", func(t *testing.T) {
		req, _ := stormrpc.NewRequest(context.Background(), "test", map[string]string{"howdy": "partner"})
		handler := stormrpc.HandlerFunc(func(r stormrpc.Request) stormrpc.Response {
			ctx := otel.GetTextMapPropagator().Extract(r.Context, propagation.HeaderCarrier(r.Header))
			span := trace.SpanFromContext(ctx)
			if span.SpanContext().IsRemote() {
				t.Fatal("expected span not to be remote")
			}

			return stormrpc.NewErrorResponse("test", fmt.Errorf("hi"))
		})
		h := Tracing(tr)(handler)
		h(req)
	})

	t.Run("header present", func(t *testing.T) {
		req, _ := stormrpc.NewRequest(context.Background(), "test", map[string]string{"howdy": "partner"})
		req.Header.Set("Traceparent", "00-5c8a51cfe13f1a03d87bcee2f870c518-82fc7641b3345990-01")
		handler := stormrpc.HandlerFunc(func(r stormrpc.Request) stormrpc.Response {
			ctx := otel.GetTextMapPropagator().Extract(r.Context, propagation.HeaderCarrier(r.Header))
			span := trace.SpanFromContext(ctx)
			if !span.SpanContext().IsRemote() {
				t.Fatal("expected span to be remote")
			}

			return stormrpc.NewErrorResponse("test", fmt.Errorf("hi"))
		})
		h := Tracing(tr)(handler)
		h(req)
	})

	t.Run("header is on response", func(t *testing.T) {
		var want string
		req, _ := stormrpc.NewRequest(context.Background(), "test", map[string]string{"howdy": "partner"})
		handler := stormrpc.HandlerFunc(func(r stormrpc.Request) stormrpc.Response {
			ctx := otel.GetTextMapPropagator().Extract(r.Context, propagation.HeaderCarrier(r.Header))
			span := trace.SpanFromContext(ctx)
			if span.SpanContext().IsRemote() {
				t.Fatal("expected span to be remote")
			}

			want = fmt.Sprintf("%s-%s-%s-%s",
				"00",
				span.SpanContext().TraceID(),
				span.SpanContext().SpanID(),
				span.SpanContext().TraceFlags().String(),
			)

			return stormrpc.NewErrorResponse("test", fmt.Errorf("hi"))
		})
		h := Tracing(tr)(handler)
		resp := h(req)

		got := resp.Header.Get("Traceparent")
		if got != want {
			t.Fatalf("got = %v, want %v", got, want)
		}
	})
}
