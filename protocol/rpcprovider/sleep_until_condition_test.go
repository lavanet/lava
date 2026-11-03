package rpcprovider

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSleepUntilTimeOrConditionReached_ReturnsWhenConditionMet(t *testing.T) {
	rpcps := &RPCProviderServer{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var calls atomic.Int32
	slept := rpcps.SleepUntilTimeOrConditionReached(ctx, 10*time.Millisecond, func() bool {
		return calls.Add(1) >= 3
	})

	require.Equal(t, int32(3), calls.Load())
	require.GreaterOrEqual(t, slept, 30*time.Millisecond)
	require.Less(t, slept, 2*time.Second)
}

func TestSleepUntilTimeOrConditionReached_StopsAtDeadlineWithoutBusyLoop(t *testing.T) {
	rpcps := &RPCProviderServer{}
	timeout := 200 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var calls atomic.Int32
	slept := rpcps.SleepUntilTimeOrConditionReached(ctx, 50*time.Millisecond, func() bool {
		calls.Add(1)
		return false
	})

	require.GreaterOrEqual(t, slept, timeout-20*time.Millisecond)
	require.Less(t, slept, timeout+500*time.Millisecond)
	// one check per 50ms step: about 4, never thousands
	require.LessOrEqual(t, calls.Load(), int32(10))
}

func TestSleepUntilTimeOrConditionReached_ExpiredContextReturnsNonNegative(t *testing.T) {
	rpcps := &RPCProviderServer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)

	slept := rpcps.SleepUntilTimeOrConditionReached(ctx, 50*time.Millisecond, func() bool { return false })
	require.GreaterOrEqual(t, slept, time.Duration(0))
	require.Less(t, slept, 50*time.Millisecond)
}

// Run with -race: many concurrent callers whose deadline passes while they wait.
func TestSleepUntilTimeOrConditionReached_ConcurrentCallersNoRaceNoNegative(t *testing.T) {
	rpcps := &RPCProviderServer{}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
			defer cancel()
			slept := rpcps.SleepUntilTimeOrConditionReached(ctx, 50*time.Millisecond, func() bool { return false })
			require.GreaterOrEqual(t, slept, time.Duration(0))
		}()
	}
	wg.Wait()
}
