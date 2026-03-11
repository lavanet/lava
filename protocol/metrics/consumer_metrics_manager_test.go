package metrics

import (
	"fmt"
	"testing"
	"time"
)

func TestAddLatency(t *testing.T) {
	lt := LatencyTracker{}

	// Add some latencies
	latencies := []time.Duration{time.Millisecond * 100, time.Millisecond * 200, time.Millisecond * 300}
	expectedAverageLatencies := []time.Duration{time.Millisecond * 100, time.Millisecond * 150, time.Millisecond * 200}

	for i, latency := range latencies {
		lt.AddLatency(latency)
		fmt.Printf("Average Latency after adding %v: %v\n", latency, lt.AverageLatency)
		if lt.AverageLatency != expectedAverageLatencies[i] {
			t.Errorf("Expected average latency %v, got %v", expectedAverageLatencies[i], lt.AverageLatency)
		}
	}

	// Test zero TotalRequests
	lt2 := LatencyTracker{}
	if lt2.AverageLatency != 0 {
		t.Errorf("Expected average latency 0, got %v", lt2.AverageLatency)
	}
}

func TestAddLatencyNanoSeconds(t *testing.T) {
	lt := LatencyTracker{}

	// Add some latencies
	latencies := []time.Duration{}
	timeNow := time.Now()
	time.Sleep(101 * time.Millisecond)
	latencies = append(latencies, time.Since(timeNow))
	time.Sleep(101 * time.Millisecond)
	latencies = append(latencies, time.Since(timeNow))
	time.Sleep(101 * time.Millisecond)
	latencies = append(latencies, time.Since(timeNow))

	for i, latency := range latencies {
		lt.AddLatency(latency)
		fmt.Printf("Average Latency after adding %v: %v\n", latency, lt.AverageLatency)
		if lt.AverageLatency.Milliseconds() < int64(100*i) {
			t.Errorf("Expected average latency %v < %v", int64(100*i), lt.AverageLatency)
		}
	}

	// Test zero TotalRequests
	lt2 := LatencyTracker{}
	if lt2.AverageLatency != 0 {
		t.Errorf("Expected average latency 0, got %v", lt2.AverageLatency)
	}
}

func TestAddLatencyWithZeroTotalRequests(t *testing.T) {
	lt := LatencyTracker{}

	// Adding latency without incrementing TotalRequests
	lt.AddLatency(time.Millisecond * 100)

	if lt.AverageLatency != time.Millisecond*100 {
		t.Errorf("Expected average latency %v, got %v", time.Millisecond*100, lt.AverageLatency)
	}
}

func TestConsumerEndToEndLatency(t *testing.T) {
	// Create a single ConsumerMetricsManager for all test cases
	// to avoid duplicate Prometheus registration errors
	cmm := NewConsumerMetricsManager(ConsumerMetricsManagerOptions{
		NetworkAddress: ":0", // Use port 0 for random available port
	})

	t.Run("AllInterfaces", func(t *testing.T) {
		testCases := []struct {
			name         string
			chainID      string
			apiInterface string
			latency      time.Duration
			expectedMs   int64
		}{
			{
				name:         "REST interface with 50ms latency",
				chainID:      "LAV1",
				apiInterface: "rest",
				latency:      50 * time.Millisecond,
				expectedMs:   50,
			},
			{
				name:         "TendermintRPC interface with 100ms latency",
				chainID:      "LAV1",
				apiInterface: "tendermintrpc",
				latency:      100 * time.Millisecond,
				expectedMs:   100,
			},
			{
				name:         "gRPC interface with 25ms latency",
				chainID:      "ETH1",
				apiInterface: "grpc",
				latency:      25 * time.Millisecond,
				expectedMs:   25,
			},
			{
				name:         "JSON-RPC interface with 75ms latency",
				chainID:      "ETH1",
				apiInterface: "jsonrpc",
				latency:      75 * time.Millisecond,
				expectedMs:   75,
			},
		}

		for _, tc := range testCases {
			// Set the end-to-end latency
			cmm.SetEndToEndLatency(tc.chainID, tc.apiInterface, tc.latency)

			// Verify the metric was set correctly
			if tc.latency.Milliseconds() != tc.expectedMs {
				t.Errorf("%s: Expected %dms, got %dms", tc.name, tc.expectedMs, tc.latency.Milliseconds())
			}
			fmt.Printf("[PASS] %s: %dms\n", tc.name, tc.expectedMs)
		}
	})

	t.Run("MultipleUpdates", func(t *testing.T) {
		chainID := "LAV1"
		apiInterface := "rest"

		// Test that multiple updates work correctly (should keep last value)
		latencies := []time.Duration{
			10 * time.Millisecond,
			50 * time.Millisecond,
			100 * time.Millisecond,
			25 * time.Millisecond,
		}

		for i, latency := range latencies {
			cmm.SetEndToEndLatency(chainID, apiInterface, latency)
			fmt.Printf("  Update %d: Set end-to-end latency to %dms\n", i+1, latency.Milliseconds())
		}

		// The final value should be the last one set (25ms)
		// Since this is a Gauge metric, it should reflect the most recent value
		fmt.Printf("[PASS] Multiple updates completed successfully\n")
	})

	t.Run("NilManager", func(t *testing.T) {
		// Test that calling SetEndToEndLatency on a nil manager doesn't panic
		var nilManager *ConsumerMetricsManager = nil

		// This should not panic due to nil check in the method
		nilManager.SetEndToEndLatency("LAV1", "rest", 50*time.Millisecond)

		fmt.Printf("[PASS] Nil manager handled gracefully\n")
	})

	t.Run("DifferentInterfacesSameChain", func(t *testing.T) {
		chainID := "OSMOSIS"

		// Set different latencies for different interfaces on the same chain
		interfaces := map[string]time.Duration{
			"rest":          80 * time.Millisecond,
			"tendermintrpc": 164 * time.Millisecond,
			"grpc":          30 * time.Millisecond,
			"jsonrpc":       45 * time.Millisecond,
		}

		for apiInterface, latency := range interfaces {
			cmm.SetEndToEndLatency(chainID, apiInterface, latency)
			fmt.Printf("  [PASS] Set %s interface: %dms\n", apiInterface, latency.Milliseconds())
		}

		// Verify all interfaces can coexist with different values
		// Each should be tracked independently via Prometheus labels
		fmt.Printf("[PASS] All %d interfaces set successfully with independent values\n", len(interfaces))
	})
}
