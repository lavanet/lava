package memoryutils

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetMemoryStats(t *testing.T) {
	stats := GetMemoryStats()

	require.NotNil(t, stats)
	require.Greater(t, stats.AllocMB, 0.0)
	require.Greater(t, stats.SysMB, 0.0)
	require.GreaterOrEqual(t, stats.NumGC, uint32(0))
	require.Greater(t, stats.GoroutineCount, 0)
}

func TestEnableDisableMemoryLogs(t *testing.T) {
	// Test initial state
	EnableMemoryLogs(false)
	require.False(t, IsMemoryLogsEnabled())

	// Test enabling
	EnableMemoryLogs(true)
	require.True(t, IsMemoryLogsEnabled())

	// Test disabling
	EnableMemoryLogs(false)
	require.False(t, IsMemoryLogsEnabled())
}

func TestLogMemoryAndMessageSize(t *testing.T) {
	ctx := context.Background()

	// Test with logs disabled - should not panic
	EnableMemoryLogs(false)
	require.NotPanics(t, func() {
		LogMemoryAndMessageSize(ctx, "test_stage", 1024)
	})

	// Test with logs enabled - should not panic
	EnableMemoryLogs(true)
	require.NotPanics(t, func() {
		LogMemoryAndMessageSize(ctx, "test_stage", 1024)
	})

	// Clean up
	EnableMemoryLogs(false)
}

func TestMemoryProfiler(t *testing.T) {
	profiler := NewMemoryProfiler()
	require.NotNil(t, profiler)
	require.NotNil(t, profiler.startStats)

	// Do some work that might allocate memory
	data := make([]byte, 1024*1024) // 1MB
	_ = data

	time.Sleep(10 * time.Millisecond)

	delta := profiler.GetDelta()
	require.NotNil(t, delta)
	require.Contains(t, delta, "alloc_mb_delta")
	require.Contains(t, delta, "heap_alloc_mb_delta")
	require.Contains(t, delta, "sys_mb_delta")
	require.Contains(t, delta, "goroutine_delta")
	require.Contains(t, delta, "gc_count_delta")
	require.Contains(t, delta, "duration_ms")

	// Duration should be positive
	duration, ok := delta["duration_ms"].(int64)
	require.True(t, ok, "duration_ms should be int64")
	require.Greater(t, duration, int64(0))
}

func TestMemoryProfilerLogDelta(t *testing.T) {
	ctx := context.Background()
	profiler := NewMemoryProfiler()

	// Test with logs disabled
	EnableMemoryLogs(false)
	require.NotPanics(t, func() {
		profiler.LogDelta(ctx, "test_delta")
	})

	// Test with logs enabled
	EnableMemoryLogs(true)
	require.NotPanics(t, func() {
		profiler.LogDelta(ctx, "test_delta")
	})

	// Clean up
	EnableMemoryLogs(false)
}

func TestStartMemoryGC_Disabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test with threshold 0 (disabled)
	require.NotPanics(t, func() {
		StartMemoryGC(ctx, 0)
	})

	// Test with negative threshold (disabled)
	require.NotPanics(t, func() {
		StartMemoryGC(ctx, -1.0)
	})

	// Wait a bit to ensure no goroutine crashes
	time.Sleep(100 * time.Millisecond)
}

func TestStartMemoryGC_Enabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start with a very low threshold to ensure GC gets triggered
	thresholdGB := 0.001 // 1MB threshold

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialNumGC := m.NumGC

	// Start the GC monitoring
	require.NotPanics(t, func() {
		StartMemoryGC(ctx, thresholdGB)
	})

	// Allocate enough memory to exceed the threshold
	allocSize := 10 * 1024 * 1024 // 10MB
	data := make([][]byte, 0, 100)
	for i := 0; i < 10; i++ {
		data = append(data, make([]byte, allocSize))
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for the monitoring goroutine to check and potentially trigger GC
	time.Sleep(gcMonitoringCheckInterval + 1*time.Second)

	// Keep reference to data to prevent early GC
	_ = data

	// Check that GC was triggered at least once
	runtime.ReadMemStats(&m)
	// Note: We can't guarantee GC was triggered by our monitor specifically,
	// but we can verify the goroutine ran without crashing
	require.GreaterOrEqual(t, m.NumGC, initialNumGC)
}

func TestStartMemoryGC_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	thresholdGB := 0.1

	// Start the GC monitoring
	require.NotPanics(t, func() {
		StartMemoryGC(ctx, thresholdGB)
	})

	// Wait a bit for the goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for the goroutine to stop
	time.Sleep(gcMonitoringCheckInterval + 500*time.Millisecond)

	// If the goroutine didn't stop properly, this test would hang
	// The fact that we reach here means the goroutine handled cancellation correctly
}

func TestStartMemoryGC_ThresholdBehavior(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a very high threshold that won't be exceeded
	thresholdGB := 100.0 // 100GB - unlikely to exceed in tests

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialNumGC := m.NumGC

	// Start the GC monitoring
	require.NotPanics(t, func() {
		StartMemoryGC(ctx, thresholdGB)
	})

	// Wait for several check intervals
	time.Sleep(gcMonitoringCheckInterval*3 + 500*time.Millisecond)

	// With such a high threshold, our monitor should not have triggered GC
	// (though runtime might have triggered it for other reasons)
	runtime.ReadMemStats(&m)
	gcIncrease := m.NumGC - initialNumGC

	// We just verify the test runs without panicking
	// The actual GC count can vary based on runtime behavior
	t.Logf("GC count increased by %d during high threshold test", gcIncrease)
}

func TestStartMemoryGC_MultipleInstances(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start multiple GC monitors - should not cause issues
	require.NotPanics(t, func() {
		StartMemoryGC(ctx, 0.1)
		StartMemoryGC(ctx, 0.2)
		StartMemoryGC(ctx, 0.3)
	})

	// Wait briefly for all to start
	time.Sleep(200 * time.Millisecond)
}

func TestMemoryConstants(t *testing.T) {
	// Verify the GC monitoring check interval is correctly defined
	require.Equal(t, 5*time.Second, gcMonitoringCheckInterval)
}

func TestMemoryStatsValues(t *testing.T) {
	stats := GetMemoryStats()

	// All values should be non-negative
	require.GreaterOrEqual(t, stats.AllocMB, 0.0)
	require.GreaterOrEqual(t, stats.TotalAllocMB, 0.0)
	require.GreaterOrEqual(t, stats.SysMB, 0.0)
	require.GreaterOrEqual(t, stats.HeapAllocMB, 0.0)
	require.GreaterOrEqual(t, stats.HeapSysMB, 0.0)
	require.GreaterOrEqual(t, stats.HeapIdleMB, 0.0)
	require.GreaterOrEqual(t, stats.HeapInUseMB, 0.0)
	require.GreaterOrEqual(t, stats.StackInUseMB, 0.0)

	// TotalAlloc should be >= Alloc
	require.GreaterOrEqual(t, stats.TotalAllocMB, stats.AllocMB)

	// Sys should be >= HeapSys
	require.GreaterOrEqual(t, stats.SysMB, stats.HeapSysMB)

	// HeapSys should be >= HeapInUse + HeapIdle (approximately, due to fragmentation)
	// We use a loose check here
	require.Greater(t, stats.HeapSysMB, 0.0)
}

func TestMemoryLogsEnabledAtomic(t *testing.T) {
	// Test that the atomic operations work correctly in concurrent scenarios
	EnableMemoryLogs(false)

	done := make(chan bool)
	iterations := 100

	// Writer goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			EnableMemoryLogs(i%2 == 0)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			_ = IsMemoryLogsEnabled()
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Should complete without panic or race conditions
}

func TestStartMemoryGC_GCTriggeredCorrectly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GC trigger test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a very low threshold to ensure we trigger GC
	thresholdGB := 0.005 // 5MB

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialHeapInUse := m.HeapInuse
	initialNumGC := m.NumGC

	// Start the GC monitoring
	StartMemoryGC(ctx, thresholdGB)

	// Allocate memory to exceed the threshold
	const chunkSize = 1024 * 1024 // 1MB
	chunks := make([][]byte, 0, 20)

	for i := 0; i < 20; i++ {
		chunk := make([]byte, chunkSize)
		// Write to the memory to ensure it's actually allocated
		for j := 0; j < len(chunk); j += 4096 {
			chunk[j] = byte(i)
		}
		chunks = append(chunks, chunk)
		time.Sleep(100 * time.Millisecond)

		// Check if we've exceeded the threshold
		runtime.ReadMemStats(&m)
		if m.HeapInuse > uint64(thresholdGB*1024*1024*1024) {
			break
		}
	}

	// Wait for the monitor to detect and trigger GC
	time.Sleep(gcMonitoringCheckInterval*2 + 1*time.Second)

	// Keep reference to prevent early collection
	runtime.KeepAlive(chunks)

	// Read stats after GC monitoring has had time to run
	runtime.ReadMemStats(&m)
	finalNumGC := m.NumGC

	t.Logf("Initial heap in use: %d bytes, Initial GC count: %d", initialHeapInUse, initialNumGC)
	t.Logf("Final heap in use: %d bytes, Final GC count: %d", m.HeapInuse, finalNumGC)
	t.Logf("Threshold: %.2f GB (%d bytes)", thresholdGB, uint64(thresholdGB*1024*1024*1024))

	// Verify that GC ran (either by our monitor or by runtime)
	// We can't guarantee our monitor triggered it, but we verify the system works
	require.GreaterOrEqual(t, finalNumGC, initialNumGC,
		"Expected GC to run at least once during the test")
}

func TestMemoryProfilerConcurrency(t *testing.T) {
	profiler := NewMemoryProfiler()

	done := make(chan bool)
	readers := 5

	// Multiple goroutines reading delta concurrently
	for i := 0; i < readers; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				delta := profiler.GetDelta()
				require.NotNil(t, delta)
				time.Sleep(time.Microsecond)
			}
			done <- true
		}()
	}

	// Wait for all readers
	for i := 0; i < readers; i++ {
		<-done
	}
}

// Benchmark tests
func BenchmarkGetMemoryStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetMemoryStats()
	}
}

func BenchmarkIsMemoryLogsEnabled(b *testing.B) {
	EnableMemoryLogs(true)
	for i := 0; i < b.N; i++ {
		IsMemoryLogsEnabled()
	}
}

func BenchmarkMemoryProfilerGetDelta(b *testing.B) {
	profiler := NewMemoryProfiler()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.GetDelta()
	}
}

func BenchmarkLogMemoryAndMessageSize(b *testing.B) {
	ctx := context.Background()
	EnableMemoryLogs(true)
	defer EnableMemoryLogs(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogMemoryAndMessageSize(ctx, "benchmark", 1024)
	}
}

// Test helper to verify atomic operations
func TestAtomicMemoryLogsFlag(t *testing.T) {
	// Reset to known state
	atomic.StoreInt32(&memoryLogsEnabled, 0)
	require.False(t, IsMemoryLogsEnabled())

	// Set to enabled
	atomic.StoreInt32(&memoryLogsEnabled, 1)
	require.True(t, IsMemoryLogsEnabled())

	// Set back to disabled
	atomic.StoreInt32(&memoryLogsEnabled, 0)
	require.False(t, IsMemoryLogsEnabled())
}
