package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNoopUsageSink_IsZeroCost(t *testing.T) {
	var sink UsageEventSink = NoopUsageSink{}
	// Each call must return immediately and mutate no shared state.
	for i := 0; i < 1000; i++ {
		sink.Emit(RelayUsageEvent{
			TimestampNs:  time.Now().UnixNano(),
			ProjectHash:  "p",
			ChainID:      "eth",
			ComputeUnits: 1,
		})
	}
	require.Equal(t, SinkStats{}, sink.Stats())
	sink.Close() // safe, idempotent
}

func TestNoopUsageSink_NilSafetyByValue(t *testing.T) {
	// NoopUsageSink is a value type; the zero value is also valid.
	var sink NoopUsageSink
	require.NotPanics(t, func() {
		sink.Emit(RelayUsageEvent{})
		sink.EmitOptimizerQoS(OptimizerQoSReportToSend{})
		_ = sink.Stats()
		sink.Close()
	})
}

func TestNoopUsageSink_OptimizerQoSIsZeroCost(t *testing.T) {
	var sink UsageEventSink = NoopUsageSink{}
	for i := 0; i < 1000; i++ {
		sink.EmitOptimizerQoS(OptimizerQoSReportToSend{
			Timestamp:       time.Now(),
			ProviderAddress: "lava@p",
			ChainId:         "eth",
		})
	}
	require.Equal(t, SinkStats{}, sink.Stats())
}

func TestNewRelayUsageEvent_FullFidelity(t *testing.T) {
	now := time.Now()
	rm := &RelayMetrics{
		Timestamp:       now,
		ProjectHash:     "tenant-Z",
		ChainID:         "sol",
		APIType:         "rest",
		ApiMethod:       "getBlockHeight",
		ComputeUnits:    7,
		Latency:         42,
		Success:         true,
		CacheHit:        true,
		IsArchive:       true,
		HedgeCount:      2,
		ProviderAddress: "lava@provider1",
		Origin:          "https://app.example",
	}

	event := NewRelayUsageEvent(rm)
	require.Equal(t, now.UnixNano(), event.TimestampNs)
	require.Equal(t, "tenant-Z", event.ProjectHash)
	require.Equal(t, "sol", event.ChainID)
	require.Equal(t, "rest", event.APIInterface)
	require.Equal(t, "getBlockHeight", event.APIMethod)
	require.Equal(t, uint64(7), event.ComputeUnits)
	require.Equal(t, int64(42), event.LatencyMs)
	require.True(t, event.Success)
	require.True(t, event.CacheHit)
	require.True(t, event.IsArchive)
	require.Equal(t, uint64(2), event.HedgeCount)
	require.Equal(t, "lava@provider1", event.ProviderAddress)
	require.Equal(t, "https://app.example", event.Origin)
}
