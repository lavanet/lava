package chainstate

import (
	"sync"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/stretchr/testify/require"
)

func TestIsOutlier_ColdStart(t *testing.T) {
	// majorityBaseline == 0 means no consensus yet — all blocks should be allowed
	cs := &ChainState{}
	require.False(t, cs.IsOutlier(1_000_000))
	require.False(t, cs.IsOutlier(999_999_999))
	require.False(t, cs.IsOutlier(0))
}

func TestIsOutlier_WithMajorityBaseline(t *testing.T) {
	cs := &ChainState{}
	cs.SetMajorityBaseline(1000, 200, "test")

	// Within threshold — not outlier
	require.False(t, cs.IsOutlier(1000))   // exactly at baseline
	require.False(t, cs.IsOutlier(1100))   // above baseline but within threshold
	require.False(t, cs.IsOutlier(1200))   // exactly at baseline + threshold
	require.False(t, cs.IsOutlier(900))    // below baseline
	require.False(t, cs.IsOutlier(0))      // zero block

	// Exceeds threshold — outlier
	require.True(t, cs.IsOutlier(1201))    // one above threshold
	require.True(t, cs.IsOutlier(2000))    // way above threshold
}

func TestSetMajorityBaseline_UpdatesValues(t *testing.T) {
	cs := &ChainState{}

	cs.SetMajorityBaseline(1000, 200, "test")
	require.Equal(t, int64(1000), cs.GetMajorityBaseline())
	require.False(t, cs.IsOutlier(1200))
	require.True(t, cs.IsOutlier(1201))

	// Update to new values
	cs.SetMajorityBaseline(2000, 100, "test")
	require.Equal(t, int64(2000), cs.GetMajorityBaseline())
	require.False(t, cs.IsOutlier(2100))
	require.True(t, cs.IsOutlier(2101))
}

func TestSetMajorityBaseline_AllowsDecrease(t *testing.T) {
	cs := &ChainState{}

	cs.SetMajorityBaseline(2000, 200, "test")
	require.Equal(t, int64(2000), cs.GetMajorityBaseline())

	// Decrease baseline (chain revert scenario)
	cs.SetMajorityBaseline(1500, 200, "test")
	require.Equal(t, int64(1500), cs.GetMajorityBaseline())
	require.False(t, cs.IsOutlier(1700))
	require.True(t, cs.IsOutlier(1701))
}

func TestGetMajorityBaseline_DefaultZero(t *testing.T) {
	cs := &ChainState{}
	require.Equal(t, int64(0), cs.GetMajorityBaseline())
}

func TestChainState_ConcurrentAccess(t *testing.T) {
	cs := &ChainState{}
	cs.SetMajorityBaseline(1000, 200, "test")

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent writers
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer wg.Done()
			cs.SetMajorityBaseline(int64(1000+i), 200, "test")
		}(i)
	}

	// Concurrent readers
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			_ = cs.GetMajorityBaseline()
			_ = cs.IsOutlier(1500)
		}()
	}

	wg.Wait()
	// No race conditions — if we get here without panic/deadlock, the test passes
	// Verify state is still consistent
	floor := cs.GetMajorityBaseline()
	require.True(t, floor >= 1000 && floor < 2000)
}

// --- ComputeMajorityBaseline tests ---

func TestComputeMajorityBaseline_PlanExample(t *testing.T) {
	// From the plan: LAV1, bucketWidth=8
	blocks := []int64{1000093, 1000098, 1000100, 1000100, 1020000}
	result := ComputeMajorityBaseline(blocks, 8)
	require.Equal(t, int64(1000100), result)
}

func TestComputeMajorityBaseline_AllSameBlock(t *testing.T) {
	blocks := []int64{5000, 5000, 5000, 5000}
	result := ComputeMajorityBaseline(blocks, 10)
	require.Equal(t, int64(5000), result)
}

func TestComputeMajorityBaseline_NoMajority(t *testing.T) {
	// 4 providers, all widely spread — no bucket gets >=3 (majority of 4)
	blocks := []int64{100, 300, 500, 700}
	result := ComputeMajorityBaseline(blocks, 5)
	require.Equal(t, int64(0), result)
}

func TestComputeMajorityBaseline_SingleProvider(t *testing.T) {
	blocks := []int64{1000}
	result := ComputeMajorityBaseline(blocks, 10)
	require.Equal(t, int64(0), result) // need >=2
}

func TestComputeMajorityBaseline_TwoProvidersAgree(t *testing.T) {
	blocks := []int64{1000, 1005}
	result := ComputeMajorityBaseline(blocks, 10)
	require.Equal(t, int64(1005), result) // 2/2 = 100%, >=2 providers
}

func TestComputeMajorityBaseline_TwoProvidersDisagree(t *testing.T) {
	blocks := []int64{1000, 2000}
	result := ComputeMajorityBaseline(blocks, 5)
	require.Equal(t, int64(0), result) // neither bucket has >=2
}

func TestComputeMajorityBaseline_ThreeProviders_TwoAgree(t *testing.T) {
	blocks := []int64{1000, 1002, 5000}
	result := ComputeMajorityBaseline(blocks, 10)
	require.Equal(t, int64(1002), result) // 2/3 = 67%, majority=2
}

func TestComputeMajorityBaseline_EmptyInput(t *testing.T) {
	require.Equal(t, int64(0), ComputeMajorityBaseline(nil, 10))
	require.Equal(t, int64(0), ComputeMajorityBaseline([]int64{}, 10))
}

func TestComputeMajorityBaseline_UnsortedInput(t *testing.T) {
	// Input doesn't need to be pre-sorted
	blocks := []int64{1000100, 1020000, 1000093, 1000100, 1000098}
	result := ComputeMajorityBaseline(blocks, 8)
	require.Equal(t, int64(1000100), result)
}

func TestComputeMajorityBaseline_ReturnsMaxOfWinningBucket(t *testing.T) {
	blocks := []int64{990, 995, 1000, 1005, 1010}
	result := ComputeMajorityBaseline(blocks, 10)
	require.Equal(t, int64(1010), result) // all within +/-10, max = 1010
}

func TestComputeMajorityBaseline_DoesNotMutateInput(t *testing.T) {
	blocks := []int64{300, 100, 200}
	original := make([]int64, len(blocks))
	copy(original, blocks)
	ComputeMajorityBaseline(blocks, 10)
	require.Equal(t, original, blocks) // input slice unchanged
}

// --- Edge case tests (Task 7.4) ---

func TestComputeMajorityBaseline_StaticProvidersExcluded(t *testing.T) {
	// Simulates the filtering done in probeProviders Phase 2b:
	// static providers return latestBlock=0, which are filtered out before consensus.
	// After filtering, only the non-zero blocks are passed to ComputeMajorityBaseline.
	allBlocks := []int64{0, 1000, 0, 1002, 0} // 2 static providers (0), 2 real, 1 static
	var filtered []int64
	for _, b := range allBlocks {
		if b > 0 {
			filtered = append(filtered, b)
		}
	}
	result := ComputeMajorityBaseline(filtered, 10)
	require.Equal(t, int64(1002), result) // 2/2 = 100%, both agree
}

func TestComputeMajorityBaseline_AllZeroBlocks(t *testing.T) {
	// All probes failed or all static — after filtering, empty slice
	allBlocks := []int64{0, 0, 0}
	var filtered []int64
	for _, b := range allBlocks {
		if b > 0 {
			filtered = append(filtered, b)
		}
	}
	result := ComputeMajorityBaseline(filtered, 10)
	require.Equal(t, int64(0), result) // no consensus possible
}

func TestIsOutlier_AfterReset(t *testing.T) {
	// Simulates 7.2: majorityBaseline was set, then reset to 0 on consensus failure.
	// After reset, IsOutlier should allow all blocks.
	cs := &ChainState{}
	cs.SetMajorityBaseline(1000, 100, "test")
	require.True(t, cs.IsOutlier(1200)) // outlier while baseline is active

	// Consensus failed — reset
	cs.SetMajorityBaseline(0, 0, "test")
	require.False(t, cs.IsOutlier(1200))   // now allowed
	require.False(t, cs.IsOutlier(999999)) // anything allowed
}

// --- ComputeBucketWidth tests ---

func TestComputeBucketWidth_Chains(t *testing.T) {
	tests := []struct {
		name           string
		timeWindow     time.Duration
		avgBlockTime   time.Duration
		expectedWidth  int64
	}{
		{"Solana (0.4s)", 2 * time.Minute, 400 * time.Millisecond, 300},
		{"Ethereum (13s)", 2 * time.Minute, 13 * time.Second, 10},     // ceil(120/13) = 10
		{"Lava (15s)", 2 * time.Minute, 15 * time.Second, 8},
		{"Cosmos Hub (6.5s)", 2 * time.Minute, 6500 * time.Millisecond, 19}, // ceil(120/6.5) = 19
		{"Bitcoin (600s)", 2 * time.Minute, 600 * time.Second, 2},     // ceil(120/600) = 1, floor to 2
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeBucketWidth(tt.timeWindow, tt.avgBlockTime)
			require.Equal(t, tt.expectedWidth, result)
		})
	}
}

func TestComputeBucketWidth_ZeroBlockTime(t *testing.T) {
	require.Equal(t, int64(2), ComputeBucketWidth(2*time.Minute, 0))
}

func TestComputeBucketWidth_NegativeBlockTime(t *testing.T) {
	require.Equal(t, int64(2), ComputeBucketWidth(2*time.Minute, -1*time.Second))
}

// --- ComputeOutlierThreshold tests ---

func TestComputeOutlierThreshold_NormalCases(t *testing.T) {
	tests := []struct {
		name           string
		updateInterval time.Duration
		avgBlockTime   time.Duration
		expected       int64
	}{
		{"Lava (30s interval, 15s blocks)", 30 * time.Second, 15 * time.Second, 100},     // floor(30/15)*20 = 40, clamped to 100
		{"Ethereum (30s interval, 13s blocks)", 30 * time.Second, 13 * time.Second, 100},  // floor(30/13)*20 = 40, clamped to 100
		{"Solana (30s interval, 0.4s blocks)", 30 * time.Second, 400 * time.Millisecond, 1500}, // floor(30/0.4)*20 = 75*20 = 1500
		{"Longer interval (2min, 15s blocks)", 2 * time.Minute, 15 * time.Second, 160},   // floor(120/15)*20 = 8*20 = 160
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeOutlierThreshold(tt.updateInterval, tt.avgBlockTime)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeOutlierThreshold_ZeroBlockTime(t *testing.T) {
	require.Equal(t, int64(100), ComputeOutlierThreshold(30*time.Second, 0))
}

func TestComputeOutlierThreshold_MinimumFloor(t *testing.T) {
	// Very short interval relative to block time -> result < 100 -> clamped to 100
	require.Equal(t, int64(100), ComputeOutlierThreshold(5*time.Second, 15*time.Second))
}

// --- SetLatestBlock tests (A6) ---

func TestSetLatestBlock_Monotonic(t *testing.T) {
	cs := &ChainState{}

	// First write advances from 0 → 1000
	_, _, ok := cs.SetLatestBlock(1000)
	require.True(t, ok)
	got, found := cs.GetLatestBlock()
	require.True(t, found)
	require.Equal(t, int64(1000), got)

	// Higher block advances
	_, _, ok = cs.SetLatestBlock(1500)
	require.True(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1500), got)

	// Equal block is rejected (monotonic, strict)
	_, _, ok = cs.SetLatestBlock(1500)
	require.False(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1500), got, "equal block should not change stored value")

	// Lower block is rejected
	_, _, ok = cs.SetLatestBlock(1499)
	require.False(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1500), got, "lower block should not change stored value")

	// Much lower block is rejected
	_, _, ok = cs.SetLatestBlock(1)
	require.False(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1500), got)
}

func TestSetLatestBlock_OutlierRejected(t *testing.T) {
	cs := &ChainState{}
	cs.SetMajorityBaseline(1000, 200, "test") // floor=1000, threshold=200 → reject if block > 1200

	// Within threshold — accepted
	_, _, ok := cs.SetLatestBlock(1100)
	require.True(t, ok)
	got, _ := cs.GetLatestBlock()
	require.Equal(t, int64(1100), got)

	// Exactly at floor + threshold — accepted (boundary inclusive)
	_, _, ok = cs.SetLatestBlock(1200)
	require.True(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1200), got)

	// One above threshold — rejected as outlier
	_, _, ok = cs.SetLatestBlock(1201)
	require.False(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1200), got, "outlier should not change stored value")

	// Way above threshold — rejected as outlier
	_, _, ok = cs.SetLatestBlock(20_000_000)
	require.False(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1200), got, "extreme outlier should not change stored value")
}

func TestSetLatestBlock_IsOutlierGuardOnColdStart(t *testing.T) {
	// majorityBaseline == 0 disables the outlier guard — any block above current is accepted.
	cs := &ChainState{}
	require.Equal(t, int64(0), cs.GetMajorityBaseline(), "precondition: cold start")

	_, _, ok := cs.SetLatestBlock(999_999_999)
	require.True(t, ok)
	got, found := cs.GetLatestBlock()
	require.True(t, found)
	require.Equal(t, int64(999_999_999), got)

	// Even larger block also accepted while in cold start
	_, _, ok = cs.SetLatestBlock(1_000_000_000)
	require.True(t, ok)
	got, _ = cs.GetLatestBlock()
	require.Equal(t, int64(1_000_000_000), got)
}

func TestSetLatestBlock_ReturnsBoolCorrectly(t *testing.T) {
	cs := &ChainState{}

	// Advance: true
	_, _, ok := cs.SetLatestBlock(100)
	require.True(t, ok, "advance from 0 must return true")

	// Equal: false (stale, monotonic)
	_, _, ok = cs.SetLatestBlock(100)
	require.False(t, ok, "equal block must return false (stale)")

	// Lower: false (stale, monotonic)
	_, _, ok = cs.SetLatestBlock(50)
	require.False(t, ok, "lower block must return false (stale)")

	// Higher: true (advance)
	_, _, ok = cs.SetLatestBlock(200)
	require.True(t, ok, "higher block must return true (advance)")

	// Outlier: false
	cs.SetMajorityBaseline(200, 50, "test")
	_, _, ok = cs.SetLatestBlock(300)
	require.False(t, ok, "outlier (300 > 200+50) must return false")
	_, _, ok = cs.SetLatestBlock(1_000_000)
	require.False(t, ok, "extreme outlier must return false")

	// Confirm outliers did not change value
	got, _ := cs.GetLatestBlock()
	require.Equal(t, int64(200), got)
}

// TestSetLatestBlock_ReturnsSnapshot pins SetLatestBlock's atomic-snapshot return contract:
// (latestBlock, lastUpdated, advanced). The advance path returns the new state; both
// rejection paths (outlier, monotonic) return the existing state unchanged. This snapshot
// is what provider_optimizer's sync-lag baseline consumes, closing a set/read race window
// that a follow-up GetLatestBlockWithTime read would have.
func TestSetLatestBlock_ReturnsSnapshot(t *testing.T) {
	cs := &ChainState{}

	// --- Advance path: returns (new block, ~now, true) ---
	before := time.Now()
	block, ts, advanced := cs.SetLatestBlock(1000)
	after := time.Now()
	require.True(t, advanced, "advance must return advanced=true")
	require.Equal(t, int64(1000), block, "advance must return the new block value")
	require.WithinDuration(t, time.Now(), ts, time.Second, "advance must return a fresh lastUpdated near time.Now()")
	require.False(t, ts.Before(before), "lastUpdated must be >= pre-call time")
	require.False(t, ts.After(after), "lastUpdated must be <= post-call time")

	advanceTs := ts // capture for rejection-path equality checks below

	// --- Monotonic-rejected (equal): returns (existing block, existing time, false) ---
	block, ts, advanced = cs.SetLatestBlock(1000)
	require.False(t, advanced, "monotonic-rejected (equal block) must return advanced=false")
	require.Equal(t, int64(1000), block, "monotonic-rejected must return the existing block")
	require.Equal(t, advanceTs, ts, "monotonic-rejected must return the existing lastUpdated unchanged")

	// --- Monotonic-rejected (lower): same shape ---
	block, ts, advanced = cs.SetLatestBlock(500)
	require.False(t, advanced, "monotonic-rejected (lower block) must return advanced=false")
	require.Equal(t, int64(1000), block, "monotonic-rejected must return the existing block")
	require.Equal(t, advanceTs, ts, "monotonic-rejected must return the existing lastUpdated unchanged")

	// --- Outlier-rejected: returns (existing block, existing time, false) ---
	cs.SetMajorityBaseline(1000, 100, "test") // floor=1000, threshold=100 → reject if block > 1100
	block, ts, advanced = cs.SetLatestBlock(2000)
	require.False(t, advanced, "outlier-rejected must return advanced=false")
	require.Equal(t, int64(1000), block, "outlier-rejected must return the existing block (unchanged)")
	require.Equal(t, advanceTs, ts, "outlier-rejected must return the existing lastUpdated unchanged")
}

// --- GetLatestBlock tests (A7) ---

func TestGetLatestBlock_NeverSet(t *testing.T) {
	cs := &ChainState{}
	got, found := cs.GetLatestBlock()
	require.False(t, found, "fresh ChainState must report not-found")
	require.Equal(t, int64(0), got, "fresh ChainState must return 0")
}

func TestGetLatestBlock_AfterSet(t *testing.T) {
	cs := &ChainState{}

	_, _, ok := cs.SetLatestBlock(12345)
	require.True(t, ok)
	got, found := cs.GetLatestBlock()
	require.True(t, found)
	require.Equal(t, int64(12345), got)

	// A subsequent advance is reflected
	_, _, ok = cs.SetLatestBlock(12346)
	require.True(t, ok)
	got, found = cs.GetLatestBlock()
	require.True(t, found)
	require.Equal(t, int64(12346), got)
}

// --- AlignLatestBlockWithConsensus tests (A8) ---
//
// Patterns ported from provider_optimizer_test.go:1188-1278 (TestResetLatestSyncDataIfOutlier_*).
// Contract differs (see A5): the new method snaps down whenever latestBlock > floor, not only
// when the gap exceeds threshold. So "healthy unchanged" in the new contract means
// latestBlock <= floor (not "within threshold").

func TestAlignLatestBlockWithConsensus_PoisonedReset(t *testing.T) {
	cs := &ChainState{}

	// Simulate poisoning: latestBlock pushed to 20M during a consensus gap.
	// Set via SetMajorityBaseline=0 (cold start) which disables the outlier guard.
	_, _, ok := cs.SetLatestBlock(20_000_000)
	require.True(t, ok)

	// Probe consensus arrives with floor=1M, threshold=100 → gap (19M) far exceeds threshold.
	cs.AlignLatestBlockWithConsensus(1_000_000, 100, "test")

	got, found := cs.GetLatestBlock()
	require.True(t, found, "after reset, value remains found")
	require.Equal(t, int64(1_000_000), got, "poisoned value should be reset to floor")
}

func TestAlignLatestBlockWithConsensus_HealthyUnchanged(t *testing.T) {
	// Under the new contract, "healthy" means latestBlock <= floor — no reset needed.
	// Sub-case A: latestBlock < floor (probe consensus is ahead of local tracker).
	cs := &ChainState{}
	_, _, ok := cs.SetLatestBlock(999_995)
	require.True(t, ok)
	cs.AlignLatestBlockWithConsensus(1_000_000, 100, "test")
	got, _ := cs.GetLatestBlock()
	require.Equal(t, int64(999_995), got, "latestBlock < floor: must not be touched")

	// Sub-case B: latestBlock == floor (already aligned).
	cs2 := &ChainState{}
	_, _, ok = cs2.SetLatestBlock(1_000_000)
	require.True(t, ok)
	cs2.AlignLatestBlockWithConsensus(1_000_000, 100, "test")
	got, _ = cs2.GetLatestBlock()
	require.Equal(t, int64(1_000_000), got, "latestBlock == floor: must not be touched")
}

func TestAlignLatestBlockWithConsensus_ZeroBlockUnchanged(t *testing.T) {
	// Cold start: latestBlock = 0 (never set). Even with a positive consensus floor,
	// 0 > floor is false, so no reset fires.
	cs := &ChainState{}
	got, found := cs.GetLatestBlock()
	require.False(t, found)
	require.Equal(t, int64(0), got)

	cs.AlignLatestBlockWithConsensus(1_000_000, 100, "test")

	got, found = cs.GetLatestBlock()
	require.False(t, found, "zero (never-set) block should remain not-found after reset call")
	require.Equal(t, int64(0), got, "zero block should not be touched")
}

// TestSetLatestBlock_IsOutlierGuard pins the invariant that SetLatestBlock's inline outlier
// guard agrees with IsOutlier — a block that IsOutlier rejects must also be rejected by
// SetLatestBlock, and vice-versa. The two have separate inline implementations (because
// sync.RWMutex is non-reentrant); this test fails fast if either drifts.
//
// Method: for each block across the boundary, run SetLatestBlock against a fresh ChainState
// (so the monotonic guard never fires) and compare the accept/reject decision against
// IsOutlier on the same baseline. A fresh ChainState is required because SetLatestBlock
// would otherwise reject same-or-lower blocks via the monotonic guard, conflating the two
// reasons for rejection.
func TestSetLatestBlock_IsOutlierGuard(t *testing.T) {
	const (
		floor     = int64(1000)
		threshold = int64(200)
	)

	// Mix of values: well below, at boundary edges, well above.
	blocks := []int64{
		1,
		floor - 1,
		floor,
		floor + threshold - 1,
		floor + threshold,     // boundary: accepted (IsOutlier uses strict >)
		floor + threshold + 1, // first rejection
		floor + threshold + 100,
		1_000_000,
		20_000_000,
	}

	for _, block := range blocks {
		// Fresh ChainState so monotonic guard never fires.
		cs := &ChainState{}
		cs.SetMajorityBaseline(floor, threshold, "test")

		isOutlier := cs.IsOutlier(block)
		_, _, setOk := cs.SetLatestBlock(block)

		// Equivalence: SetLatestBlock accepts exactly when IsOutlier says "not an outlier".
		// Note: in this fresh-ChainState setup, the monotonic guard rejects only block <= 0,
		// which never appears in the test inputs, so isOutlier and !setOk must match exactly.
		require.Equal(t, !isOutlier, setOk,
			"outlier-guard invariant broken at block=%d: IsOutlier=%v, SetLatestBlock=%v",
			block, isOutlier, setOk,
		)
	}
}

// --- Unification observability metric assertions (M1-M5) ---
//
// These tests pin the contract between ChainState and ConsumerMetricsManagerInf
// surfaced by the D2 action item in unification-demo-env-setup.md. They use a
// recording mock that embeds NoOpConsumerMetrics and overrides only the five
// methods ChainState calls, so adding unrelated interface methods later doesn't
// require updating the mock.

type recordingMetrics struct {
	metrics.NoOpConsumerMetrics

	mu                         sync.Mutex
	alignOutcomes              []alignOutcomeCall
	outlierRejected            []string // chainID
	chainStateLatestBlockGauge []chainStateBlockSet
	majorityBaselineGaugeCalls []majorityBaselineCall
	alignLatestBlockGapCalls   []alignGapCall
	sharedStatePropagations    int
}

type alignOutcomeCall struct {
	chainID, apiInterface, outcome string
}

type chainStateBlockSet struct {
	chainID string
	block   int64
}

type majorityBaselineCall struct {
	chainID, apiInterface string
	value                 int64
}

type alignGapCall struct {
	chainID, apiInterface string
	gap                   int64
}

func (r *recordingMetrics) SetAlignLatestBlockOutcome(chainID, apiInterface, outcome string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alignOutcomes = append(r.alignOutcomes, alignOutcomeCall{chainID, apiInterface, outcome})
}

func (r *recordingMetrics) SetLatestBlockOutlierRejected(chainID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outlierRejected = append(r.outlierRejected, chainID)
}

func (r *recordingMetrics) SetChainStateLatestBlock(chainID string, block int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chainStateLatestBlockGauge = append(r.chainStateLatestBlockGauge, chainStateBlockSet{chainID, block})
}

func (r *recordingMetrics) SetMajorityBaselineGauge(chainID, apiInterface string, value int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.majorityBaselineGaugeCalls = append(r.majorityBaselineGaugeCalls, majorityBaselineCall{chainID, apiInterface, value})
}

func (r *recordingMetrics) SetAlignLatestBlockGap(chainID, apiInterface string, gap int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alignLatestBlockGapCalls = append(r.alignLatestBlockGapCalls, alignGapCall{chainID, apiInterface, gap})
}

func (r *recordingMetrics) SetSharedStatePropagation(chainID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sharedStatePropagations++
}

func TestSetLatestBlock_OutlierRejection_IncrementsMetric(t *testing.T) {
	rec := &recordingMetrics{}
	cs := NewChainState(0, "ETH1", rec)
	cs.SetMajorityBaseline(1000, 100, "jsonrpc") // outliers: > 1100

	_, _, advanced := cs.SetLatestBlock(2000)
	require.False(t, advanced, "outlier write must be rejected")
	require.Equal(t, []string{"ETH1"}, rec.outlierRejected, "M2 counter must increment with chainID label")

	// Healthy write does NOT touch the rejection counter.
	_, _, advanced = cs.SetLatestBlock(1050)
	require.True(t, advanced)
	require.Len(t, rec.outlierRejected, 1, "M2 counter must not increment on healthy advance")
}

func TestSetLatestBlock_Advance_UpdatesGauge(t *testing.T) {
	rec := &recordingMetrics{}
	cs := NewChainState(0, "LAV1", rec)

	_, _, advanced := cs.SetLatestBlock(500)
	require.True(t, advanced)
	require.Equal(t, []chainStateBlockSet{{"LAV1", 500}}, rec.chainStateLatestBlockGauge, "M3 gauge must be set to advanced value")

	// Monotonic-rejected does NOT touch the gauge.
	_, _, advanced = cs.SetLatestBlock(400)
	require.False(t, advanced)
	require.Len(t, rec.chainStateLatestBlockGauge, 1, "M3 gauge must not be set on monotonic rejection")
}

func TestAlignLatestBlockWithConsensus_HealthyNoChange_IncrementsCounter(t *testing.T) {
	rec := &recordingMetrics{}
	cs := NewChainState(0, "LAV1", rec)
	// latestBlock starts at 0, so floor=100 puts us in the healthy_no_change branch.

	cs.AlignLatestBlockWithConsensus(100, 50, "tendermintrpc")

	require.Equal(t, []alignOutcomeCall{{"LAV1", "tendermintrpc", "healthy_no_change"}}, rec.alignOutcomes,
		"M1 counter must increment with outcome=healthy_no_change in steady state")
	require.Empty(t, rec.alignLatestBlockGapCalls, "M5 gap gauge must not be set on healthy_no_change")
}

func TestAlignLatestBlockWithConsensus_Revert_IncrementsCounterAndUpdatesGap(t *testing.T) {
	rec := &recordingMetrics{}
	cs := NewChainState(0, "LAV1", rec)

	// Push latestBlock to 1010 via SetLatestBlock; baseline=0 so no outlier guard fires.
	_, _, _ = cs.SetLatestBlock(1010)
	rec.chainStateLatestBlockGauge = nil // clear M3 advance recorded above; we want to see the revert-driven update

	// Floor=1000, threshold=100 → gap=10, gap <= threshold → revert outcome.
	cs.AlignLatestBlockWithConsensus(1000, 100, "tendermintrpc")

	require.Equal(t, []alignOutcomeCall{{"LAV1", "tendermintrpc", "revert"}}, rec.alignOutcomes,
		"M1 counter must increment with outcome=revert when gap <= threshold")
	require.Equal(t, []alignGapCall{{"LAV1", "tendermintrpc", 10}}, rec.alignLatestBlockGapCalls,
		"M5 gauge must capture the gap (previousLatestBlock - floor)")
	require.Equal(t, []chainStateBlockSet{{"LAV1", 1000}}, rec.chainStateLatestBlockGauge,
		"M3 gauge must be updated to the new realigned value")
}

func TestAlignLatestBlockWithConsensus_PoisoningReset_IncrementsCounterAndUpdatesGap(t *testing.T) {
	rec := &recordingMetrics{}
	cs := NewChainState(0, "LAV1", rec)

	// Push latestBlock to 5000 via SetLatestBlock (no baseline, no outlier guard).
	_, _, _ = cs.SetLatestBlock(5000)
	rec.chainStateLatestBlockGauge = nil

	// Floor=1000, threshold=100 → gap=4000, gap > threshold → poisoning_reset outcome.
	cs.AlignLatestBlockWithConsensus(1000, 100, "rest")

	require.Equal(t, []alignOutcomeCall{{"LAV1", "rest", "poisoning_reset"}}, rec.alignOutcomes,
		"M1 counter must increment with outcome=poisoning_reset when gap > threshold")
	require.Equal(t, []alignGapCall{{"LAV1", "rest", 4000}}, rec.alignLatestBlockGapCalls,
		"M5 gauge must capture the (large) poisoning gap")
	require.Equal(t, []chainStateBlockSet{{"LAV1", 1000}}, rec.chainStateLatestBlockGauge,
		"M3 gauge must snap down to floor on poisoning reset")
}

func TestSetLatestBlockFromSharedState_IncrementsPropagationCounterAndAdvances(t *testing.T) {
	rec := &recordingMetrics{}
	cs := NewChainState(0, "ETH1", rec)

	// Healthy advance via the shared-state path.
	_, _, advanced := cs.SetLatestBlockFromSharedState(500)
	require.True(t, advanced, "first shared-state advance must succeed")
	require.Len(t, rec.outlierRejected, 0, "M2 must not fire on healthy shared-state propagation")
	require.Equal(t, []chainStateBlockSet{{"ETH1", 500}}, rec.chainStateLatestBlockGauge,
		"M3 must reflect the advanced value when the shared-state path advances the tracker")

	// Monotonic-rejected shared-state call still increments M7 (event-counted, not outcome-counted).
	_, _, advanced = cs.SetLatestBlockFromSharedState(400)
	require.False(t, advanced, "lower value must be monotonic-rejected")
	require.Equal(t, 2, rec.sharedStatePropagations, "M7 must increment on every shared-state propagation event regardless of advance/reject")
}

func TestSetMajorityBaseline_UpdatesGauge(t *testing.T) {
	rec := &recordingMetrics{}
	cs := NewChainState(0, "LAV1", rec)

	cs.SetMajorityBaseline(1234, 100, "tendermintrpc")
	cs.SetMajorityBaseline(0, 0, "rest") // consensus failure path — still emits gauge=0

	require.Equal(t, []majorityBaselineCall{
		{"LAV1", "tendermintrpc", 1234},
		{"LAV1", "rest", 0},
	}, rec.majorityBaselineGaugeCalls, "M4 gauge must record per-apiInterface updates including consensus-failure resets")
}
