package rpcsmartrouter

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/stretchr/testify/require"
)

// TestProcessingTimestampLifecycle tests the full lifecycle of ProcessingTimestamp
func TestProcessingTimestampLifecycle(t *testing.T) {
	startTime := time.Now()

	// Create analytics as done in chainlib
	analytics := metrics.NewRelayAnalytics("test-project", "LAV1", "rest")
	analytics.SetProcessingTimestampBeforeRelay(startTime)

	// Verify initial state
	require.Equal(t, startTime, analytics.ProcessingTimestamp, "ProcessingTimestamp should be set")
	require.False(t, analytics.MeasureAfterProviderProcessingTime, "MeasureAfterProviderProcessingTime should be false initially")

	// Simulate successful relay (as done in rpcconsumer_server.go)
	time.Sleep(10 * time.Millisecond)
	afterProviderTime := time.Now()
	analytics.SetProcessingTimestampAfterRelay(afterProviderTime)

	// Verify after provider state
	require.Equal(t, afterProviderTime, analytics.ProcessingTimestamp, "ProcessingTimestamp should be updated")
	require.True(t, analytics.MeasureAfterProviderProcessingTime, "MeasureAfterProviderProcessingTime should be true")
}
