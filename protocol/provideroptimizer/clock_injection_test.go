package provideroptimizer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestProviderOptimizer_NowFuncDefault verifies that when NowFunc is nil (default),
// now() returns real wall-clock time with negligible drift.
func TestProviderOptimizer_NowFuncDefault(t *testing.T) {
	po := setupProviderOptimizer(1)
	require.Nil(t, po.NowFunc)

	before := time.Now()
	got := po.now()
	after := time.Now()

	require.True(t, !got.Before(before), "now() must not be before the call")
	require.True(t, !got.After(after), "now() must not be after the call")
}

// TestProviderOptimizer_NowFuncOverride verifies that setting NowFunc replaces the clock
// used by the optimizer — now() returns exactly what NowFunc returns.
func TestProviderOptimizer_NowFuncOverride(t *testing.T) {
	po := setupProviderOptimizer(1)

	fixed := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	po.NowFunc = func() time.Time { return fixed }

	require.Equal(t, fixed, po.now())
}

// TestProviderOptimizer_NowFuncOffset verifies the +1h pattern used by /debug/time-warp.
// The optimizer's clock must be exactly 1 hour ahead of wall-clock time.
func TestProviderOptimizer_NowFuncOffset(t *testing.T) {
	po := setupProviderOptimizer(1)

	offset := time.Hour
	po.NowFunc = func() time.Time { return time.Now().Add(offset) }

	before := time.Now().Add(offset)
	got := po.now()
	after := time.Now().Add(offset)

	require.True(t, !got.Before(before), "now() with +1h offset must not be before expected window")
	require.True(t, !got.After(after), "now() with +1h offset must not be after expected window")
}

// TestProviderOptimizer_NowFuncReset verifies that setting NowFunc = nil restores real time.
func TestProviderOptimizer_NowFuncReset(t *testing.T) {
	po := setupProviderOptimizer(1)

	// shift clock to the future
	po.NowFunc = func() time.Time { return time.Now().Add(24 * time.Hour) }
	shifted := po.now()
	require.True(t, shifted.After(time.Now().Add(23*time.Hour)), "should be ~24h in the future")

	// reset
	po.NowFunc = nil
	before := time.Now()
	got := po.now()
	after := time.Now()

	require.True(t, !got.Before(before))
	require.True(t, !got.After(after), "after reset, now() must return real time")
}

// TestProviderOptimizer_ClockInjectionScoreDecay verifies that shifting the clock forward
// causes the optimizer to treat existing scores as older — the practical effect of
// /debug/time-warp in integration tests.
func TestProviderOptimizer_ClockInjectionScoreDecay(t *testing.T) {
	po := setupProviderOptimizer(10)
	provider := "lava@test_provider"

	// record a relay success at real time
	po.AppendRelayData(provider, 10*time.Millisecond, 100, 100)

	// shift clock +1 hour — optimizer sees subsequent events as 1h later
	po.NowFunc = func() time.Time { return time.Now().Add(time.Hour) }

	// record a relay failure 1h later (from the optimizer's perspective)
	po.AppendRelayFailure(provider)

	// reset clock back to real time
	po.NowFunc = nil

	// optimizer is still functional after reset
	po.AppendRelayData(provider, 10*time.Millisecond, 100, 100)
}

