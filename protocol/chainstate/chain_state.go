package chainstate

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
)

// ChainState holds shared per-chain state: the chain-wide latestBlock tracker plus the
// majorityBaseline + outlier threshold used to guard block-height writes from poisoning.
// One instance per chain, shared across all consumer-side components that read or write
// the chain head.
type ChainState struct {
	majorityBaseline int64
	outlierThreshold int64

	// latestBlock is the chain-wide "current head" tracker. Writes are monotonic and
	// outlier-guarded (see SetLatestBlock); downward moves go through AlignLatestBlockWithConsensus.
	// lastUpdated is the wall-clock time of the most recent advancing write; consumed by
	// the optimizer's sync-score baseline and the optional TTL-aware read path.
	// latestBlockTTL=0 (default) disables TTL; a positive value enables lazy read-time expiry.
	latestBlock    int64
	lastUpdated    time.Time
	latestBlockTTL time.Duration

	// chainID + metricsManager wire the unification observability metrics surfaced by
	// the D2 action item. metricsManager is nil-tolerant via metrics.SafeMetrics so unit
	// tests that construct ChainState via the bare literal `&ChainState{}` continue to work.
	chainID        string
	metricsManager metrics.ConsumerMetricsManagerInf

	mu sync.RWMutex
}

// NewChainState constructs a ChainState with the given latestBlock TTL, chainID, and
// metrics manager. Pass 0 for latestBlockTTL to disable TTL (default per §11.5 Option A).
// The chainID is used as the `spec` label on emitted Prometheus metrics. metricsManager
// is wrapped with metrics.SafeMetrics so a nil value becomes a no-op without panicking.
func NewChainState(latestBlockTTL time.Duration, chainID string, metricsManager metrics.ConsumerMetricsManagerInf) *ChainState {
	return &ChainState{
		latestBlockTTL: latestBlockTTL,
		chainID:        chainID,
		metricsManager: metrics.SafeMetrics(metricsManager),
	}
}

// metrics returns a non-nil ConsumerMetricsManagerInf. Tests that construct ChainState via
// the bare literal `&ChainState{}` leave metricsManager as the zero-value nil interface;
// SafeMetrics turns that into a no-op.
func (cs *ChainState) metricsHandle() metrics.ConsumerMetricsManagerInf {
	return metrics.SafeMetrics(cs.metricsManager)
}

// SetLatestBlock applies a monotonic, outlier-guarded update and returns the post-call
// snapshot (latestBlock, lastUpdated, advanced). `advanced` is true when this call moved
// the value forward; false when rejected (outlier or stale-monotonic). On rejection the
// returned (block, time) reflect existing state, so callers that need a paired sync-baseline
// can use the snapshot directly without a follow-up GetLatestBlockWithTime read — closing
// a set/read race window. The outlier check is an inline copy of IsOutlier to avoid
// re-acquiring the read lock under the write lock.
func (cs *ChainState) SetLatestBlock(block int64) (int64, time.Time, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Outlier guard — reject when majorityBaseline is set and block exceeds floor+threshold.
	// majorityBaseline == 0 (cold start / no consensus) disables the guard.
	if cs.majorityBaseline > 0 && block > cs.majorityBaseline+cs.outlierThreshold {
		utils.LavaFormatWarning("latestBlock update rejected: outlier block height", nil,
			utils.LogAttr("block", block),
			utils.LogAttr("majorityBaseline", cs.majorityBaseline),
			utils.LogAttr("outlierThreshold", cs.outlierThreshold),
		)
		// M2: surface IsOutlier rejections via Prometheus so operators can detect
		// adversarial providers and chain misconfiguration without log scraping.
		cs.metricsHandle().SetLatestBlockOutlierRejected(cs.chainID)
		return cs.latestBlock, cs.lastUpdated, false
	}

	// Monotonic — never decrease via SetLatestBlock. Downward consensus moves go through
	// AlignLatestBlockWithConsensus (chain reverts + poisoning recovery).
	if block <= cs.latestBlock {
		return cs.latestBlock, cs.lastUpdated, false
	}

	cs.latestBlock = block
	cs.lastUpdated = time.Now()
	// M3: track the unified tracker's value for staleness detection + dashboard correlation.
	cs.metricsHandle().SetChainStateLatestBlock(cs.chainID, cs.latestBlock)
	return cs.latestBlock, cs.lastUpdated, true
}

// GetLatestBlock returns the unified chain-head tracker value and a "found" flag.
// Returns (0, false) when the value has never been set, or when latestBlockTTL > 0
// and the most recent advancing write is older than the TTL (lazy read-time expiry).
// The (int64, bool) shape matches today's GetSeenBlock contract so call sites migrate
// without semantic change.
func (cs *ChainState) GetLatestBlock() (int64, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.latestBlock == 0 {
		return 0, false
	}
	if cs.latestBlockTTL > 0 && time.Since(cs.lastUpdated) > cs.latestBlockTTL {
		return 0, false
	}
	return cs.latestBlock, true
}

// GetLatestBlockWithTime returns the latestBlock value, the wall-clock time of the most
// recent advancing write, and a "found" flag — atomically under one read lock. Required
// for paired-snapshot consumers (e.g. sync-lag scoring) that would otherwise race between
// two separate reads. Returns (0, time.Time{}, false) on the same conditions as GetLatestBlock.
func (cs *ChainState) GetLatestBlockWithTime() (int64, time.Time, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.latestBlock == 0 {
		return 0, time.Time{}, false
	}
	if cs.latestBlockTTL > 0 && time.Since(cs.lastUpdated) > cs.latestBlockTTL {
		return 0, time.Time{}, false
	}
	return cs.latestBlock, cs.lastUpdated, true
}

// AlignLatestBlockWithConsensus realigns latestBlock with the consensus floor when consensus
// reports a head lower than what the local tracker holds. Two scenarios it covers:
//
//  1. Poisoning recovery: a contaminated relay/probe pushed latestBlock far above the real
//     chain head while majorityBaseline was 0 (cold start / consensus gap). On the next
//     consensus, gap > threshold → snap down to floor.
//  2. Chain revert / probe-driven downward consensus: the chain reorged or the probe
//     majority moved backwards; gap may be small (within threshold) but consensus is
//     authoritative — snap down to floor anyway.
//
// Called after each probe consensus. floor == 0 means "no consensus" → no-op (don't reset
// to 0; that would erase the local tracker mid-cold-start). The threshold parameter
// discriminates poisoning (gap > threshold, warning log) from revert (gap ≤ threshold,
// debug log) for operator visibility.
// AlignLatestBlockWithConsensus accepts apiInterface so emitted metrics carry per-CSM
// breakdown — multiple CSMs (one per apiInterface) call this on the same shared ChainState
// each consensus cycle, and operators want to confirm each wire-up is firing independently.
func (cs *ChainState) AlignLatestBlockWithConsensus(floor int64, threshold int64, apiInterface string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	mm := cs.metricsHandle()

	if floor <= 0 {
		// floor=0 is the "no consensus this cycle" path; treat as a non-event for the
		// outcome counter — the existing SetMajorityBaselineConsensusFailure metric
		// already covers this case from the caller's perspective.
		return
	}
	if cs.latestBlock <= floor {
		// M1: healthy_no_change — the dominant outcome in steady state. Operators rely
		// on this counter ticking to confirm Phase 2b is wired up (D2 action item).
		mm.SetAlignLatestBlockOutcome(cs.chainID, apiInterface, "healthy_no_change")
		return
	}

	previous := cs.latestBlock
	gap := previous - floor

	if gap > threshold {
		utils.LavaFormatWarning("latestBlock reset: value was outlier relative to consensus", nil,
			utils.LogAttr("previousBlock", previous),
			utils.LogAttr("consensusFloor", floor),
			utils.LogAttr("outlierThreshold", threshold),
		)
		// M1: poisoning_reset — page-worthy alert source.
		mm.SetAlignLatestBlockOutcome(cs.chainID, apiInterface, "poisoning_reset")
	} else {
		utils.LavaFormatDebug("latestBlock reset: realigning with downward consensus",
			utils.LogAttr("previousBlock", previous),
			utils.LogAttr("consensusFloor", floor),
			utils.LogAttr("outlierThreshold", threshold),
		)
		// M1: revert — dashboard signal for chain reorg activity.
		mm.SetAlignLatestBlockOutcome(cs.chainID, apiInterface, "revert")
	}

	// M5: gap gauge so operators can distinguish small reverts from large poisoning
	// resets. Set on every realignment (revert + poisoning_reset); not updated on
	// healthy_no_change so the gauge reflects "last realignment magnitude".
	mm.SetAlignLatestBlockGap(cs.chainID, apiInterface, gap)

	cs.latestBlock = floor
	cs.lastUpdated = time.Now()
	// M3: keep the tracker gauge consistent with the realignment.
	mm.SetChainStateLatestBlock(cs.chainID, cs.latestBlock)
}

// SetLatestBlockFromSharedState is identical to SetLatestBlock but also emits the
// shared-state propagation counter (M7) so operators can confirm cross-consumer state
// coherence is firing in sharded deployments. Called from rpcconsumer / rpcsmartrouter
// relay paths when the shared cache backend returned a higher seenBlock than the local
// tracker (rpcconsumer_server.go:927 and rpcsmartrouter_server.go mirror).
//
// Wrapping SetLatestBlock here (rather than plumbing a metrics manager into the servers)
// keeps the metric local to the package that owns the unified tracker; the per-relay
// hot path picks up the additional emit through the existing nil-tolerant metrics handle.
func (cs *ChainState) SetLatestBlockFromSharedState(block int64) (int64, time.Time, bool) {
	cs.metricsHandle().SetSharedStatePropagation(cs.chainID)
	return cs.SetLatestBlock(block)
}

// SetMajorityBaseline updates the majorityBaseline and outlier threshold.
// Called by ConsumerSessionManager after probe consensus. Allows decreases (for chain reverts).
// apiInterface is used as a metric label so operators can see per-CSM updates; the underlying
// baseline value is single (shared across CSMs since ChainState is per-chain).
func (cs *ChainState) SetMajorityBaseline(floor int64, threshold int64, apiInterface string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.majorityBaseline = floor
	cs.outlierThreshold = threshold
	// M4: gauge of current consensus-derived floor. Pairs with M3 — operators correlate
	// (latestBlock, majorityBaseline) to spot drift / wedged consensus.
	cs.metricsHandle().SetMajorityBaselineGauge(cs.chainID, apiInterface, floor)
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
