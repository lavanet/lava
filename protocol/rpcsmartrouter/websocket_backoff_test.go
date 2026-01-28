package rpcsmartrouter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExponentialBackoff_ExponentialGrowth(t *testing.T) {
	backoff := NewWebSocketBackoff()
	backoff.JitterFactor = 0 // Disable jitter for predictable testing

	delay1, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 100*time.Millisecond, delay1, "first delay should be initial interval")

	delay2, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 200*time.Millisecond, delay2, "second delay should be 2x initial")

	delay3, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 400*time.Millisecond, delay3, "third delay should be 4x initial")

	delay4, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 800*time.Millisecond, delay4, "fourth delay should be 8x initial")
}

func TestExponentialBackoff_RespectsMaxInterval(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialInterval: 1 * time.Second,
		MaxInterval:     5 * time.Second,
		Multiplier:      10.0,
		JitterFactor:    0,
		MaxRetries:      0, // Unlimited
	}

	delay1, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 1*time.Second, delay1)

	delay2, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 5*time.Second, delay2, "should be capped at max interval (10s > 5s)")

	delay3, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 5*time.Second, delay3, "should still be capped at max interval")
}

func TestExponentialBackoff_MaxRetriesExceeded(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
		MaxRetries:      3,
	}

	_, ok := backoff.NextBackoff() // Attempt 1
	require.True(t, ok)
	_, ok = backoff.NextBackoff() // Attempt 2
	require.True(t, ok)
	_, ok = backoff.NextBackoff() // Attempt 3
	require.True(t, ok)
	_, ok = backoff.NextBackoff() // Attempt 4 - exceeds max
	assert.False(t, ok, "should return false when max retries exceeded")
}

func TestExponentialBackoff_ResetAfterSuccess(t *testing.T) {
	backoff := NewWebSocketBackoff()
	backoff.JitterFactor = 0

	backoff.NextBackoff() // 100ms
	backoff.NextBackoff() // 200ms
	assert.Equal(t, 2, backoff.Attempt())

	backoff.Reset()

	assert.Equal(t, 0, backoff.Attempt(), "attempt counter should be reset")

	delay, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 100*time.Millisecond, delay, "should restart from initial after reset")
}

func TestExponentialBackoff_JitterAddsRandomness(t *testing.T) {
	backoff := NewWebSocketBackoff()
	// JitterFactor is 0.3 by default

	delays := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		backoff.Reset()
		delays[i], _ = backoff.NextBackoff()
	}

	// Check that we have variation (not all the same)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "jitter should produce varied delays")

	// Check all delays are within expected range (100ms ± 30%)
	for _, d := range delays {
		assert.GreaterOrEqual(t, d, 70*time.Millisecond, "delay should be >= 70ms (100ms - 30%%)")
		assert.LessOrEqual(t, d, 130*time.Millisecond, "delay should be <= 130ms (100ms + 30%%)")
	}
}

func TestExponentialBackoff_UnlimitedRetries(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
		MaxRetries:      0, // Unlimited
	}

	// Should never return false
	for i := 0; i < 100; i++ {
		_, ok := backoff.NextBackoff()
		assert.True(t, ok, "unlimited retries should never exceed max")
	}
}

func TestExponentialBackoff_Clone(t *testing.T) {
	original := NewWebSocketBackoff()
	original.JitterFactor = 0

	original.NextBackoff() // Advance state
	original.NextBackoff()
	assert.Equal(t, 2, original.Attempt())

	clone := original.Clone()

	assert.Equal(t, original.InitialInterval, clone.InitialInterval)
	assert.Equal(t, original.MaxInterval, clone.MaxInterval)
	assert.Equal(t, original.Multiplier, clone.Multiplier)
	assert.Equal(t, original.MaxRetries, clone.MaxRetries)
	assert.Equal(t, 0, clone.Attempt(), "clone should have reset state")
}

func TestExponentialBackoff_BackoffProgression(t *testing.T) {
	// Verify the documented progression from WEBSOCKET_SUPPORT.md
	backoff := NewWebSocketBackoff()
	backoff.JitterFactor = 0 // Disable jitter for exact values

	expected := []time.Duration{
		100 * time.Millisecond,  // Attempt 1
		200 * time.Millisecond,  // Attempt 2
		400 * time.Millisecond,  // Attempt 3
		800 * time.Millisecond,  // Attempt 4
		1600 * time.Millisecond, // Attempt 5
		3200 * time.Millisecond, // Attempt 6
		6400 * time.Millisecond, // Attempt 7
		12800 * time.Millisecond,// Attempt 8
		25600 * time.Millisecond,// Attempt 9
		30 * time.Second,        // Attempt 10 (capped)
	}

	for i, exp := range expected {
		delay, ok := backoff.NextBackoff()
		require.True(t, ok, "attempt %d should succeed", i+1)
		assert.Equal(t, exp, delay, "attempt %d delay mismatch", i+1)
	}

	// 11th attempt should fail (MaxRetries = 10)
	_, ok := backoff.NextBackoff()
	assert.False(t, ok, "11th attempt should exceed max retries")
}

func TestExponentialBackoff_ConcurrentAccess(t *testing.T) {
	backoff := NewWebSocketBackoff()

	// Run multiple goroutines accessing the backoff
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				backoff.NextBackoff()
				backoff.Attempt()
				if j%10 == 0 {
					backoff.Reset()
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	// If we reach here without panic, concurrent access is safe
}

func TestNewExponentialBackoff_CustomParameters(t *testing.T) {
	backoff := NewExponentialBackoff(
		500*time.Millisecond, // initial
		10*time.Second,       // max
		3.0,                  // multiplier
		5,                    // max retries
	)
	backoff.JitterFactor = 0

	delay1, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 500*time.Millisecond, delay1)

	delay2, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 1500*time.Millisecond, delay2, "should be 3x initial")

	delay3, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 4500*time.Millisecond, delay3, "should be 9x initial")

	delay4, ok := backoff.NextBackoff()
	require.True(t, ok)
	assert.Equal(t, 10*time.Second, delay4, "should be capped at max (13.5s > 10s)")
}
