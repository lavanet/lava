package tracing

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StartConsumerSendRelay starts the top-level consumer relay span. The caller
// is the chainlib middleware's already-rooted ctx (so this span is a child
// of chainlib.HandleHTTP). Caller `defer span.End()` and use Record* helpers.
func StartConsumerSendRelay(ctx context.Context, guid uint64, chainID, apiInterface string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanConsumerSendRelay,
		trace.WithAttributes(
			attribute.String(attrRelayGUID, strconv.FormatUint(guid, 10)),
			attribute.String(attrRelayChainID, chainID),
			attribute.String(attrRelayAPIInterface, apiInterface),
		),
	)
}

// StartConsumerParseRelay starts the parse-relay span (child of SendRelay).
func StartConsumerParseRelay(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanConsumerParseRelay)
}

// StartConsumerSendParsedRelay starts the send-parsed-relay span (parent for ProcessRelaySend).
func StartConsumerSendParsedRelay(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanConsumerSendParsedRelay)
}

// StartConsumerProcessRelaySend wraps the state-machine retry loop. Caller sets
// relay.retry_count via the existing RecordRetryCount helper (in span_helpers.go)
// once the loop has resolved.
func StartConsumerProcessRelaySend(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanConsumerProcessRelaySend)
}

// StartConsumerCacheLookup starts the consumer-side cache-lookup span.
// Caller fills attributes with RecordCacheLookup.
func StartConsumerCacheLookup(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanConsumerCacheLookup)
}

// StartConsumerGetSessions starts the session-acquisition span.
// Caller records via RecordSessionStats (from span_helpers.go) once the optimizer returns.
func StartConsumerGetSessions(ctx context.Context, requested int) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanConsumerGetSessions,
		trace.WithAttributes(attribute.Int(attrSessionRequested, requested)),
	)
}

// StartConsumerRelayInner starts the per-attempt span around the gRPC call to
// the provider. The CLIENT span (auto-created by otelgrpc on the underlying
// dial) will be nested under this span.
//
// The caller passes the current batch number (1-indexed), typically derived
// from `relayProcessor.GetUsedProviders().BatchNumber()` — same pattern SR
// uses in rpcsmartrouter_server.go:838.
func StartConsumerRelayInner(ctx context.Context, providerAddr string, attempt int) (context.Context, trace.Span) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, SpanConsumerRelayInner,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.String(attrProviderAddress, providerAddr)),
	)
	RecordRelayAttempt(span, attempt) // existing helper in span_helpers.go
	return ctx, span
}

// StartConsumerProcessingResult starts the consensus / finalization span.
func StartConsumerProcessingResult(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanConsumerProcessingResult)
}

// RecordCacheLookup attaches role + hit + latency to a cache-lookup span and
// bubbles cache.hit to the surrounding relay span via context (the existing
// behavior of RecordCacheResult in span_helpers.go). Use this for both
// consumer- and provider-side cache spans: role disambiguates them on a
// single trace.
//
// role is "consumer" or "provider".
func RecordCacheLookup(ctx context.Context, span trace.Span, role string, hit bool, latencyMs float64) {
	if span.IsRecording() {
		span.SetAttributes(attribute.String(attrCacheRole, role))
	}
	RecordCacheResult(ctx, span, hit, latencyMs)
}
