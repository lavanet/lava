package chainstate

import (
	"math"
	"sort"
	"sync"
	"time"
)

// ChainState holds shared per-chain state for majorityBaseline protection and outlier detection.
// One instance is created per chain and shared across ConsumerSessionManager, ProviderOptimizer,
// and RelayProcessor to guard block-height writes from poisoning.
type ChainState struct {
	majorityBaseline int64
	outlierThreshold int64
	mu               sync.RWMutex
}

// SetMajorityBaseline updates the majorityBaseline and outlier threshold.
// Called by ConsumerSessionManager after probe consensus.
// Allows decreases (for chain reverts).
func (cs *ChainState) SetMajorityBaseline(floor int64, threshold int64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.majorityBaseline = floor
	cs.outlierThreshold = threshold
}

// GetMajorityBaseline returns the current majorityBaseline value.
func (cs *ChainState) GetMajorityBaseline() int64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.majorityBaseline
}

// IsOutlier returns true if newBlock exceeds majorityBaseline + outlierThreshold.
// The majorityBaseline is the consensus-agreed block height — the single point of truth.
// The threshold is intentionally generous (20x update interval) to act as a safety net
// against extreme outliers (e.g., cross-chain contamination), not a tight filter.
// Returns false (allows all) when majorityBaseline == 0 (cold start / no consensus yet).
func (cs *ChainState) IsOutlier(newBlock int64) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.majorityBaseline == 0 {
		return false
	}

	return newBlock > cs.majorityBaseline+cs.outlierThreshold
}

// ComputeMajorityBaseline takes a set of LatestBlock values from probe results and computes
// the majorityBaseline via majority consensus. Blocks are grouped into buckets of +/-bucketWidth.
// The winning bucket must contain >=50% of respondents AND >=2 providers.
// Returns max(winningBucket) as the majorityBaseline, or 0 if no consensus.
func ComputeMajorityBaseline(blocks []int64, bucketWidth int64) int64 {
	if len(blocks) < 2 {
		return 0
	}

	sorted := make([]int64, len(blocks))
	copy(sorted, blocks)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	total := len(sorted)
	majority := (total / 2) + 1 // strict majority: >50%

	bestCount := 0
	bestMax := int64(0)
	windowWidth := 2 * bucketWidth

	// Sliding window on sorted array: for each starting index i, find the
	// furthest index j where sorted[j] - sorted[i] <= windowWidth.
	// This is equivalent to checking [center-bucketWidth, center+bucketWidth]
	// for center = sorted[i] + bucketWidth.
	j := 0
	for i := 0; i < total; i++ {
		for j < total && sorted[j]-sorted[i] <= windowWidth {
			j++
		}
		count := j - i // number of blocks in [sorted[i], sorted[i]+windowWidth]
		if count > bestCount {
			bestCount = count
			bestMax = sorted[j-1] // max in this window
		}
	}

	if bestCount >= majority && bestCount >= 2 {
		return bestMax
	}

	return 0
}

// ComputeBucketWidth returns the number of blocks that correspond to the given
// time window, with a minimum of 2. This normalizes bucket width across chains
// with different block times (e.g., 2min -> 300 blocks on Solana, 10 blocks on Ethereum).
func ComputeBucketWidth(timeWindow, avgBlockTime time.Duration) int64 {
	if avgBlockTime <= 0 {
		return 2
	}
	width := int64(math.Ceil(float64(timeWindow) / float64(avgBlockTime)))
	if width < 2 {
		return 2
	}
	return width
}

// ComputeOutlierThreshold returns the maximum allowed block distance above the majorityBaseline
// before a value is rejected as an outlier. Set to 20x the number of blocks that can
// be produced during the majorityBaseline update interval, with a minimum of 100.
// This is intentionally generous — the threshold is a safety net against extreme outliers
// (e.g., the Feb 17 incident with 20,000 blocks ahead), not a tight filter.
func ComputeOutlierThreshold(updateInterval, avgBlockTime time.Duration) int64 {
	if avgBlockTime <= 0 {
		return 100
	}
	blocksPerInterval := int64(updateInterval / avgBlockTime)
	threshold := blocksPerInterval * 20
	if threshold < 100 {
		return 100
	}
	return threshold
}
