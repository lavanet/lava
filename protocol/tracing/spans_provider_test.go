package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestStartProviderHandleRelay_RecordsAttrs(t *testing.T) {
	sr := setupTestTracer(t)

	_, span := StartProviderHandleRelay(context.Background(), 999, "ETH1", "jsonrpc")
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanProviderHandleRelay, spans[0].Name())
}

func TestStartProviderHandleConsistency_RecordsTimingAttrs(t *testing.T) {
	sr := setupTestTracer(t)

	_, span := StartProviderHandleConsistency(context.Background(), 1500)
	RecordHandleConsistencyResult(span, 25, false)
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanProviderHandleConsistency, spans[0].Name())

	target, ok := findInt64Attr(spans[0].Attributes(), attrConsistencyTargetBlock)
	require.True(t, ok)
	require.Equal(t, int64(1500), target)

	bailed, ok := findBoolAttr(spans[0].Attributes(), attrConsistencyBailed)
	require.True(t, ok)
	require.False(t, bailed)
}

func findInt64Attr(attrs []attribute.KeyValue, key string) (int64, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value.AsInt64(), true
		}
	}
	return 0, false
}

func findBoolAttr(attrs []attribute.KeyValue, key string) (bool, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value.AsBool(), true
		}
	}
	return false, false
}
