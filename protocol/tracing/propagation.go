package tracing

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// InjectHTTP injects W3C trace context headers (traceparent/tracestate)
// into a Metadata slice. Existing entries with matching (case-insensitive)
// names are replaced in place; remaining propagator keys are appended.
// Replacing rather than blindly appending avoids producing two
// `traceparent` headers downstream when an inbound traceparent has been
// passed through by the chain spec — receivers cannot reliably pick the
// correct one in that case.
func InjectHTTP(ctx context.Context, headers []pairingtypes.Metadata) []pairingtypes.Metadata {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if len(carrier) == 0 {
		return headers
	}
	// The W3C TraceContext / Baggage propagators emit canonical lowercase
	// keys ("traceparent", "tracestate", "baggage"), so we can index the
	// carrier with a lowercased header name directly.
	// Overwrite every case-insensitive match so duplicates (e.g. both
	// "traceparent" and "Traceparent") all receive the fresh value rather
	// than leaving the second occurrence stale.
	seen := make(map[string]struct{}, len(carrier))
	for i := range headers {
		lower := strings.ToLower(headers[i].Name)
		if v, ok := carrier[lower]; ok {
			headers[i].Value = v
			seen[lower] = struct{}{}
		}
	}
	for k, v := range carrier {
		if _, ok := seen[k]; ok {
			continue
		}
		headers = append(headers, pairingtypes.Metadata{Name: k, Value: v})
	}
	return headers
}

// InjectGRPC sets W3C trace context headers (traceparent/tracestate)
// in a string map used for gRPC-style metadata.
func InjectGRPC(ctx context.Context, headers map[string]string) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		headers[k] = v
	}
}

// ExtractGRPC reads W3C trace context from gRPC incoming-context metadata into ctx,
// returning a new context that carries the remote span as parent. Returns ctx
// unchanged when no metadata is present.
//
// For multi-valued metadata keys, only the first value is used. This is
// sufficient for W3C trace context (traceparent/tracestate/baggage are
// single-valued per spec); non-conforming senders that repeat these headers
// will have additional values silently dropped.
func ExtractGRPC(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	headers := make(map[string]string, md.Len())
	for k, vs := range md {
		if len(vs) > 0 {
			headers[strings.ToLower(k)] = vs[0]
		}
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
}

// ExtractHTTP reads W3C trace context from a Metadata slice into ctx,
// returning a new context that carries the remote span as parent.
// Keys are lowercased because HTTP/1.1 headers may arrive in Title-Case
// while the W3C propagator expects lowercase keys.
func ExtractHTTP(ctx context.Context, metadata []pairingtypes.Metadata) context.Context {
	headers := make(map[string]string, len(metadata))
	for _, md := range metadata {
		headers[strings.ToLower(md.Name)] = md.Value
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
}
