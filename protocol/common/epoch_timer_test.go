package common

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEpochTimer_CalculateCurrentEpoch(t *testing.T) {
	// Test epoch calculation from absolute time
	epochDuration := 1 * time.Hour
	timer := NewEpochTimer(epochDuration)

	// Get current epoch
	currentEpoch := timer.CalculateCurrentEpoch()
	require.NotNil(t, currentEpoch)

	// Epoch should be deterministic based on time
	epoch1 := timer.CalculateCurrentEpoch()
	time.Sleep(10 * time.Millisecond)
	epoch2 := timer.CalculateCurrentEpoch()

	// Within the same hour, epochs should be the same
	require.Equal(t, epoch1, epoch2, "Epochs within same period should be equal")
}

func TestEpochTimer_GetTimeUntilNextEpoch(t *testing.T) {
	// Test time until next epoch
	epochDuration := 30 * time.Minute
	timer := NewEpochTimer(epochDuration)

	timeUntilNext := timer.GetTimeUntilNextEpoch()
	require.Greater(t, timeUntilNext, time.Duration(0), "Time until next epoch should be positive")
	require.LessOrEqual(t, timeUntilNext, epochDuration, "Time until next should not exceed epoch duration")
}

func TestEpochTimer_GetEpochBoundaryTime(t *testing.T) {
	// Test epoch boundary time calculation
	epochDuration := 1 * time.Hour
	timer := NewEpochTimer(epochDuration)

	currentEpoch := timer.CalculateCurrentEpoch()
	boundaryTime := timer.GetEpochBoundaryTime(currentEpoch)

	// Boundary should be in the past (current epoch already started)
	require.True(t, boundaryTime.Before(time.Now()) || boundaryTime.Equal(time.Now()),
		"Current epoch boundary should be in the past")

	// Next epoch boundary should be in the future
	nextBoundaryTime := timer.GetEpochBoundaryTime(currentEpoch + 1)
	require.True(t, nextBoundaryTime.After(time.Now()),
		"Next epoch boundary should be in the future")

	// Time difference between consecutive epochs should equal epoch duration
	timeDiff := nextBoundaryTime.Sub(boundaryTime)
	require.Equal(t, epochDuration, timeDiff,
		"Time between consecutive epochs should equal epoch duration")
}

func TestEpochTimer_Callbacks(t *testing.T) {
	// Test callback registration and triggering
	epochDuration := 100 * time.Millisecond
	timer := NewEpochTimer(epochDuration)

	var callbackCalled atomic.Bool
	var receivedEpoch atomic.Uint64

	// Register callback
	timer.RegisterCallback(func(epoch uint64) {
		callbackCalled.Store(true)
		receivedEpoch.Store(epoch)
	})

	// Start timer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timer.Start(ctx)

	// Wait for callback to be called (initial callback + potential boundary crossing)
	time.Sleep(200 * time.Millisecond)

	// Verify callback was called
	require.True(t, callbackCalled.Load(), "Callback should have been called")
	require.Greater(t, receivedEpoch.Load(), uint64(0), "Received epoch should be greater than 0")

	// Stop timer
	timer.Stop()
}

func TestEpochTimer_MultipleCallbacks(t *testing.T) {
	// Test multiple callback registration
	epochDuration := 100 * time.Millisecond
	timer := NewEpochTimer(epochDuration)

	var callback1Called atomic.Bool
	var callback2Called atomic.Bool

	// Register multiple callbacks
	timer.RegisterCallback(func(epoch uint64) {
		callback1Called.Store(true)
	})
	timer.RegisterCallback(func(epoch uint64) {
		callback2Called.Store(true)
	})

	// Start timer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timer.Start(ctx)

	// Wait for callbacks
	time.Sleep(200 * time.Millisecond)

	// Verify both callbacks were called
	require.True(t, callback1Called.Load(), "Callback 1 should have been called")
	require.True(t, callback2Called.Load(), "Callback 2 should have been called")

	// Stop timer
	timer.Stop()
}

func TestEpochTimer_SynchronizationAcrossInstances(t *testing.T) {
	// Test that two timer instances calculate the same epoch
	epochDuration := 1 * time.Hour
	timer1 := NewEpochTimer(epochDuration)
	timer2 := NewEpochTimer(epochDuration)

	epoch1 := timer1.CalculateCurrentEpoch()
	epoch2 := timer2.CalculateCurrentEpoch()

	require.Equal(t, epoch1, epoch2,
		"Two timers with same duration should calculate same epoch")
}

func TestEpochTimer_ContextCancellation(t *testing.T) {
	// Test that timer respects context cancellation
	epochDuration := 100 * time.Millisecond
	timer := NewEpochTimer(epochDuration)

	var callbackCallCount atomic.Int32

	timer.RegisterCallback(func(epoch uint64) {
		callbackCallCount.Add(1)
	})

	// Start timer with short context
	ctx, cancel := context.WithCancel(context.Background())
	timer.Start(ctx)

	// Wait for initial callback
	time.Sleep(150 * time.Millisecond)
	initialCount := callbackCallCount.Load()
	require.Greater(t, initialCount, int32(0), "Should have at least one callback")

	// Cancel context
	cancel()
	time.Sleep(150 * time.Millisecond)

	// Callback count should not increase after cancellation
	finalCount := callbackCallCount.Load()
	// Allow for one more callback that might have been scheduled
	require.LessOrEqual(t, finalCount-initialCount, int32(1),
		"Should not have many callbacks after context cancellation")
}

func TestEpochTimer_Stop(t *testing.T) {
	// Test Stop() method
	epochDuration := 100 * time.Millisecond
	timer := NewEpochTimer(epochDuration)

	var callbackCallCount atomic.Int32

	timer.RegisterCallback(func(epoch uint64) {
		callbackCallCount.Add(1)
	})

	// Start timer
	ctx := context.Background()
	timer.Start(ctx)

	// Wait for initial callback
	time.Sleep(150 * time.Millisecond)
	initialCount := callbackCallCount.Load()
	require.Greater(t, initialCount, int32(0), "Should have at least one callback")

	// Stop timer
	timer.Stop()
	time.Sleep(150 * time.Millisecond)

	// Callback count should not increase after stop
	finalCount := callbackCallCount.Load()
	require.LessOrEqual(t, finalCount-initialCount, int32(1),
		"Should not have many callbacks after stop")
}

func TestEpochTimer_DifferentDurations(t *testing.T) {
	// Test timers with different durations
	testCases := []struct {
		name     string
		duration time.Duration
	}{
		{"10 minutes", 10 * time.Minute},
		{"30 minutes", 30 * time.Minute},
		{"1 hour", 1 * time.Hour},
		{"2 hours", 2 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			timer := NewEpochTimer(tc.duration)
			epoch := timer.CalculateCurrentEpoch()
			timeUntilNext := timer.GetTimeUntilNextEpoch()

			require.NotNil(t, epoch)
			require.Greater(t, timeUntilNext, time.Duration(0))
			require.LessOrEqual(t, timeUntilNext, tc.duration)
		})
	}
}

func TestEpochTimer_EpochProgression(t *testing.T) {
	// Test that epochs progress correctly
	// Note: This test verifies callbacks are triggered, but epoch changes
	// depend on actual time boundaries, so we just verify callbacks work
	epochDuration := 50 * time.Millisecond
	timer := NewEpochTimer(epochDuration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var callbackCount atomic.Int32

	timer.RegisterCallback(func(epoch uint64) {
		callbackCount.Add(1)
	})

	timer.Start(ctx)

	// Wait for callbacks
	time.Sleep(300 * time.Millisecond)
	timer.Stop()

	count := callbackCount.Load()
	require.Greater(t, count, int32(0), "Should have at least one callback (initial)")
}

func TestEpochTimer_FixedEpochZeroTime(t *testing.T) {
	// Test that epoch zero time is fixed (Jan 1, 2024)
	timer := NewEpochTimer(1 * time.Hour)

	// Calculate what epoch we should be in
	epochZeroTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedEpoch := uint64(time.Since(epochZeroTime) / (1 * time.Hour))

	actualEpoch := timer.CalculateCurrentEpoch()

	// Should match expected epoch (allow for timing differences)
	require.InDelta(t, expectedEpoch, actualEpoch, 1,
		"Epoch should be calculated from fixed Jan 1, 2024 reference")
}
