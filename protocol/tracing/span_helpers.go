package tracing

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/lavanet/lava/v5/protocol/lavasession"
)

// SmartRouterTracerName is the OTel instrumentation scope name used for all
// SmartRouter spans. Centralised here so call sites don't drift.
const SmartRouterTracerName = "smartrouter"

// tracer returns the smartrouter tracer. otel.Tracer is lazy-resolving so
// this is safe even before TraceManager.New() installs the global provider.
func tracer() trace.Tracer {
	return otel.Tracer(SmartRouterTracerName)
}

// StartServerSpan starts a SERVER-kind span. Use this for inbound request
// handling — e.g. the top-level relay span at the entry of SendRelay. There
// is no parent-recording guard here because a server span is by definition
// the start of a server-side trace; if no parent context is propagated the
// configured sampler decides whether to start a new root.
func StartServerSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return tracer().Start(ctx, name, trace.WithSpanKind(trace.SpanKindServer))
}

// StartClientSpan starts a CLIENT-kind child span only when the context
// already carries a recording parent. When called from a code path that
// has no parent span (e.g. internal init/crafted relays from
// sendCraftedRelays → sendRelayWithRetries), it returns the existing
// non-recording span so call sites can unconditionally defer span.End()
// without producing orphan root spans.
func StartClientSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	parent := trace.SpanFromContext(ctx)
	if !parent.IsRecording() {
		return ctx, parent
	}
	return tracer().Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))
}

// StartInternalSpan starts an INTERNAL-kind child span only when the
// context already carries a recording parent. Same orphan-root protection
// as StartClientSpan.
func StartInternalSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	parent := trace.SpanFromContext(ctx)
	if !parent.IsRecording() {
		return ctx, parent
	}
	return tracer().Start(ctx, name)
}

// --- Relay-span context plumbing ---

type relaySpanKey struct{}

// WithRelaySpan stores the inbound relay span on the context so deeply
// nested helpers (e.g. cache lookup) can decorate it directly without
// having to plumb the span through every function signature.
func WithRelaySpan(ctx context.Context, span trace.Span) context.Context {
	return context.WithValue(ctx, relaySpanKey{}, span)
}

// RelaySpanFromContext returns the inbound relay span previously stored
// via WithRelaySpan. When none was set it falls back to the current span
// in context (which may itself be a non-recording noop span).
func RelaySpanFromContext(ctx context.Context) trace.Span {
	if s, ok := ctx.Value(relaySpanKey{}).(trace.Span); ok {
		return s
	}
	return trace.SpanFromContext(ctx)
}

// --- Generic span attribute helpers ---

// RecordError records the error on the span and sets its status to Error.
func RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(otelcodes.Error, err.Error())
}

// RecordBody conditionally records a body attribute on the span when
// --otel-trace-body is enabled. We pre-truncate to the SDK's configured
// OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT so a large body with a small cap
// doesn't pay a full string allocation before the SDK truncates.
func RecordBody(span trace.Span, attrKey string, body []byte) {
	if !traceBodyEnabled || !span.IsRecording() {
		return
	}
	if bodyAttrLimit > 0 && len(body) > bodyAttrLimit {
		body = body[:bodyAttrLimit]
	}
	span.SetAttributes(attribute.String(attrKey, string(body)))
}

// RecordRelayAttributes records the standard relay identification attributes
// (GUID, chain ID, API interface) at the entry of a relay span.
func RecordRelayAttributes(span trace.Span, guid uint64, chainID, apiInterface string) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(
		attribute.String(attrRelayGUID, strconv.FormatUint(guid, 10)),
		attribute.String(attrRelayChainID, chainID),
		attribute.String(attrRelayAPIInterface, apiInterface),
	)
}

// RecordRelayMethod records the parsed API method name on the span.
func RecordRelayMethod(span trace.Span, methodName string) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(attribute.String(attrRelayMethod, methodName))
}

// RecordProviderAttributes records per-provider identification attributes
// (GUID and provider address) on a provider-scoped span.
func RecordProviderAttributes(span trace.Span, guid uint64, providerAddress string) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(
		attribute.String(attrRelayGUID, strconv.FormatUint(guid, 10)),
		attribute.String(attrProviderAddress, providerAddress),
	)
}

// RecordConsistencyStats records consistency filter counts on the span.
func RecordConsistencyStats(span trace.Span, total, passed, rejected int) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(
		attribute.Int(attrConsistencyTotal, total),
		attribute.Int(attrConsistencyPassed, passed),
		attribute.Int(attrConsistencyRejected, rejected),
	)
}

// RecordCacheResult records a cache lookup outcome (hit/miss) and its
// latency on the cache lookup child span, AND bubbles `cache.hit` up to
// the inbound relay span (if one is on the context) so TraceQL filters on
// the top-level trace (`{name="smartrouter.SendRelay" && cache.hit=true}`)
// work as operators expect.
func RecordCacheResult(ctx context.Context, span trace.Span, hit bool, latencyMs float64) {
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Bool(attrCacheHit, hit),
			attribute.Float64(attrCacheLatencyMs, latencyMs),
		)
	}
	if relaySpan := RelaySpanFromContext(ctx); relaySpan.IsRecording() {
		relaySpan.SetAttributes(attribute.Bool(attrCacheHit, hit))
	}
}

// RecordSessionStats records session manager requested/acquired counts.
func RecordSessionStats(span trace.Span, requested, acquired int) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(
		attribute.Int(attrSessionRequested, requested),
		attribute.Int(attrSessionAcquired, acquired),
	)
}

// RecordRetryCount records the total number of retry batches dispatched
// for a relay. Negative values are clamped to 0 (e.g. cache hit, no
// batches dispatched).
func RecordRetryCount(span trace.Span, retries int) {
	if !span.IsRecording() {
		return
	}
	if retries < 0 {
		retries = 0
	}
	span.SetAttributes(attribute.Int(attrRelayRetryCount, retries))
}

// RecordRelayAttempt records the 1-indexed batch number this provider
// call belongs to. All parallel calls in the same batch (cross-validation)
// share the same value.
func RecordRelayAttempt(span trace.Span, attempt int) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(attribute.Int(attrRelayAttempt, attempt))
}

// --- Semconv-aligned HTTP/gRPC client span helpers ---

// RecordHTTPRequest sets `http.method`, `http.url`, and (best-effort)
// `net.peer.name`/`net.peer.port` on a client-kind HTTP span. Call once
// when the span is started, before the request is dispatched, so the
// attributes are present even if the request later errors out.
func RecordHTTPRequest(span trace.Span, method, urlStr string) {
	if !span.IsRecording() {
		return
	}
	attrs := []attribute.KeyValue{
		semconv.HTTPMethod(method),
		semconv.HTTPURL(urlStr),
	}
	attrs = appendNetPeer(attrs, urlStr)
	span.SetAttributes(attrs...)
}

// RecordHTTPResponse records `http.status_code` and
// `http.response_content_length` on a client HTTP span. Skipped when
// response is nil (transport error before a response was received).
func RecordHTTPResponse(span trace.Span, response *lavasession.HTTPDirectRPCResponse) {
	if !span.IsRecording() || response == nil {
		return
	}
	span.SetAttributes(
		semconv.HTTPStatusCode(response.StatusCode),
		semconv.HTTPResponseContentLength(len(response.Body)),
	)
}

// RecordGRPCRequest sets `rpc.system`="grpc", `rpc.service`, `rpc.method`,
// and (best-effort) `net.peer.name`/`net.peer.port` on a client-kind gRPC
// span. methodPath is split into service+method by the last "/" if present
// (e.g. "cosmos.bank.v1beta1.Query/Balance" → service="cosmos.bank.v1beta1.Query",
// method="Balance"). Paths without a slash are recorded as method only.
func RecordGRPCRequest(span trace.Span, methodPath, urlStr string) {
	if !span.IsRecording() {
		return
	}
	service, method := splitGRPCMethod(methodPath)
	attrs := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		semconv.RPCMethod(method),
	}
	if service != "" {
		attrs = append(attrs, semconv.RPCService(service))
	}
	attrs = appendNetPeer(attrs, urlStr)
	span.SetAttributes(attrs...)
}

// RecordGRPCResponse records the gRPC response status and payload size on
// a client gRPC span. Skipped when response is nil. We reuse the
// http.status_code attribute because the SmartRouter's DirectRPCResponse
// surfaces a unified status field across protocols.
func RecordGRPCResponse(span trace.Span, response *lavasession.DirectRPCResponse) {
	if !span.IsRecording() || response == nil {
		return
	}
	span.SetAttributes(
		semconv.HTTPStatusCode(response.StatusCode),
		semconv.HTTPResponseContentLength(len(response.Data)),
	)
}

// appendNetPeer parses a URL and appends `net.peer.name`/`net.peer.port`
// attributes when the URL has a hostname. Best-effort: malformed URLs and
// missing ports are silently skipped.
func appendNetPeer(attrs []attribute.KeyValue, urlStr string) []attribute.KeyValue {
	if urlStr == "" {
		return attrs
	}
	u, err := url.Parse(urlStr)
	if err != nil || u.Hostname() == "" {
		return attrs
	}
	attrs = append(attrs, semconv.NetPeerName(u.Hostname()))
	if port := u.Port(); port != "" {
		if p, perr := strconv.Atoi(port); perr == nil {
			attrs = append(attrs, semconv.NetPeerPort(p))
		}
	}
	return attrs
}

// splitGRPCMethod splits a gRPC method path of the form
// "package.Service/Method" into ("package.Service", "Method"). When the
// path has no "/", the entire string is treated as the method.
func splitGRPCMethod(methodPath string) (service, method string) {
	if i := strings.LastIndex(methodPath, "/"); i >= 0 {
		return strings.TrimPrefix(methodPath[:i], "/"), methodPath[i+1:]
	}
	return "", methodPath
}
