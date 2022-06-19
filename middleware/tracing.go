package middleware

import (
	"github.com/actatum/stormrpc"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Tracing extracts the span from the incoming request headers. If none is present a new root span is created.
// This tracing information is also passed into the response headers.
func Tracing(tracer trace.Tracer) func(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{}),
	)
	return func(next stormrpc.HandlerFunc) stormrpc.HandlerFunc {
		return func(r stormrpc.Request) stormrpc.Response {
			ctx := otel.GetTextMapPropagator().Extract(r.Context, propagation.HeaderCarrier(r.Header))
			spanCtx, serverSpan := tracer.Start(
				ctx,
				r.Subject(),
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer serverSpan.End()

			r.Context = trace.ContextWithSpan(r.Context, serverSpan)

			resp := next(r)

			if resp.Header == nil {
				resp.Header = nats.Header{}
			}
			otel.GetTextMapPropagator().Inject(spanCtx, propagation.HeaderCarrier(resp.Header))

			return resp
		}
	}
}
