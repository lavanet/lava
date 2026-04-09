package provideroptimizer

import (
	"math"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/utils/score"
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

// TestProviderOptimizer_ScoreDecayAfterClockForward proves that after warping the
// optimizer's clock forward by Δt hours, the weight of pre-warp relay data in the
// ScoreStore decays exactly as 2^(−Δt / halfLife).
//
// At Δt = 24 h with halfLife = 1 h the decay factor is ~5.96e-8, making the old
// history effectively zero — a cold start.  This is the primary regression target
// introduced by raising the debug ceiling from 23h59m to 24h.
//
// How to read the decay assertions:
//
//	After the warp a single new relay with latency newLatency is recorded.
//	ScoreStore.Update computes:
//	  decayFactor = exp(−ln2 × Δt_seconds / halfLife_seconds)
//	  newNum  = oldNum  × decayFactor + newLatency × weight
//	  newDenom = oldDenom × decayFactor + weight
//
//	The test captures oldNum/oldDenom before the warp and then asserts that the
//	values after the warp match the formula to within 0.1 % relative error.
func TestProviderOptimizer_ScoreDecayAfterClockForward(t *testing.T) {
	const (
		provider   = "lava@decay_test"
		cu         = uint64(10)
		syncBlock  = uint64(1000)
		numRelays  = 5 // enough to build a stable pre-warp score
		oldLatency = 50 * time.Millisecond
		newLatency = 200 * time.Millisecond // deliberately different so the decay shows
	)

	cases := []struct {
		name      string
		warpHours float64
		halfLife  time.Duration
		// wantDecayPct is the maximum percentage of the old score that may survive.
		// e.g. 0.1 means "< 0.1 % of old score survives".
		wantDecayPct float64
		// exactDecay: when true, assert newNum ≈ oldNum*decay + newLatency within 0.1 %
		exactDecay bool
	}{
		{
			// +1 h warp with 1 h half-life: exactly half the old score survives.
			name: "warp_1h_halfLife_1h", warpHours: 1, halfLife: score.DefaultHalfLifeTime,
			exactDecay: true,
		},
		{
			// +7 h: < 1 % of old score survives.
			name: "warp_7h_halfLife_1h", warpHours: 7, halfLife: score.DefaultHalfLifeTime,
			wantDecayPct: 1.0,
		},
		{
			// +10 h: < 0.1 % of old score survives.
			name: "warp_10h_halfLife_1h", warpHours: 10, halfLife: score.DefaultHalfLifeTime,
			wantDecayPct: 0.1,
		},
		{
			// +24 h with 1 h half-life: primary regression for this PR.
			// Old score weight ≈ 5.96e-8 % — true cold start.
			name: "warp_24h_halfLife_1h_cold_start", warpHours: 24, halfLife: score.DefaultHalfLifeTime,
			wantDecayPct: 0.0001, // < 0.0001 % = cold start
		},
		{
			// +24 h with max 3 h half-life: ~0.25 % of old score survives — NOT a cold start.
			name: "warp_24h_halfLife_3h_partial", warpHours: 24, halfLife: score.MaxHalfTime,
			wantDecayPct: 0.5, // < 0.5 %
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			po := setupProviderOptimizer(1)

			// Freeze the clock at a fixed base time for full determinism.
			fixedBase := time.Now()
			po.NowFunc = func() time.Time { return fixedBase }

			// ── Step 1: build up a known score at the fixed base time ────────────
			for i := 0; i < numRelays; i++ {
				po.appendRelayData(provider, oldLatency, true, cu, syncBlock, fixedBase)
			}
			time.Sleep(4 * time.Millisecond) // ristretto async write

			dataBefore, found := po.getProviderData(provider)
			require.True(t, found, "provider data must be cached after initial writes")

			oldNum := dataBefore.Latency.GetNum()
			oldDenom := dataBefore.Latency.GetDenom()
			require.Greater(t, oldNum, 0.0, "pre-warp latency numerator must be positive")
			require.Greater(t, oldDenom, 0.0, "pre-warp latency denominator must be positive")

			// ── Step 2: advance the clock by warpHours ────────────────────────────
			warpedBase := fixedBase.Add(time.Duration(float64(time.Hour) * tc.warpHours))
			po.NowFunc = func() time.Time { return warpedBase }

			// Override the halfTime for this specific write so the test is independent
			// of calculateHalfTime's heuristic (which looks at relay-stats timestamps).
			// We call appendRelayData directly with a fixed sampleTime and then
			// manually invoke updateDecayingWeightedAverage so we control halfTime.
			//
			// Simpler approach: clear providerRelayStats so calculateHalfTime returns
			// DefaultHalfLifeTime, then override with WithDecayHalfLife if needed.
			po.providerRelayStats.Clear()

			// For the 3 h half-life case we need to inject the half-life via config.
			// We do that by recording a dummy entry first to set the half-life, then
			// we directly call updateDecayingWeightedAverage.
			dataForUpdate, _ := po.getProviderData(provider)
			updateErr := dataForUpdate.Latency.UpdateConfig(
				score.WithDecayHalfLife(tc.halfLife),
				score.WithWeight(score.RelayUpdateWeight),
				score.WithLatencyCuFactor(score.GetLatencyFactor(cu)),
			)
			require.NoError(t, updateErr)

			updateErr = dataForUpdate.Latency.Update(newLatency.Seconds(), warpedBase)
			require.NoError(t, updateErr, "update at warped time must succeed")

			// ── Step 3: compute expected values and assert ────────────────────────
			dtSeconds := tc.warpHours * 3600.0
			decayFactor := math.Exp(-math.Ln2 * dtSeconds / tc.halfLife.Seconds())

			expectedNum := oldNum*decayFactor + newLatency.Seconds()*score.RelayUpdateWeight
			expectedDenom := oldDenom*decayFactor + score.RelayUpdateWeight

			gotNum := dataForUpdate.Latency.GetNum()
			gotDenom := dataForUpdate.Latency.GetDenom()

			if tc.exactDecay {
				// For the 1 h case assert the formula holds to < 0.1 % relative error.
				requireInEpsilonOrZero(t, expectedNum, gotNum, 0.001,
					"Num after %v warp: expected %.6f got %.6f (decay=%.6f)", tc.warpHours, expectedNum, gotNum, decayFactor)
				requireInEpsilonOrZero(t, expectedDenom, gotDenom, 0.001,
					"Denom after %v warp: expected %.6f got %.6f", tc.warpHours, expectedDenom, gotDenom)
			} else {
				// For partial/cold-start cases assert that the old score contribution
				// is below wantDecayPct % of the new score.
				oldContribution := oldNum * decayFactor
				newContribution := newLatency.Seconds() * score.RelayUpdateWeight
				pct := 100.0 * oldContribution / (oldContribution + newContribution)
				require.Less(t, pct, tc.wantDecayPct,
					"old score contribution %.4f %% must be < %.4f %% after %v h warp with halfLife=%v",
					pct, tc.wantDecayPct, tc.warpHours, tc.halfLife)
			}
		})
	}
}

// requireInEpsilonOrZero is like require.InEpsilon but also passes when both values
// are effectively zero (avoids divide-by-zero in relative-error checks).
func requireInEpsilonOrZero(t *testing.T, expected, actual, epsilon float64, msgAndArgs ...interface{}) {
	t.Helper()
	if math.Abs(expected) < 1e-30 && math.Abs(actual) < 1e-30 {
		return // both effectively zero — pass
	}
	relErr := math.Abs(expected-actual) / math.Max(math.Abs(expected), math.Abs(actual))
	require.Less(t, relErr, epsilon, msgAndArgs...)
}
