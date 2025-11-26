package rpcprovider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResourceLimiter_Disabled(t *testing.T) {
	// When disabled, should pass through all requests
	rl := newResourceLimiterForTesting(false, 8, 100)
	require.NotNil(t, rl)
	require.False(t, rl.enabled)

	executed := false
	err := rl.Acquire(context.Background(), 500, "debug_trace", func() error {
		executed = true
		return nil
	})

	require.NoError(t, err)
	require.True(t, executed, "Request should execute even when limiter is disabled")
}

func TestResourceLimiter_SelectBucket_CUPriority(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)
	require.NotNil(t, rl)

	tests := []struct {
		name           string
		computeUnits   uint64
		methodName     string
		expectedBucket BucketType
		reason         string
	}{
		{
			name:           "High CU triggers heavy bucket",
			computeUnits:   150,
			methodName:     "eth_call",
			expectedBucket: BucketHeavy,
			reason:         "CU >= threshold",
		},
		{
			name:           "Low CU with debug prefix triggers heavy bucket",
			computeUnits:   50,
			methodName:     "debug_traceTransaction",
			expectedBucket: BucketHeavy,
			reason:         "debug_ prefix",
		},
		{
			name:           "Low CU with trace prefix triggers heavy bucket",
			computeUnits:   50,
			methodName:     "trace_block",
			expectedBucket: BucketHeavy,
			reason:         "trace_ prefix",
		},
		{
			name:           "Normal method with low CU",
			computeUnits:   20,
			methodName:     "eth_blockNumber",
			expectedBucket: BucketNormal,
			reason:         "Low CU, no special prefix",
		},
		{
			name:           "Exactly at threshold goes to heavy",
			computeUnits:   100,
			methodName:     "some_method",
			expectedBucket: BucketHeavy,
			reason:         "CU == threshold",
		},
		{
			name:           "One below threshold with no prefix goes to normal",
			computeUnits:   99,
			methodName:     "some_method",
			expectedBucket: BucketNormal,
			reason:         "CU < threshold, no prefix",
		},
		{
			name:           "Case insensitive debug check",
			computeUnits:   10,
			methodName:     "DEBUG_TraceCall",
			expectedBucket: BucketHeavy,
			reason:         "Case insensitive debug_ prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket := rl.selectBucket(tt.computeUnits, tt.methodName)
			require.Equal(t, tt.expectedBucket, bucket,
				"Expected %s bucket because: %s", tt.expectedBucket, tt.reason)
		})
	}
}

func TestResourceLimiter_HeavyConcurrencyLimit(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)
	require.NotNil(t, rl)

	ctx := context.Background()

	// Block to control when first requests complete
	block := make(chan struct{})
	var executing atomic.Int32
	var maxConcurrent atomic.Int32

	// Helper to track concurrent executions
	executeFunc := func() error {
		current := executing.Add(1)

		// Track max concurrent
		for {
			max := maxConcurrent.Load()
			if current <= max {
				break
			}
			if maxConcurrent.CompareAndSwap(max, current) {
				break
			}
		}

		<-block // Wait for unblock
		executing.Add(-1)
		return nil
	}

	var wg sync.WaitGroup
	results := make([]error, 10)

	// Launch 10 concurrent heavy requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = rl.Acquire(ctx, 200, "debug_trace", executeFunc)
		}()
	}

	// Give goroutines time to start
	time.Sleep(100 * time.Millisecond)

	// Check that only 2 are executing (heavy max concurrent = 2)
	require.Equal(t, int32(2), executing.Load(),
		"Should have exactly 2 heavy requests executing")

	// Unblock all
	close(block)
	wg.Wait()

	// Count successful and rejected
	var successful, rejected int
	for _, err := range results {
		if err == nil {
			successful++
		} else {
			rejected++
		}
	}

	// With max 2 concurrent and queue size 5:
	// Expected: 2 execute immediately, 5 queued, 3 rejected
	// But queue processing is fast, so we might get 7-8 successful
	require.GreaterOrEqual(t, successful, 7, "Should have at least 7 successful")
	require.LessOrEqual(t, successful, 8, "Should have at most 8 successful")
	require.GreaterOrEqual(t, rejected, 2, "Should have at least 2 rejected")
	require.LessOrEqual(t, rejected, 3, "Should have at most 3 rejected")
	require.Equal(t, 10, successful+rejected, "All 10 requests should be accounted for")
	require.LessOrEqual(t, maxConcurrent.Load(), int32(2),
		"Should never exceed 2 concurrent heavy requests")
}

func TestResourceLimiter_NormalConcurrencyNoQueue(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)
	require.NotNil(t, rl)

	ctx := context.Background()

	block := make(chan struct{})
	var executing atomic.Int32

	executeFunc := func() error {
		executing.Add(1)
		<-block
		executing.Add(-1)
		return nil
	}

	var wg sync.WaitGroup
	results := make([]error, 105) // More than max concurrent (100)

	// Launch 105 concurrent normal requests
	for i := 0; i < 105; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = rl.Acquire(ctx, 20, "eth_blockNumber", executeFunc)
		}()
	}

	// Give goroutines time to start
	time.Sleep(100 * time.Millisecond)

	// Should have 100 executing
	require.Equal(t, int32(100), executing.Load(),
		"Should have exactly 100 normal requests executing")

	// Unblock all
	close(block)
	wg.Wait()

	// Count results
	var successful, rejected int
	for _, err := range results {
		if err == nil {
			successful++
		} else {
			rejected++
		}
	}

	// Normal bucket has no queue, so excess is rejected immediately
	require.Equal(t, 100, successful, "Should have 100 successful")
	require.Equal(t, 5, rejected, "Should have 5 rejected (no queue)")
}

func TestResourceLimiter_QueueTimeout(t *testing.T) {
	// Create limiter with very short timeout for testing
	rl := newResourceLimiterForTesting(true, 8, 100)
	// Modify the timeout to be very short
	rl.config[BucketHeavy].Timeout = 100 * time.Millisecond

	ctx := context.Background()

	// Block indefinitely to cause timeout
	block := make(chan struct{})

	executeFunc := func() error {
		<-block
		return nil
	}

	var wg sync.WaitGroup

	// Fill both slots
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Acquire(ctx, 200, "debug_trace", executeFunc)
		}()
	}

	time.Sleep(50 * time.Millisecond) // Ensure they're executing

	// This one should be queued and timeout
	err := rl.Acquire(ctx, 200, "debug_trace", executeFunc)

	close(block) // Cleanup
	wg.Wait()

	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout in queue",
		"Should timeout while waiting in queue")
}

func TestResourceLimiter_ContextCancellation(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)

	ctx, cancel := context.WithCancel(context.Background())

	// Block to control execution
	block := make(chan struct{})

	executeFunc := func() error {
		select {
		case <-block:
			return nil
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}

	var wg sync.WaitGroup

	// Fill both heavy slots
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Acquire(context.Background(), 200, "debug_trace", executeFunc)
		}()
	}

	time.Sleep(50 * time.Millisecond)

	// This should be queued
	errChan := make(chan error, 1)
	go func() {
		errChan <- rl.Acquire(ctx, 200, "debug_trace", executeFunc)
	}()

	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Get error from cancelled request
	err := <-errChan
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)

	// Unblock running requests
	close(block)
	wg.Wait()
}

func TestResourceLimiter_MemoryLimit(t *testing.T) {
	// Create limiter with very low memory threshold
	// Heavy request reserves 512MB, so with 1GB threshold:
	// First request: 512MB (OK)
	// Second request would be: 512MB + 512MB = 1024MB = 1GB
	// We need it to exceed, so set threshold to 0.8 GB (just under 1GB)
	rl := newResourceLimiterForTesting(true, 1, 100) // 1GB threshold

	// Manually set a lower threshold for testing
	rl.memoryThreshold = 800 * 1024 * 1024 // 800MB threshold

	ctx := context.Background()

	// Try to acquire with heavy method (512MB per call)
	// First request should work (512MB < 800MB)
	err1 := rl.Acquire(ctx, 200, "debug_trace", func() error {
		// Second request should be rejected due to memory
		// (512MB + 512MB = 1024MB > 800MB threshold)
		err2 := rl.Acquire(ctx, 200, "debug_trace", func() error {
			return nil
		})
		require.Error(t, err2)
		require.Contains(t, err2.Error(), "memory limit",
			"Should reject due to memory limit")
		return nil
	})

	require.NoError(t, err1, "First request should succeed")
}

func TestResourceLimiter_ExecutionError(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)

	ctx := context.Background()
	expectedErr := errors.New("execution failed")

	err := rl.Acquire(ctx, 20, "eth_call", func() error {
		return expectedErr
	})

	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

func TestResourceLimiter_Metrics(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)
	require.NotNil(t, rl.metrics)

	ctx := context.Background()

	// Execute a successful request
	err := rl.Acquire(ctx, 20, "eth_call", func() error {
		return nil
	})
	require.NoError(t, err)

	// Fill up heavy bucket to cause rejections
	block := make(chan struct{})
	var wg sync.WaitGroup

	// Fill 2 slots and queue (5)
	for i := 0; i < 7; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Acquire(ctx, 200, "debug_trace", func() error {
				<-block
				return nil
			})
		}()
	}

	time.Sleep(100 * time.Millisecond)

	// These should be rejected
	err1 := rl.Acquire(ctx, 200, "debug_trace", func() error { return nil })
	err2 := rl.Acquire(ctx, 200, "debug_trace", func() error { return nil })

	require.Error(t, err1)
	require.Error(t, err2)

	close(block)
	wg.Wait()

	// Check metrics
	require.Greater(t, rl.metrics.TotalRejected, uint64(0),
		"Should have recorded rejections")
	require.Greater(t, rl.metrics.TotalQueued, uint64(0),
		"Should have recorded queued requests")
}

func TestResourceLimiter_ConcurrentMixedRequests(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)

	ctx := context.Background()
	block := make(chan struct{})

	var heavyExecuting, normalExecuting atomic.Int32
	var heavyCompleted, normalCompleted atomic.Int32

	executeHeavy := func() error {
		heavyExecuting.Add(1)
		<-block
		heavyExecuting.Add(-1)
		heavyCompleted.Add(1)
		return nil
	}

	executeNormal := func() error {
		normalExecuting.Add(1)
		<-block
		normalExecuting.Add(-1)
		normalCompleted.Add(1)
		return nil
	}

	var wg sync.WaitGroup

	// Launch 10 heavy requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Acquire(ctx, 200, "debug_trace", executeHeavy)
		}()
	}

	// Launch 50 normal requests
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Acquire(ctx, 20, "eth_call", executeNormal)
		}()
	}

	time.Sleep(200 * time.Millisecond)

	// Check concurrent execution limits
	require.LessOrEqual(t, heavyExecuting.Load(), int32(2),
		"Should not exceed 2 concurrent heavy requests")
	require.LessOrEqual(t, normalExecuting.Load(), int32(100),
		"Should not exceed 100 concurrent normal requests")

	// Normal requests should not be affected by heavy queue
	require.Greater(t, normalExecuting.Load(), int32(0),
		"Normal requests should be executing despite heavy queue")

	close(block)
	wg.Wait()

	// At least some requests should have completed
	require.Greater(t, heavyCompleted.Load(), int32(0))
	require.Greater(t, normalCompleted.Load(), int32(0))
}

func TestResourceLimiter_MemoryReservationRelease(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)

	ctx := context.Background()

	// Check initial memory
	initialMemory := rl.currentMemory
	require.Equal(t, uint64(0), initialMemory)

	// Execute request and check memory during execution
	var memoryDuringExecution uint64
	err := rl.Acquire(ctx, 200, "debug_trace", func() error {
		memoryDuringExecution = rl.currentMemory
		return nil
	})

	require.NoError(t, err)
	require.Greater(t, memoryDuringExecution, uint64(0),
		"Memory should be reserved during execution")

	// Check memory after execution
	finalMemory := rl.currentMemory
	require.Equal(t, uint64(0), finalMemory,
		"Memory should be released after execution")
}

func TestResourceLimiter_BucketTypeString(t *testing.T) {
	require.Equal(t, "heavy", BucketHeavy.String())
	require.Equal(t, "normal", BucketNormal.String())
}

func TestResourceLimiter_ErrorMessages(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)

	ctx := context.Background()

	// Test 1: Queue full error
	// Launch many heavy requests simultaneously to fill queue
	block1 := make(chan struct{})
	var wg1 sync.WaitGroup

	// Launch 10 heavy requests (more than 2 + 5 = 7 capacity)
	errors1 := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg1.Add(1)
		go func() {
			defer wg1.Done()
			errors1[i] = rl.Acquire(ctx, 200, "debug_trace", func() error {
				<-block1
				return nil
			})
		}()
	}

	time.Sleep(200 * time.Millisecond) // Ensure all have attempted

	// At least one should be rejected with "queue full"
	var queueFullFound bool
	for _, err := range errors1 {
		if err != nil && (strings.Contains(err.Error(), "queue full") ||
			strings.Contains(err.Error(), "timeout in queue")) {
			queueFullFound = true
			require.Contains(t, err.Error(), "heavy")
			break
		}
	}
	require.True(t, queueFullFound, "Should have at least one queue full or timeout error")

	close(block1)
	wg1.Wait()

	// Test 2: Max concurrent error (normal has no queue)
	block2 := make(chan struct{})
	var wg2 sync.WaitGroup

	// Launch 105 normal requests (more than 100 max)
	errors2 := make([]error, 105)
	for i := 0; i < 105; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			errors2[i] = rl.Acquire(ctx, 20, "eth_call", func() error {
				<-block2
				return nil
			})
		}()
	}

	time.Sleep(200 * time.Millisecond) // Ensure all have attempted

	// At least one should be rejected with "max concurrent"
	var maxConcurrentFound bool
	for _, err := range errors2 {
		if err != nil && strings.Contains(err.Error(), "max concurrent") {
			maxConcurrentFound = true
			require.Contains(t, err.Error(), "normal")
			break
		}
	}
	require.True(t, maxConcurrentFound, "Should have at least one max concurrent error")

	close(block2)
	wg2.Wait()
}

func BenchmarkResourceLimiter_LightLoad(b *testing.B) {
	rl := newResourceLimiterForTesting(true, 8, 100)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rl.Acquire(ctx, 20, "eth_call", func() error {
				return nil
			})
		}
	})
}

func BenchmarkResourceLimiter_Disabled(b *testing.B) {
	rl := newResourceLimiterForTesting(false, 8, 100)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rl.Acquire(ctx, 20, "eth_call", func() error {
				return nil
			})
		}
	})
}

func BenchmarkResourceLimiter_BucketSelection(b *testing.B) {
	rl := newResourceLimiterForTesting(true, 8, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.selectBucket(150, "debug_traceTransaction")
	}
}

// Test helper to verify metrics are tracking correctly
func TestResourceLimiter_MetricsTracking(t *testing.T) {
	rl := newResourceLimiterForTesting(true, 8, 100)
	ctx := context.Background()

	// Track initial metrics
	initialRejected := rl.metrics.TotalRejected
	initialQueued := rl.metrics.TotalQueued

	// Launch many concurrent requests to trigger rejections
	block := make(chan struct{})
	var wg sync.WaitGroup

	// Launch 15 heavy requests (more than capacity of 7)
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Acquire(ctx, 200, "debug_trace", func() error {
				<-block
				return nil
			})
		}()
	}

	time.Sleep(200 * time.Millisecond)

	close(block)
	wg.Wait()

	// Verify metrics increased
	require.Greater(t, rl.metrics.TotalRejected, initialRejected,
		"Metrics should track rejections")
	require.Greater(t, rl.metrics.TotalQueued, initialQueued,
		"Metrics should track queued requests")

	// Should have rejected at least some requests (not all 15 can fit)
	rejectionsTracked := rl.metrics.TotalRejected - initialRejected
	require.GreaterOrEqual(t, rejectionsTracked, uint64(3),
		"Should have rejected at least 3 requests (more than capacity of 7)")
}

func TestResourceLimiter_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	rl := newResourceLimiterForTesting(true, 8, 100)
	ctx := context.Background()

	var totalRequests atomic.Int32
	var successfulRequests atomic.Int32
	var failedRequests atomic.Int32

	executeFunc := func() error {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	}

	// Launch 100 requests of mixed types
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			totalRequests.Add(1)

			var cu uint64
			var method string

			if i%3 == 0 {
				// Heavy request
				cu = 200
				method = "debug_trace"
			} else {
				// Normal request
				cu = 20
				method = "eth_call"
			}

			err := rl.Acquire(ctx, cu, method, executeFunc)
			if err != nil {
				failedRequests.Add(1)
			} else {
				successfulRequests.Add(1)
			}
		}()
	}

	wg.Wait()

	// Verify all requests were processed
	total := totalRequests.Load()
	successful := successfulRequests.Load()
	failed := failedRequests.Load()

	require.Equal(t, int32(100), total, "Should have launched 100 requests")
	require.Equal(t, total, successful+failed,
		"All requests should be either successful or failed")
	require.Greater(t, successful, int32(0),
		"Some requests should succeed")

	fmt.Printf("Stress test results: %d total, %d successful, %d failed\n",
		total, successful, failed)
}
