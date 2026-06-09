package tracing

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// setupTestTracingWithExporter installs a TracerProvider with an in-memory
// span exporter so completed spans can be inspected.
func setupTestTracingWithExporter(t *testing.T) (trace.Tracer, *tracetest.InMemoryExporter) {
	t.Helper()

	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Tracer("test"), exporter
}

func TestRecordError(t *testing.T) {
	tracer, exporter := setupTestTracingWithExporter(t)

	tests := []struct {
		name   string
		err    error
		errMsg string
	}{
		{
			name:   "simple error",
			err:    errors.New("connection refused"),
			errMsg: "connection refused",
		},
		{
			name:   "wrapped error",
			err:    errors.New("rpc failed: context deadline exceeded"),
			errMsg: "rpc failed: context deadline exceeded",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exporter.Reset()

			_, span := tracer.Start(context.Background(), "test-span")
			RecordError(span, tc.err)
			span.End()

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)

			s := spans[0]
			require.Equal(t, otelcodes.Error, s.Status.Code)
			require.Equal(t, tc.errMsg, s.Status.Description)

			// Must have an event with the error recorded.
			require.NotEmpty(t, s.Events)
			foundException := false
			for _, ev := range s.Events {
				if ev.Name == "exception" {
					foundException = true
					break
				}
			}
			require.True(t, foundException, "span must contain an exception event")
		})
	}
}

func TestRecordBody(t *testing.T) {
	shortBody := []byte(`{"jsonrpc":"2.0","id":1}`)

	// RecordBody no longer self-gates on traceBodyEnabled — that check now
	// lives at the call site (see IsTraceBodyEnabled). These cases cover
	// what RecordBody itself is responsible for: handing the body to the
	// SDK and pre-truncating to the configured attribute-value limit.
	tests := []struct {
		name        string
		envLimit    string // OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT ("" = unset)
		body        []byte
		expectValue string
	}{
		{
			name:        "short body",
			body:        shortBody,
			expectValue: string(shortBody),
		},
		{
			name:        "nil body",
			body:        nil,
			expectValue: "",
		},
		{
			// Verifies the SDK's truncation contract on our behalf:
			// (1) the attribute is still PRESENT (SDK truncates, doesn't drop)
			// (2) value is the first `limit` bytes of the original
			// (3) no "...(truncated)" marker is added by the SDK
			name:        "body exceeding env limit, truncated by SDK",
			envLimit:    "100",
			body:        []byte(strings.Repeat("x", 200)),
			expectValue: strings.Repeat("x", 100),
		},
		{
			// With no env var set, the SDK default is -1 (unlimited), so the
			// full body must round-trip onto the span unchanged.
			name:        "no env limit, full body recorded",
			envLimit:    "",
			body:        []byte(strings.Repeat("x", 8192)),
			expectValue: strings.Repeat("x", 8192),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Each subtest gets a clean env so an OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT
			// from one case doesn't leak into the next. Set the limit BEFORE
			// constructing the TracerProvider — sdktrace.NewSpanLimits() reads
			// the env var at provider construction time, not at SetAttributes time.
			clearTracingEnv(t)
			if tc.envLimit != "" {
				t.Setenv("OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT", tc.envLimit)
			}

			tracer, exporter := setupTestTracingWithExporter(t)

			_, span := tracer.Start(context.Background(), "test-span")
			RecordBody(span, AttrRelayRequestBody, tc.body)
			span.End()

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)

			var found bool
			var value string
			for _, attr := range spans[0].Attributes {
				if string(attr.Key) == AttrRelayRequestBody {
					found = true
					value = attr.Value.AsString()
					break
				}
			}

			require.True(t, found, "body attribute must be recorded")
			require.Equal(t, tc.expectValue, value)
		})
	}
}

// recordSpanWith runs fn against a fresh span and returns the recorded span's
// attributes as a map keyed by attribute name. If fn returns false, the
// resulting span is not started so the test can verify "no-op" behavior.
func recordSpanWith(t *testing.T, fn func(span trace.Span)) map[string]attribute.Value {
	t.Helper()
	tracer, exporter := setupTestTracingWithExporter(t)
	_, span := tracer.Start(context.Background(), "test-span")
	fn(span)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	out := make(map[string]attribute.Value, len(spans[0].Attributes))
	for _, attr := range spans[0].Attributes {
		out[string(attr.Key)] = attr.Value
	}
	return out
}

func TestRecordRelayAttributes(t *testing.T) {
	tests := []struct {
		name         string
		guid         uint64
		chainID      string
		apiInterface string
	}{
		{name: "typical relay", guid: 12345, chainID: "ETH1", apiInterface: "jsonrpc"},
		{name: "zero guid", guid: 0, chainID: "LAV1", apiInterface: "tendermintrpc"},
		{name: "max uint64 guid", guid: ^uint64(0), chainID: "COS5", apiInterface: "rest"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := recordSpanWith(t, func(span trace.Span) {
				RecordRelayAttributes(span, tc.guid, tc.chainID, tc.apiInterface)
			})

			require.Equal(t, strconv.FormatUint(tc.guid, 10), attrs[attrRelayGUID].AsString())
			require.Equal(t, tc.chainID, attrs[attrRelayChainID].AsString())
			require.Equal(t, tc.apiInterface, attrs[attrRelayAPIInterface].AsString())
		})
	}
}

func TestRecordRelayMethod(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
	}{
		{name: "eth_blockNumber", methodName: "eth_blockNumber"},
		{name: "empty method", methodName: ""},
		{name: "namespaced method", methodName: "cosmos.bank.v1beta1.Query/TotalSupply"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := recordSpanWith(t, func(span trace.Span) {
				RecordRelayMethod(span, tc.methodName)
			})

			require.Equal(t, tc.methodName, attrs[attrRelayMethod].AsString())
		})
	}
}

func TestRecordProviderAttributes(t *testing.T) {
	tests := []struct {
		name            string
		guid            uint64
		providerAddress string
	}{
		{name: "typical provider", guid: 999, providerAddress: "lava@1abc"},
		{name: "zero guid", guid: 0, providerAddress: "lava@1xyz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := recordSpanWith(t, func(span trace.Span) {
				RecordProviderAttributes(span, tc.guid, tc.providerAddress)
			})

			require.Equal(t, strconv.FormatUint(tc.guid, 10), attrs[attrRelayGUID].AsString())
			require.Equal(t, tc.providerAddress, attrs[attrProviderAddress].AsString())
		})
	}
}

func TestRecordConsistencyStats(t *testing.T) {
	tests := []struct {
		name                    string
		total, passed, rejected int
	}{
		{name: "all passed", total: 5, passed: 5, rejected: 0},
		{name: "all rejected", total: 5, passed: 0, rejected: 5},
		{name: "mixed", total: 10, passed: 7, rejected: 3},
		{name: "zeros", total: 0, passed: 0, rejected: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := recordSpanWith(t, func(span trace.Span) {
				RecordConsistencyStats(span, tc.total, tc.passed, tc.rejected)
			})

			require.Equal(t, int64(tc.total), attrs[attrConsistencyTotal].AsInt64())
			require.Equal(t, int64(tc.passed), attrs[attrConsistencyPassed].AsInt64())
			require.Equal(t, int64(tc.rejected), attrs[attrConsistencyRejected].AsInt64())
		})
	}
}

func TestRecordCacheResult(t *testing.T) {
	tests := []struct {
		name      string
		hit       bool
		latencyMs float64
	}{
		{name: "cache hit", hit: true, latencyMs: 1.5},
		{name: "cache miss", hit: false, latencyMs: 0.3},
		{name: "zero latency", hit: true, latencyMs: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := recordSpanWith(t, func(span trace.Span) {
				RecordCacheResult(context.Background(), span, tc.hit, tc.latencyMs)
			})

			require.Equal(t, tc.hit, attrs[attrCacheHit].AsBool())
			require.Equal(t, tc.latencyMs, attrs[attrCacheLatencyMs].AsFloat64())
		})
	}
}

// TestRecordCacheResult_BubblesToRelaySpan verifies that cache.hit is mirrored
// onto the inbound relay span attached via WithRelaySpan, so TraceQL filters
// on the top-level trace can find cached responses.
func TestRecordCacheResult_BubblesToRelaySpan(t *testing.T) {
	tracer, exporter := setupTestTracingWithExporter(t)

	ctx, relaySpan := tracer.Start(context.Background(), "consumer.SendRelay")
	ctx = WithRelaySpan(ctx, relaySpan)
	_, cacheSpan := tracer.Start(ctx, "consumer.CacheLookup")

	RecordCacheResult(ctx, cacheSpan, true, 1.2)

	cacheSpan.End()
	relaySpan.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 2)

	// Verify cache.hit landed on BOTH spans (the cache lookup child and the
	// relay parent).
	for _, s := range spans {
		var hit bool
		var found bool
		for _, attr := range s.Attributes {
			if string(attr.Key) == attrCacheHit {
				hit = attr.Value.AsBool()
				found = true
				break
			}
		}
		require.True(t, found, "cache.hit must be present on %s", s.Name)
		require.True(t, hit, "cache.hit must be true on %s", s.Name)
	}
}

func TestRecordSessionStats(t *testing.T) {
	tests := []struct {
		name                string
		requested, acquired int
	}{
		{name: "all acquired", requested: 3, acquired: 3},
		{name: "partial", requested: 5, acquired: 2},
		{name: "none acquired", requested: 4, acquired: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := recordSpanWith(t, func(span trace.Span) {
				RecordSessionStats(span, tc.requested, tc.acquired)
			})

			require.Equal(t, int64(tc.requested), attrs[attrSessionRequested].AsInt64())
			require.Equal(t, int64(tc.acquired), attrs[attrSessionAcquired].AsInt64())
		})
	}
}

func TestStartClientSpan_OrphanGuard(t *testing.T) {
	tracer, exporter := setupTestTracingWithExporter(t)
	// Make `tracer()` (used inside StartClientSpan) resolve to our test
	// provider. setupTestTracingWithExporter already installs it as the
	// global TracerProvider.
	_ = tracer

	t.Run("no parent → returns non-recording span, no orphan span exported", func(t *testing.T) {
		exporter.Reset()
		ctx, span := StartClientSpan(context.Background(), "consumer.relayInner")
		require.False(t, span.IsRecording(), "no parent → must not start a recording span")
		span.End()
		_ = ctx
		require.Empty(t, exporter.GetSpans(), "no orphan root span may be exported")
	})

	t.Run("recording parent → starts a real client span", func(t *testing.T) {
		exporter.Reset()
		parentCtx, parent := tracer.Start(context.Background(), "parent")
		ctx, span := StartClientSpan(parentCtx, "consumer.relayInner")
		require.True(t, span.IsRecording())
		span.End()
		parent.End()
		_ = ctx

		spans := exporter.GetSpans()
		require.Len(t, spans, 2)
		// SDK exports children before parents.
		require.Equal(t, "consumer.relayInner", spans[0].Name)
		require.Equal(t, trace.SpanKindClient, spans[0].SpanKind)
	})
}

func TestStartServerSpan_AlwaysCreatesServerSpan(t *testing.T) {
	tracer, exporter := setupTestTracingWithExporter(t)
	_ = tracer
	exporter.Reset()

	_, span := StartServerSpan(context.Background(), "consumer.SendRelay")
	require.True(t, span.IsRecording())
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, trace.SpanKindServer, spans[0].SpanKind)
	require.Equal(t, "consumer.SendRelay", spans[0].Name)
}
