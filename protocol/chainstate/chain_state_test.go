package chainstate

import (
	"sync"
	"testing"
	"time"

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
	cs.SetMajorityBaseline(1000, 200)

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

	cs.SetMajorityBaseline(1000, 200)
	require.Equal(t, int64(1000), cs.GetMajorityBaseline())
	require.False(t, cs.IsOutlier(1200))
	require.True(t, cs.IsOutlier(1201))

	// Update to new values
	cs.SetMajorityBaseline(2000, 100)
	require.Equal(t, int64(2000), cs.GetMajorityBaseline())
	require.False(t, cs.IsOutlier(2100))
	require.True(t, cs.IsOutlier(2101))
}

func TestSetMajorityBaseline_AllowsDecrease(t *testing.T) {
	cs := &ChainState{}

	cs.SetMajorityBaseline(2000, 200)
	require.Equal(t, int64(2000), cs.GetMajorityBaseline())

	// Decrease baseline (chain revert scenario)
	cs.SetMajorityBaseline(1500, 200)
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
	cs.SetMajorityBaseline(1000, 200)

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent writers
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer wg.Done()
			cs.SetMajorityBaseline(int64(1000+i), 200)
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
	cs.SetMajorityBaseline(1000, 100)
	require.True(t, cs.IsOutlier(1200)) // outlier while baseline is active

	// Consensus failed — reset
	cs.SetMajorityBaseline(0, 0)
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
