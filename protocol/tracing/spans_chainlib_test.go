package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestStartChainlibHTTP_CreatesServerSpan(t *testing.T) {
	sr := setupTestTracer(t)

	ctx, span := StartChainlibHTTP(context.Background(), "POST", "/eth_blockNumber")
	require.NotNil(t, ctx)
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanChainlibHTTP, spans[0].Name())
	require.Equal(t, trace.SpanKindServer, spans[0].SpanKind())

	method, ok := findStringAttr(spans[0].Attributes(), "http.method")
	require.True(t, ok)
	require.Equal(t, "POST", method)
}
