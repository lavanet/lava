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
	expectedAverageLatencies := []time.Duration{100, 150, 200}

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

func TestAddLatencyWithZeroTotalRequests(t *testing.T) {
	lt := LatencyTracker{}

	// Adding latency without incrementing TotalRequests
	lt.AddLatency(time.Millisecond * 100)

	if lt.AverageLatency != 100 {
		t.Errorf("Expected average latency %v, got %v", 100, lt.AverageLatency)
	}
}
