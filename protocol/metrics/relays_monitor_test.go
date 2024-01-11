package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHappyFlow(t *testing.T) {
	isHealthy := true
	relaySender := func() (bool, error) {
		return isHealthy, nil
	}

	interval := time.Second * 3
	extraTimeToWait := time.Second
	timeToSleep := interval + extraTimeToWait

	relaysMonitor := NewRelaysMonitor(interval, "test_chain", "rest")
	relaysMonitor.SetRelaySender(relaySender)
	relaysMonitor.Start(context.Background())

	t.Run("HappyFlow", func(t *testing.T) {
		// Log a relay
		relaysMonitor.LogRelay()

		// Check if the relays monitor is healthy
		require.True(t, relaysMonitor.IsHealthy())

		// Sleep for the interval
		time.Sleep(timeToSleep)

		// Check if the relays monitor is still healthy
		require.True(t, relaysMonitor.IsHealthy())

		// Set to false
		isHealthy = false

		// Sleep for the interval
		time.Sleep(timeToSleep)

		// Check if the relays monitor is still healthy
		require.False(t, relaysMonitor.IsHealthy())

		// Sleep for the interval
		time.Sleep(timeToSleep)

		// Log a relay
		relaysMonitor.LogRelay()

		// Now should be healthy again
		require.True(t, relaysMonitor.IsHealthy())

		// Sleep for the interval
		time.Sleep(timeToSleep)

		// Now should be not healthy
		require.False(t, relaysMonitor.IsHealthy())
	})
}
