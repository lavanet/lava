package tracing

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// setupTestTracing installs a real TracerProvider and W3C propagator for the
// duration of a single test, then restores the previous globals on cleanup.
func setupTestTracing(t *testing.T) trace.Tracer {
	t.Helper()

	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Tracer("test")
}

func TestInjectHTTP(t *testing.T) {
	tracer := setupTestTracing(t)

	activeCtx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	tests := []struct {
		name            string
		ctx             context.Context
		initial         []pairingtypes.Metadata
		expectAdded     bool // whether traceparent should be appended
		expectPreserved bool // whether initial headers survive
	}{
		{
			name:            "active span appends traceparent",
			ctx:             activeCtx,
			initial:         []pairingtypes.Metadata{{Name: "X-Custom", Value: "keep"}},
			expectAdded:     true,
			expectPreserved: true,
		},
		{
			name:        "active span with nil slice",
			ctx:         activeCtx,
			initial:     nil,
			expectAdded: true,
		},
		{
			name:    "no span injects nothing",
			ctx:     context.Background(),
			initial: nil,
		},
		{
			name:            "no span preserves existing headers",
			ctx:             context.Background(),
			initial:         []pairingtypes.Metadata{{Name: "X-Keep", Value: "v"}},
			expectPreserved: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := InjectHTTP(tc.ctx, tc.initial)

			if tc.expectPreserved {
				require.Equal(t, tc.initial[0].Name, result[0].Name)
				require.Equal(t, tc.initial[0].Value, result[0].Value)
			}

			traceparent := findMetadata(result, "traceparent")
			if tc.expectAdded {
				require.NotEmpty(t, traceparent, "traceparent must be injected")
				require.Contains(t, traceparent, span.SpanContext().TraceID().String())
			} else {
				require.Empty(t, traceparent, "traceparent must not be injected")
			}
		})
	}
}

func TestInjectGRPC(t *testing.T) {
	tracer := setupTestTracing(t)

	activeCtx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	tests := []struct {
		name        string
		ctx         context.Context
		initial     map[string]string
		expectAdded bool
	}{
		{
			name:        "active span sets traceparent",
			ctx:         activeCtx,
			initial:     map[string]string{"existing": "value"},
			expectAdded: true,
		},
		{
			name:        "active span on empty map",
			ctx:         activeCtx,
			initial:     map[string]string{},
			expectAdded: true,
		},
		{
			name:    "no span sets nothing",
			ctx:     context.Background(),
			initial: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			InjectGRPC(tc.ctx, tc.initial)

			if tc.expectAdded {
				require.Contains(t, tc.initial, "traceparent")
				require.Contains(t, tc.initial["traceparent"], span.SpanContext().TraceID().String())
			} else {
				require.NotContains(t, tc.initial, "traceparent")
			}

			// Existing keys must survive.
			if _, ok := tc.initial["existing"]; ok {
				require.Equal(t, "value", tc.initial["existing"])
			}
		})
	}
}

func TestExtractHTTP(t *testing.T) {
	tracer := setupTestTracing(t)

	// Build a valid traceparent by injecting from a real span.
	ctx, span := tracer.Start(context.Background(), "source-span")
	defer span.End()
	injected := InjectHTTP(ctx, nil)
	traceparentVal := findMetadata(injected, "traceparent")
	require.NotEmpty(t, traceparentVal, "setup: need a valid traceparent")

	tests := []struct {
		name        string
		metadata    []pairingtypes.Metadata
		expectValid bool // whether extracted span context is valid
	}{
		{
			name:        "lowercase traceparent",
			metadata:    []pairingtypes.Metadata{{Name: "traceparent", Value: traceparentVal}},
			expectValid: true,
		},
		{
			name:        "Title-Case traceparent (HTTP/1.1 style)",
			metadata:    []pairingtypes.Metadata{{Name: "Traceparent", Value: traceparentVal}},
			expectValid: true,
		},
		{
			name:        "empty metadata",
			metadata:    nil,
			expectValid: false,
		},
		{
			name:        "no trace headers",
			metadata:    []pairingtypes.Metadata{{Name: "X-Custom", Value: "irrelevant"}},
			expectValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			extracted := ExtractHTTP(context.Background(), tc.metadata)
			sc := trace.SpanContextFromContext(extracted)

			if tc.expectValid {
				require.True(t, sc.IsValid(), "span context must be valid")
				require.Equal(t, span.SpanContext().TraceID(), sc.TraceID())
				require.True(t, sc.IsRemote(), "extracted span context must be marked remote")
			} else {
				require.False(t, sc.IsValid(), "span context must not be valid")
			}
		})
	}
}

func TestRoundTrip_HTTP(t *testing.T) {
	tracer := setupTestTracing(t)
	ctx, span := tracer.Start(context.Background(), "roundtrip-span")
	defer span.End()

	// Inject → Extract must preserve trace ID.
	metadata := InjectHTTP(ctx, nil)
	extracted := ExtractHTTP(context.Background(), metadata)

	sc := trace.SpanContextFromContext(extracted)
	require.True(t, sc.IsValid())
	require.Equal(t, span.SpanContext().TraceID(), sc.TraceID())
}

// TestInjectHTTP_OverwritesExistingTraceparent guards G4: if a chain spec
// passes the inbound traceparent through to the outbound provider request,
// InjectHTTP must REPLACE it (not append a duplicate). Receivers cannot
// reliably pick the right one when there are two `traceparent` headers.
func TestInjectHTTP_OverwritesExistingTraceparent(t *testing.T) {
	tracer := setupTestTracing(t)
	ctx, span := tracer.Start(context.Background(), "outbound")
	defer span.End()

	stale := "00-00000000000000000000000000000001-0000000000000001-01"
	tests := []struct {
		name    string
		headers []pairingtypes.Metadata
	}{
		{
			name: "lowercase traceparent already present",
			headers: []pairingtypes.Metadata{
				{Name: "X-Custom", Value: "keep"},
				{Name: "traceparent", Value: stale},
			},
		},
		{
			name: "Title-Case Traceparent already present",
			headers: []pairingtypes.Metadata{
				{Name: "X-Custom", Value: "keep"},
				{Name: "Traceparent", Value: stale},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := InjectHTTP(ctx, tc.headers)

			// Exactly one traceparent must remain (case-insensitive).
			var count int
			var injected string
			for _, md := range result {
				if strings.EqualFold(md.Name, "traceparent") {
					count++
					injected = md.Value
				}
			}
			require.Equal(t, 1, count, "must end up with exactly one traceparent header")
			require.NotEqual(t, stale, injected, "stale traceparent must be replaced")
			require.Contains(t, injected, span.SpanContext().TraceID().String())

			// Unrelated headers must survive.
			require.Equal(t, "keep", findMetadata(result, "X-Custom"))
		})
	}
}

// TestRoundTrip_HTTP_Tracestate confirms tracestate (the W3C "ts" sibling
// header to traceparent) survives inject → extract via the HTTP carrier
// helpers. The product spec explicitly calls out tracestate forwarding,
// so this round-trip is asserted with its own test.
func TestRoundTrip_HTTP_Tracestate(t *testing.T) {
	tracer := setupTestTracing(t)

	ts, err := trace.ParseTraceState("vendor=value")
	require.NoError(t, err)
	parentCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
		TraceFlags: trace.FlagsSampled,
		TraceState: ts,
		Remote:     true,
	}))
	ctx, span := tracer.Start(parentCtx, "child")
	defer span.End()

	metadata := InjectHTTP(ctx, nil)

	require.NotEmpty(t, findMetadata(metadata, "traceparent"), "traceparent must be present")
	require.NotEmpty(t, findMetadata(metadata, "tracestate"), "tracestate must be present")
	require.Contains(t, findMetadata(metadata, "tracestate"), "vendor=value")

	extracted := ExtractHTTP(context.Background(), metadata)
	sc := trace.SpanContextFromContext(extracted)
	require.True(t, sc.IsValid())
	require.Equal(t, "vendor=value", sc.TraceState().String())
}

// TestRoundTrip_GRPC_Tracestate confirms tracestate also survives the
// gRPC carrier helpers.
func TestRoundTrip_GRPC_Tracestate(t *testing.T) {
	tracer := setupTestTracing(t)

	ts, err := trace.ParseTraceState("vendor=value")
	require.NoError(t, err)
	parentCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0x0},
		SpanID:     trace.SpanID{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
		TraceFlags: trace.FlagsSampled,
		TraceState: ts,
		Remote:     true,
	}))
	ctx, span := tracer.Start(parentCtx, "child")
	defer span.End()

	headers := map[string]string{}
	InjectGRPC(ctx, headers)

	require.Contains(t, headers, "traceparent")
	require.Contains(t, headers, "tracestate")
	require.Contains(t, headers["tracestate"], "vendor=value")

	extracted := otel.GetTextMapPropagator().Extract(
		context.Background(), propagation.MapCarrier(headers),
	)
	sc := trace.SpanContextFromContext(extracted)
	require.True(t, sc.IsValid())
	require.Equal(t, "vendor=value", sc.TraceState().String())
}

func TestRoundTrip_GRPC(t *testing.T) {
	tracer := setupTestTracing(t)
	ctx, span := tracer.Start(context.Background(), "roundtrip-span")
	defer span.End()

	// InjectGRPC → manual extract via propagator (simulates receiver).
	headers := map[string]string{}
	InjectGRPC(ctx, headers)

	extracted := otel.GetTextMapPropagator().Extract(
		context.Background(), propagation.MapCarrier(headers),
	)

	sc := trace.SpanContextFromContext(extracted)
	require.True(t, sc.IsValid())
	require.Equal(t, span.SpanContext().TraceID(), sc.TraceID())
}

// findMetadata returns the value of the first metadata entry matching the given name.
func findMetadata(metadata []pairingtypes.Metadata, name string) string {
	for _, md := range metadata {
		if md.Name == name {
			return md.Value
		}
	}
	return ""
}
