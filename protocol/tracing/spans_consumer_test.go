package tracing

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartConsumerSendRelay_RecordsRelayAttrs(t *testing.T) {
	sr := setupTestTracer(t)

	ctx, span := StartConsumerSendRelay(context.Background(), 12345, "ETH1", "jsonrpc")
	require.NotNil(t, ctx)
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanConsumerSendRelay, spans[0].Name())

	guid, ok := findStringAttr(spans[0].Attributes(), attrRelayGUID)
	require.True(t, ok)
	require.Equal(t, strconv.FormatUint(12345, 10), guid)

	chainID, ok := findStringAttr(spans[0].Attributes(), attrRelayChainID)
	require.True(t, ok)
	require.Equal(t, "ETH1", chainID)
}

func TestStartConsumerCacheLookup_RecordsResult(t *testing.T) {
	sr := setupTestTracer(t)

	ctx, span := StartConsumerCacheLookup(context.Background())
	RecordCacheLookup(ctx, span, "consumer", true, 3.0)
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanConsumerCacheLookup, spans[0].Name())

	role, ok := findStringAttr(spans[0].Attributes(), attrCacheRole)
	require.True(t, ok)
	require.Equal(t, "consumer", role)
}
