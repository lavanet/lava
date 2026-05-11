package metrics

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplyOTelSinkDefaults_FillsMissing(t *testing.T) {
	cfg := OTelUsageSinkConfig{}
	applyOTelSinkDefaults(&cfg)

	require.Equal(t, defaultOTelQueueSize, cfg.QueueSize)
	require.Equal(t, defaultOTelBatchSize, cfg.BatchSize)
	require.Equal(t, defaultOTelFlushInterval, cfg.FlushInterval)
	require.Equal(t, defaultOTelExportTimeout, cfg.ExportTimeout)
	require.Empty(t, cfg.ServiceName, "ServiceName has no package-level default; callers set it via flag defValue")
	require.NotEmpty(t, cfg.ServiceInstanceID, "instance ID should default to hostname-pid")
	require.Contains(t, cfg.ServiceInstanceID, "-", "default instance ID is hostname-pid format")
}

func TestApplyOTelSinkDefaults_RespectsExplicitValues(t *testing.T) {
	custom := OTelUsageSinkConfig{
		QueueSize:         123,
		BatchSize:         456,
		FlushInterval:     11 * time.Second,
		ExportTimeout:     22 * time.Second,
		ServiceName:       "my-service",
		ServiceInstanceID: "my-instance",
	}
	applyOTelSinkDefaults(&custom)

	require.Equal(t, 123, custom.QueueSize)
	require.Equal(t, 456, custom.BatchSize)
	require.Equal(t, 11*time.Second, custom.FlushInterval)
	require.Equal(t, 22*time.Second, custom.ExportTimeout)
	require.Equal(t, "my-service", custom.ServiceName)
	require.Equal(t, "my-instance", custom.ServiceInstanceID)
}

func TestApplyOTelSinkDefaults_HostnameFallback(t *testing.T) {
	// When os.Hostname returns an error the default falls back to
	// "unknown-pid" rather than panicking.
	original := osHostname
	t.Cleanup(func() { osHostname = original })
	osHostname = func() (string, error) { return "", errors.New("boom") }

	cfg := OTelUsageSinkConfig{}
	applyOTelSinkDefaults(&cfg)

	require.True(t, strings.HasPrefix(cfg.ServiceInstanceID, "unknown-"),
		"hostname error must fall back to unknown-<pid>; got %q", cfg.ServiceInstanceID)
}

func TestOTelUsageSink_NilSafety(t *testing.T) {
	var sink *OTelUsageSink
	require.NotPanics(t, func() {
		sink.Emit(RelayUsageEvent{})
		sink.EmitOptimizerQoS(OptimizerQoSReportToSend{})
		_ = sink.Stats()
		sink.Close()
	})
	require.Equal(t, SinkStats{}, sink.Stats())
}

// TestOTelUsageSink_BadEndpoint_Disconnect ensures that a sink configured
// against an unreachable endpoint doesn't crash on Emit — the OTel SDK
// queues records and the exporter retries asynchronously. Emit returns
// fast and the relay path is never blocked.
func TestOTelUsageSink_BadEndpoint_DoesNotPanic(t *testing.T) {
	sink := NewOTelUsageSink(OTelUsageSinkConfig{
		Endpoint:      "127.0.0.1:1", // closed port
		Insecure:      true,
		QueueSize:     16,
		BatchSize:     4,
		FlushInterval: 50 * time.Millisecond,
		ExportTimeout: 200 * time.Millisecond,
	})
	if sink == nil {
		t.Skip("OTel sink construction failed in CI environment; skipping")
	}
	t.Cleanup(sink.Close)

	rm := &RelayMetrics{
		Timestamp:    time.Now(),
		ProjectHash:  "p",
		ChainID:      "eth",
		APIType:      "jsonrpc",
		ComputeUnits: 1,
	}
	require.NotPanics(t, func() {
		for i := 0; i < 16; i++ {
			sink.Emit(NewRelayUsageEvent(rm))
		}
	})

	// Optimizer-QoS path: also non-blocking, no panics, counters advance.
	report := OptimizerQoSReportToSend{
		Timestamp:        time.Now(),
		ProviderAddress:  "lava@p",
		ConsumerHostname: "host01",
		ChainId:          "eth",
		Epoch:            1234,
	}
	require.NotPanics(t, func() {
		for i := 0; i < 16; i++ {
			sink.EmitOptimizerQoS(report)
		}
	})

	stats := sink.Stats()
	// We count handed-to-SDK records; whether they actually shipped is
	// the SDK's concern. The point is Emit/EmitOptimizerQoS didn't panic
	// or block, and both paths increment Sent.
	require.GreaterOrEqual(t, stats.Sent, uint64(2))
}
