package tracing

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StartProviderHandleRelay starts the top-level provider relay span. The
// caller is the otelgrpc server interceptor's already-rooted ctx (so this
// span is a child of the auto-generated SERVER span).
func StartProviderHandleRelay(ctx context.Context, guid uint64, chainID, apiInterface string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderHandleRelay,
		trace.WithAttributes(
			attribute.String(attrRelayGUID, strconv.FormatUint(guid, 10)),
			attribute.String(attrRelayChainID, chainID),
			attribute.String(attrRelayAPIInterface, apiInterface),
		),
	)
}

// StartProviderInitRelay starts the initRelay span (parsing + session lookup).
func StartProviderInitRelay(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderInitRelay)
}

// StartProviderValidateRequest starts the validation span (signature, addons).
func StartProviderValidateRequest(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderValidateRequest)
}

// StartProviderCacheLookup starts the provider-side cache-lookup span.
// Caller fills attributes with RecordCacheLookup from spans_consumer.go
// (passing "provider" as the role).
func StartProviderCacheLookup(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderCacheLookup)
}

// StartProviderHandleConsistency starts the block-wait span. targetBlock is
// the requestBlock or seenBlock the provider must reach.
func StartProviderHandleConsistency(ctx context.Context, targetBlock int64) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderHandleConsistency,
		trace.WithAttributes(attribute.Int64(attrConsistencyTargetBlock, targetBlock)),
	)
}

// StartProviderTryRelay starts the relay-execution span (parent for SendUpstream).
func StartProviderTryRelay(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderTryRelay)
}

// StartProviderSendUpstream starts the chainlib upstream-call span. The
// chainlib-internal sendJSONRPCRelay/sendRESTRelay/sendGRPCRelay spans
// (already shared between SR and provider) become children of this span.
func StartProviderSendUpstream(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderSendUpstream)
}

// StartProviderCacheStore starts the cache-write span.
func StartProviderCacheStore(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderCacheStore)
}

// StartProviderFinalizeSession starts the session-finalize span (accounting + signing).
func StartProviderFinalizeSession(ctx context.Context) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanProviderFinalizeSession)
}

// RecordHandleConsistencyResult attaches consistency.waited_ms and consistency.bailed
// to a HandleConsistency span once the wait loop returns.
//
// Distinct from RecordConsistencyStats (in span_helpers.go), which is for
// the consumer's cross-validation total/passed/rejected counts.
func RecordHandleConsistencyResult(span trace.Span, waitedMs int64, bailed bool) {
	span.SetAttributes(
		attribute.Int64(attrConsistencyWaitedMs, waitedMs),
		attribute.Bool(attrConsistencyBailed, bailed),
	)
}
