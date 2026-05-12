package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StartChainlibHTTP starts the root SERVER span for an inbound HTTP request
// at a chainlib listener (jsonRPC or REST). Trace context extraction from
// inbound headers is handled by the caller (Fiber middleware) before this
// helper is invoked, so ctx is already parent-aware.
func StartChainlibHTTP(ctx context.Context, method, route string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanChainlibHTTP,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", route),
		),
	)
}

// StartChainlibGRPC starts the root SERVER span for an inbound gRPC request
// at chainlib's gRPC proxy listener. Used when the otelgrpc server interceptor
// is not in play (rare — the interceptor is the preferred mechanism).
func StartChainlibGRPC(ctx context.Context, method string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanChainlibGRPC,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("rpc.method", method)),
	)
}
