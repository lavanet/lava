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

// The timer and the deadline fire at almost the same moment. The condition must
// never run once the context is done.
func TestSleepUntilTimeOrConditionReached_NoConditionCallAfterContextDone(t *testing.T) {
	rpcps := &RPCProviderServer{}
	var lateCalls atomic.Int32
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		rpcps.SleepUntilTimeOrConditionReached(ctx, 2*time.Millisecond, func() bool {
			if ctx.Err() != nil {
				lateCalls.Add(1)
			}
			return false
		})
		cancel()
	}
	require.Zero(t, lateCalls.Load())
}

func TestSleepUntilTimeOrConditionReached_CancelledContextSkipsCondition(t *testing.T) {
	rpcps := &RPCProviderServer{}
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	slept := rpcps.SleepUntilTimeOrConditionReached(ctx, 5*time.Millisecond, func() bool {
		calls.Add(1)
		cancel() // cancelled while the condition runs; it must not run again
		return false
	})
	require.Equal(t, int32(1), calls.Load())
	require.Less(t, slept, 2*time.Second)
}

func TestSleepUntilTimeOrConditionReached_NonPositiveQueryTimeDoesNotSpin(t *testing.T) {
	for _, queryTime := range []time.Duration{0, -time.Second} {
		rpcps := &RPCProviderServer{}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		var calls atomic.Int32
		slept := rpcps.SleepUntilTimeOrConditionReached(ctx, queryTime, func() bool {
			calls.Add(1)
			return false
		})
		cancel()
		require.GreaterOrEqual(t, slept, 80*time.Millisecond, "queryTime %v", queryTime)
		// one check per minSleepStep (10ms): about 10, never thousands
		require.LessOrEqual(t, calls.Load(), int32(30), "queryTime %v", queryTime)
	}
}

func TestSleepUntilTimeOrConditionReached_DeadlineShorterThanQueryTime(t *testing.T) {
	rpcps := &RPCProviderServer{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	var calls atomic.Int32
	slept := rpcps.SleepUntilTimeOrConditionReached(ctx, 5*time.Second, func() bool {
		calls.Add(1)
		return false
	})
	require.Zero(t, calls.Load())
	require.GreaterOrEqual(t, slept, 20*time.Millisecond)
	require.Less(t, slept, 2*time.Second)
}
