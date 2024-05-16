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
