package relaycore

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
	analytics := metrics.NewRelayAnalytics("test-project", "LAVA", "rest")
	analytics.SetProcessingTimestampBeforeRelay(startTime)

	require.Equal(t, startTime, analytics.ProcessingTimestamp, "ProcessingTimestamp should be set after before-relay")

	// Simulate successful relay
	time.Sleep(10 * time.Millisecond)
	afterProviderTime := time.Now()
	analytics.SetProcessingTimestampAfterRelay(afterProviderTime)

	require.Equal(t, afterProviderTime, analytics.ProcessingTimestamp, "ProcessingTimestamp should be updated after relay")
}
