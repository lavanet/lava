package provideroptimizer

import (
	"fmt"
	"math"
	stdrand "math/rand"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/score"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// Randomizer interface allows switching between global probabilistic RNG and deterministic RNG for testing
type Randomizer interface {
	Float64() float64
	Intn(n int) int
}

// globalRandomizer uses the thread-safe global random generator from utils/rand
type globalRandomizer struct{}

func (g globalRandomizer) Float64() float64 {
	return rand.Float64()
}

func (g globalRandomizer) Intn(n int) int {
	return rand.Intn(n)
}

// mutexRandomizer wraps *rand.Rand with a mutex for thread safety
type mutexRandomizer struct {
	mu  sync.Mutex
	rng *stdrand.Rand
}

func (m *mutexRandomizer) Float64() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rng.Float64()
}

func (m *mutexRandomizer) Intn(n int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rng.Intn(n)
}

// WeightedSelector implements continuous weighted random selection based on
// composite QoS scores. It replaces the tier-based selection system with a
// probability-based approach where providers are selected according to their
// overall quality without artificial tier boundaries.
type WeightedSelector struct {
	// Configuration weights for different QoS metrics (should sum to 1.0)
	availabilityWeight float64 // Default: 0.3 (30% weight)
	latencyWeight      float64 // Default: 0.3 (30% weight)
	syncWeight         float64 // Default: 0.2 (20% weight)
	stakeWeight        float64 // Default: 0.2 (20% weight)

	// Minimum selection probability to ensure all providers get some traffic
	minSelectionChance float64 // Default: 0.01 (1% minimum)

	// Strategy-specific adjustments for weights
	strategy Strategy

	// Random number generator (defaults to global probabilistic RNG)
	rng Randomizer

	// Adaptive max configuration (Phase 2)
	useAdaptiveLatencyMax bool                      // Feature flag for adaptive latency max
	adaptiveLatencyGetter func() (p10, p90 float64) // Function to get adaptive P10-P90 bounds
	useAdaptiveSyncMax    bool                      // Feature flag for adaptive sync max
	adaptiveSyncGetter    func() (p10, p90 float64) // Function to get adaptive P10-P90 bounds for sync
}

// ProviderScore represents a provider's calculated scores for selection
type ProviderScore struct {
	Address         string  // Provider address
	CompositeScore  float64 // Normalized 0-1 composite score
	SelectionWeight float64 // Score adjusted by strategy and stake
}

// SelectionStats contains detailed information about provider selection for debugging
type SelectionStats struct {
	ProviderScores   []ProviderScoreDetails // Scores for all candidates
	RNGValue         float64                // Random number used for selection
	SelectedProvider string                 // The provider that was selected
}

// ProviderScoreDetails contains detailed scoring information for a single provider
type ProviderScoreDetails struct {
	Address      string  // Provider address
	Availability float64 // Availability score (0-1)
	Latency      float64 // Latency score (0-1)
	Sync         float64 // Sync score (0-1)
	Stake        float64 // Stake score (0-1)
	Composite    float64 // Combined QoS score (0-1)
}

// WeightedSelectorConfig holds configuration options for creating a WeightedSelector
type WeightedSelectorConfig struct {
	AvailabilityWeight    float64
	LatencyWeight         float64
	SyncWeight            float64
	StakeWeight           float64
	MinSelectionChance    float64
	Strategy              Strategy
	UseAdaptiveLatencyMax bool                      // Phase 2: Enable adaptive max for latency
	AdaptiveLatencyGetter func() (p10, p90 float64) // Phase 2: Function to get adaptive P10-P90 bounds
	UseAdaptiveSyncMax    bool                      // Phase 2: Enable adaptive max for sync
	AdaptiveSyncGetter    func() (p10, p90 float64) // Phase 2: Function to get adaptive P10-P90 bounds for sync
}

// DefaultWeightedSelectorConfig returns a configuration with balanced default weights
func DefaultWeightedSelectorConfig() WeightedSelectorConfig {
	return WeightedSelectorConfig{
		AvailabilityWeight: 0.3,  // Availability remains critical (30%)
		LatencyWeight:      0.3,  // Latency shares equal emphasis (30%)
		SyncWeight:         0.2,  // Sync is third priority (20%)
		StakeWeight:        0.2,  // Stake provides meaningful influence (20%)
		MinSelectionChance: 0.01, // 1% minimum chance to prevent starvation
		Strategy:           StrategyBalanced,
	}
}

// NewWeightedSelector creates a new WeightedSelector with the given configuration
func NewWeightedSelector(config WeightedSelectorConfig) *WeightedSelector {
	// Validate and normalize weights
	//
	// Important: we only want to fallback the *weights* if they are invalid, but keep
	// other user choices (Strategy / MinSelectionChance) intact.
	validateWeight := func(name string, w float64) bool {
		if math.IsNaN(w) || math.IsInf(w, 0) || w < 0 {
			utils.LavaFormatWarning("invalid weighted selector weight, must be finite and >= 0",
				nil,
				utils.LogAttr("weightName", name),
				utils.LogAttr("weight", w),
			)
			return false
		}
		return true
	}

	weightsValid := validateWeight("availability", config.AvailabilityWeight) &&
		validateWeight("latency", config.LatencyWeight) &&
		validateWeight("sync", config.SyncWeight) &&
		validateWeight("stake", config.StakeWeight)

	totalWeight := config.AvailabilityWeight + config.LatencyWeight + config.SyncWeight + config.StakeWeight
	if !weightsValid || math.IsNaN(totalWeight) || math.IsInf(totalWeight, 0) || totalWeight <= 0 {
		utils.LavaFormatWarning("weighted selector weights sum to zero/negative or contain invalid values, using default weights", nil,
			utils.LogAttr("totalWeight", totalWeight),
			utils.LogAttr("availabilityWeight", config.AvailabilityWeight),
			utils.LogAttr("latencyWeight", config.LatencyWeight),
			utils.LogAttr("syncWeight", config.SyncWeight),
			utils.LogAttr("stakeWeight", config.StakeWeight),
		)

		defaultCfg := DefaultWeightedSelectorConfig()
		config.AvailabilityWeight = defaultCfg.AvailabilityWeight
		config.LatencyWeight = defaultCfg.LatencyWeight
		config.SyncWeight = defaultCfg.SyncWeight
		config.StakeWeight = defaultCfg.StakeWeight
		totalWeight = 1.0 // default weights sum to 1.0
	}

	if math.Abs(totalWeight-1.0) > 0.001 {
		utils.LavaFormatWarning("weighted selector weights do not sum to 1.0, normalizing",
			nil,
			utils.LogAttr("totalWeight", totalWeight),
			utils.LogAttr("availabilityWeight", config.AvailabilityWeight),
			utils.LogAttr("latencyWeight", config.LatencyWeight),
			utils.LogAttr("syncWeight", config.SyncWeight),
			utils.LogAttr("stakeWeight", config.StakeWeight),
		)
		// Normalize weights to sum to 1.0
		config.AvailabilityWeight /= totalWeight
		config.LatencyWeight /= totalWeight
		config.SyncWeight /= totalWeight
		config.StakeWeight /= totalWeight
	}

	return &WeightedSelector{
		availabilityWeight:    config.AvailabilityWeight,
		latencyWeight:         config.LatencyWeight,
		syncWeight:            config.SyncWeight,
		stakeWeight:           config.StakeWeight,
		minSelectionChance:    config.MinSelectionChance,
		strategy:              config.Strategy,
		rng:                   globalRandomizer{},
		useAdaptiveLatencyMax: config.UseAdaptiveLatencyMax,
		adaptiveLatencyGetter: config.AdaptiveLatencyGetter,
		useAdaptiveSyncMax:    config.UseAdaptiveSyncMax,
		adaptiveSyncGetter:    config.AdaptiveSyncGetter,
	}
}

// SetDeterministicSeed sets a specific seed for the weighted selector's RNG
// This should ONLY be used for testing to ensure deterministic selection
func (ws *WeightedSelector) SetDeterministicSeed(seed int64) {
	ws.rng = &mutexRandomizer{
		rng: stdrand.New(stdrand.NewSource(seed)),
	}
}

// CalculateScore computes a composite score for a provider based on QoS metrics and stake
// Returns a normalized score between 0 and 1, where higher is better
func (ws *WeightedSelector) CalculateScore(
	qos *pairingtypes.QualityOfServiceReport,
	stake sdk.Coin,
	totalStake sdk.Coin,
) float64 {
	// Extract individual scores from QoS report:
	// - availability is in [0,1] (higher is better)
	// - latency/sync are in seconds (lower is better) and are clamped by the optimizer
	availability, err := qos.Availability.Float64()
	if err != nil {
		utils.LavaFormatWarning("could not parse availability score, using 0", err)
		availability = 0
	}

	latency, err := qos.Latency.Float64()
	if err != nil {
		utils.LavaFormatWarning("could not parse latency score, using worst latency", err)
		latency = score.WorstLatencyScore
	}

	sync, err := qos.Sync.Float64()
	if err != nil {
		utils.LavaFormatWarning("could not parse sync score, using worst sync", err)
		sync = score.WorstSyncScore
	}

	// Normalize individual metrics to 0-1 range where higher is better
	availabilityScore := ws.normalizeAvailability(availability) // Rescale [0.9, 1.0] → [0.0, 1.0]

	// For latency and sync, lower raw values are better, so we invert them
	latencyScore := ws.normalizeLatency(latency)       // Convert to 0-1 where higher is better
	syncScore := ws.normalizeSync(sync)                // Convert to 0-1 where higher is better
	stakeScore := ws.normalizeStake(stake, totalStake) // Normalize stake

	// Apply strategy-specific adjustments before weighting
	latencyScore, syncScore = ws.applyStrategyAdjustments(latencyScore, syncScore)

	// Calculate weighted composite score
	composite := availabilityScore*ws.availabilityWeight +
		latencyScore*ws.latencyWeight +
		syncScore*ws.syncWeight +
		stakeScore*ws.stakeWeight

	// Ensure minimum selection chance
	if composite < ws.minSelectionChance {
		composite = ws.minSelectionChance
	}

	// Clamp to [0, 1] range
	if composite > 1.0 {
		composite = 1.0
	}

	return composite
}

// normalizeAvailability converts availability score to 0-1 range using simple rescaling
// Input: availability in [0, 1] range (higher is better, e.g., 0.98 = 98% success rate)
// Output: normalized score where 1.0 = best, 0.0 = worst
//
// Phase 1 (Simple Rescaling):
//   - Rescales [MIN_ACCEPTABLE, 1.0] → [0.0, 1.0]
//   - Below MIN_ACCEPTABLE (0.90) = score of 0.0
//   - Formula: normalized = (availability - 0.90) / (1.0 - 0.90) = (availability - 0.90) / 0.10
//   - This achieves 100% range utilization vs current ~5% (0.95-1.0 → 0.95-1.0)
//
// Example:
//   - Provider with 100% availability (1.00) → score 1.0 (perfect)
//   - Provider with 99% availability (0.99) → score 0.9
//   - Provider with 95% availability (0.95) → score 0.5
//   - Provider with 90% availability (0.90) → score 0.0 (threshold)
//   - Provider with 85% availability (0.85) → score 0.0 (below threshold)
//
// Advantages:
//   - ✅ Uses full [0,1] range (100% vs 5%)
//   - ✅ Zero complexity, no state, no memory overhead
//   - ✅ Clear business rule: < 90% availability = unacceptable
//   - ✅ Stable and predictable
func (ws *WeightedSelector) normalizeAvailability(availability float64) float64 {
	// Phase 1: Simple Rescaling
	const minAcceptable = score.MinAcceptableAvailability // 0.90
	const maxAvailability = 1.0

	// Below minimum threshold = score of 0
	if availability < minAcceptable {
		utils.LavaFormatDebug("availability below minimum threshold",
			utils.LogAttr("raw_availability", availability),
			utils.LogAttr("min_acceptable", minAcceptable),
			utils.LogAttr("normalized_score", 0.0),
			utils.LogAttr("below_threshold", true),
		)
		return 0.0
	}

	// Rescale [minAcceptable, 1.0] → [0.0, 1.0]
	// Formula: (availability - min) / (max - min)
	normalized := (availability - minAcceptable) / (maxAvailability - minAcceptable)

	// Clamp to [0, 1] (should already be in range after above check, but defensive)
	if normalized > 1.0 {
		normalized = 1.0
	}
	if normalized < 0.0 {
		normalized = 0.0
	}

	utils.LavaFormatDebug("normalized availability (Phase 1 rescaling)",
		utils.LogAttr("raw_availability", availability),
		utils.LogAttr("min_acceptable", minAcceptable),
		utils.LogAttr("rescale_range", maxAvailability-minAcceptable),
		utils.LogAttr("normalized_score", normalized),
		utils.LogAttr("below_threshold", false),
	)

	return normalized
}

// normalizeLatency converts latency score to 0-1 range where higher is better
// Input: latency in seconds (lower is better)
// Output: normalized score where 1.0 = best, 0.0 = worst
//
// Phase 2 (Hybrid P10-P90 Approach):
//   - Uses P10-P90 adaptive range from T-Digest for better distribution
//   - Formula: normalized = 1 - (clamp(latency, P10, P90) - P10) / (P90 - P10)
//   - This provides 85-95% range utilization vs 70-85% with P95-only
//   - Robust to both-sided outliers (excludes bottom 10% and top 10%)
//
// Phase 1 (Fallback):
//   - Uses fixed maximum expected latency (score.WorstLatencyScore = 30s)
//   - Formula: normalized = 1 - (latency / maxLatency)
func (ws *WeightedSelector) normalizeLatency(latency float64) float64 {
	// Phase 2: Adaptive P10-P90 normalization (if enabled)
	if ws.useAdaptiveLatencyMax && ws.adaptiveLatencyGetter != nil {
		p10, p90 := ws.adaptiveLatencyGetter()

		// Validate adaptive bounds
		if p10 <= 0 || p90 <= 0 || p90 <= p10 {
			// Invalid bounds, fallback to Phase 1
			utils.LavaFormatWarning("invalid adaptive latency bounds, falling back to fixed max",
				nil,
				utils.LogAttr("p10", p10),
				utils.LogAttr("p90", p90),
			)
		} else {
			// Clamp latency to P10-P90 range
			clampedLatency := latency
			wasClampedLow := false
			wasClampedHigh := false

			if clampedLatency < p10 {
				clampedLatency = p10
				wasClampedLow = true
			}
			if clampedLatency > p90 {
				clampedLatency = p90
				wasClampedHigh = true
			}

			// Normalize: 1 - (Li - P10) / (P90 - P10)
			// This gives better distribution than simple 1 - (L / P90)
			normalized := 1.0 - (clampedLatency-p10)/(p90-p10)

			// Clamp to [0, 1] (should already be in range, but defensive)
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1.0 {
				normalized = 1.0
			}

			// Log normalization details for distribution visualization
			utils.LavaFormatTrace("[LatencyNormalization] P10-P90 normalization applied",
				utils.LogAttr("raw_latency", latency),
				utils.LogAttr("clamped_latency", clampedLatency),
				utils.LogAttr("p10", p10),
				utils.LogAttr("p90", p90),
				utils.LogAttr("range_p10_p90", p90-p10),
				utils.LogAttr("normalized_score", normalized),
				utils.LogAttr("was_clamped_low", wasClampedLow),
				utils.LogAttr("was_clamped_high", wasClampedHigh),
			)

			return normalized
		}
	}

	// Phase 1 (Fallback): Use fixed maximum expected latency
	const maxLatency = score.WorstLatencyScore
	if latency <= 0 {
		utils.LavaFormatTrace("[LatencyNormalization] Perfect latency",
			utils.LogAttr("latency", latency),
			utils.LogAttr("normalized_score", 1.0),
		)
		return 1.0 // Perfect latency
	}

	// Normalize to 0-1 range, inverted so lower latency = higher score
	normalized := 1.0 - (latency / maxLatency)
	if normalized < 0 {
		normalized = 0 // Clamp to 0 for extremely high latency
	}

	// Log normalization details for distribution visualization
	utils.LavaFormatTrace("[LatencyNormalization] Fixed max normalization applied",
		utils.LogAttr("raw_latency", latency),
		utils.LogAttr("max_latency", maxLatency),
		utils.LogAttr("normalized_score", normalized),
	)

	return normalized
}

// normalizeSync converts sync lag to 0-1 range where higher is better
// Input: sync lag in seconds (lower is better)
// Output: normalized score where 1.0 = best, 0.0 = worst
//
// Phase 2 (Hybrid P10-P90 Approach):
//   - Uses P10-P90 adaptive range from T-Digest for better distribution
//   - Formula: normalized = 1 - (clamp(syncLag, P10, P90) - P10) / (P90 - P10)
//   - This provides 85-95% range utilization vs 55-70% with P95-only
//   - Robust to both-sided outliers (excludes bottom 10% and top 10%)
//
// Phase 1 (Fallback):
//   - Uses fixed maximum expected sync lag (score.WorstSyncScore = 1200s)
//   - Formula: normalized = 1 - (syncLag / maxSyncLag)
func (ws *WeightedSelector) normalizeSync(syncLag float64) float64 {
	// Phase 2: Adaptive P10-P90 normalization (if enabled)
	if ws.useAdaptiveSyncMax && ws.adaptiveSyncGetter != nil {
		p10, p90 := ws.adaptiveSyncGetter()

		// Validate adaptive bounds
		if p10 <= 0 || p90 <= 0 || p90 <= p10 {
			// Invalid bounds, fallback to Phase 1
			utils.LavaFormatWarning("invalid adaptive sync bounds, falling back to fixed max",
				nil,
				utils.LogAttr("p10", p10),
				utils.LogAttr("p90", p90),
			)
		} else {
			// Clamp sync lag to P10-P90 range
			clampedSyncLag := syncLag
			wasClampedLow := false
			wasClampedHigh := false

			if clampedSyncLag < p10 {
				clampedSyncLag = p10
				wasClampedLow = true
			}
			if clampedSyncLag > p90 {
				clampedSyncLag = p90
				wasClampedHigh = true
			}

			// Normalize: 1 - (Si - P10) / (P90 - P10)
			// This gives better distribution than simple 1 - (S / P90)
			normalized := 1.0 - (clampedSyncLag-p10)/(p90-p10)

			// Clamp to [0, 1] (should already be in range, but defensive)
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1.0 {
				normalized = 1.0
			}

			// Log normalization details for distribution visualization
			utils.LavaFormatTrace("[SyncNormalization] P10-P90 normalization applied",
				utils.LogAttr("raw_sync_lag", syncLag),
				utils.LogAttr("clamped_sync_lag", clampedSyncLag),
				utils.LogAttr("p10", p10),
				utils.LogAttr("p90", p90),
				utils.LogAttr("range_p10_p90", p90-p10),
				utils.LogAttr("normalized_score", normalized),
				utils.LogAttr("was_clamped_low", wasClampedLow),
				utils.LogAttr("was_clamped_high", wasClampedHigh),
			)

			return normalized
		}
	}

	// Phase 1 (Fallback): Use fixed maximum expected sync lag
	const maxSyncLag = score.WorstSyncScore
	if syncLag <= 0 {
		utils.LavaFormatTrace("[SyncNormalization] Perfect sync",
			utils.LogAttr("sync_lag", syncLag),
			utils.LogAttr("normalized_score", 1.0),
		)
		return 1.0 // Perfect sync
	}

	// Normalize to 0-1 range, inverted so lower sync = higher score
	normalized := 1.0 - (syncLag / maxSyncLag)
	if normalized < 0 {
		normalized = 0 // Clamp to 0 for extremely poor sync
	}

	// Log normalization details for distribution visualization
	utils.LavaFormatTrace("[SyncNormalization] Fixed max normalization applied",
		utils.LogAttr("raw_sync_lag", syncLag),
		utils.LogAttr("max_sync_lag", maxSyncLag),
		utils.LogAttr("normalized_score", normalized),
	)

	return normalized
}

// normalizeStake converts stake to 0-1 range relative to total stake
// Uses square root scaling to reduce whale dominance while maintaining incentives
func (ws *WeightedSelector) normalizeStake(stake sdk.Coin, totalStake sdk.Coin) float64 {
	if totalStake.IsZero() || stake.IsZero() {
		return 0.0
	}

	// Calculate stake ratio
	stakeFloat := float64(stake.Amount.Int64())
	totalStakeFloat := float64(totalStake.Amount.Int64())

	if totalStakeFloat == 0 {
		return 0.0
	}

	stakeRatio := stakeFloat / totalStakeFloat

	// Cap at 1.0 to prevent errors
	if stakeRatio > 1.0 {
		stakeRatio = 1.0
	}

	// Square root scaling: reduces whale dominance with diminishing returns
	// Example: 10% stake → 0.316 (3x boost), 80% stake → 0.894 (slight boost)
	// This reduces the gap between large and small stakers by ~17%
	normalized := math.Sqrt(stakeRatio)

	utils.LavaFormatDebug("normalized stake (square root scaling)",
		utils.LogAttr("stake", stakeFloat),
		utils.LogAttr("total_stake", totalStakeFloat),
		utils.LogAttr("stake_ratio", stakeRatio),
		utils.LogAttr("normalized_score", normalized),
		utils.LogAttr("boost_factor", normalized/stakeRatio),
	)

	return normalized
}

// applyStrategyAdjustments modifies scores based on the configured strategy
// This allows different strategies to emphasize different metrics
func (ws *WeightedSelector) applyStrategyAdjustments(latency, sync float64) (float64, float64) {
	switch ws.strategy {
	case StrategyLatency:
		// Boost latency importance by squaring good latency scores
		// This makes the difference between good and great latency more pronounced
		if latency > 0.7 {
			latency = math.Pow(latency, 0.8) // Less aggressive penalty for high scores
		}

	case StrategySyncFreshness:
		// Boost sync importance by squaring good sync scores
		if sync > 0.7 {
			sync = math.Pow(sync, 0.8) // Less aggressive penalty for high scores
		}

	case StrategyAccuracy, StrategyDistributed:
		// Slightly flatten the curve to encourage more provider diversity
		latency = math.Pow(latency, 1.2)
		sync = math.Pow(sync, 1.2)

	case StrategyCost, StrategyPrivacy:
		// No adjustments needed for cost and privacy strategies
		// Cost is handled by exploration chance, privacy by provider count limits
	}

	return latency, sync
}

// SelectProvider selects a provider using weighted random selection
// Providers with higher composite scores have higher probability of selection
func (ws *WeightedSelector) SelectProvider(
	providerScores []ProviderScore,
) string {
	selected, _ := ws.SelectProviderWithStats(providerScores, nil)
	return selected
}

// SelectProviderWithStats selects a provider using weighted random selection
// and returns detailed selection statistics if scoreDetails is provided
func (ws *WeightedSelector) SelectProviderWithStats(
	providerScores []ProviderScore,
	scoreDetails []ProviderScoreDetails,
) (string, *SelectionStats) {
	if len(providerScores) == 0 {
		return "", nil
	}

	// Handle single provider case
	if len(providerScores) == 1 {
		stats := &SelectionStats{
			ProviderScores:   scoreDetails,
			RNGValue:         0.0,
			SelectedProvider: providerScores[0].Address,
		}
		return providerScores[0].Address, stats
	}

	// Calculate total weighted score
	totalScore := 0.0
	for _, ps := range providerScores {
		totalScore += ps.SelectionWeight
	}

	if totalScore <= 0 {
		// Fallback to uniform random selection if all scores are zero
		utils.LavaFormatWarning("all provider scores are zero, using uniform selection", nil)
		selected := providerScores[ws.rng.Intn(len(providerScores))].Address
		stats := &SelectionStats{
			ProviderScores:   scoreDetails,
			RNGValue:         0.0,
			SelectedProvider: selected,
		}
		return selected, stats
	}

	// Generate random value in [0, totalScore)
	randomValue := ws.rng.Float64() * totalScore

	// Use cumulative probability to select provider
	cumulativeScore := 0.0
	for _, ps := range providerScores {
		cumulativeScore += ps.SelectionWeight
		if randomValue <= cumulativeScore {
			utils.LavaFormatTrace("[WeightedSelector] selected provider",
				utils.LogAttr("address", ps.Address),
				utils.LogAttr("score", ps.SelectionWeight),
				utils.LogAttr("totalScore", totalScore),
				utils.LogAttr("randomValue", randomValue),
			)
			stats := &SelectionStats{
				ProviderScores:   scoreDetails,
				RNGValue:         randomValue,
				SelectedProvider: ps.Address,
			}
			return ps.Address, stats
		}
	}

	// Fallback to last provider (should rarely happen due to floating point precision)
	utils.LavaFormatWarning("weighted selection fallback to last provider", nil,
		utils.LogAttr("totalScore", totalScore),
		utils.LogAttr("randomValue", randomValue),
	)
	selected := providerScores[len(providerScores)-1].Address
	stats := &SelectionStats{
		ProviderScores:   scoreDetails,
		RNGValue:         randomValue,
		SelectedProvider: selected,
	}
	return selected, stats
}

// CalculateProviderScores computes scores for all providers
func (ws *WeightedSelector) CalculateProviderScores(
	allAddresses []string,
	ignoredProviders map[string]struct{},
	providerDataGetter func(string) (*pairingtypes.QualityOfServiceReport, time.Time, bool),
	stakeGetter func(string) int64,
) ([]ProviderScore, map[string]*metrics.OptimizerQoSReport, []ProviderScoreDetails) {
	providerScores := make([]ProviderScore, 0, len(allAddresses))
	qosReports := make(map[string]*metrics.OptimizerQoSReport)
	scoreDetails := make([]ProviderScoreDetails, 0, len(allAddresses))

	// Calculate total stake
	totalStake := int64(0)
	for _, addr := range allAddresses {
		if _, ignored := ignoredProviders[addr]; ignored {
			continue
		}
		totalStake += stakeGetter(addr)
	}

	totalStakeCoin := sdk.NewCoin("ulava", sdk.NewInt(totalStake))

	// Calculate scores for each provider
	for _, providerAddress := range allAddresses {
		if _, ignored := ignoredProviders[providerAddress]; ignored {
			continue
		}

		qos, _, found := providerDataGetter(providerAddress)
		if !found || qos == nil {
			utils.LavaFormatWarning("[WeightedSelector] could not get QoS for provider",
				nil,
				utils.LogAttr("provider", providerAddress),
			)
			continue
		}

		stake := stakeGetter(providerAddress)
		stakeCoin := sdk.NewCoin("ulava", sdk.NewInt(stake))

		// Extract individual scores for detailed reporting
		latency, sync, availability := qos.GetScoresFloat64()

		// Calculate normalized scores
		availabilityScore := ws.normalizeAvailability(availability)
		latencyScore := ws.normalizeLatency(latency)
		syncScore := ws.normalizeSync(sync)
		stakeScore := ws.normalizeStake(stakeCoin, totalStakeCoin)

		// Calculate composite score
		compositeScore := ws.CalculateScore(qos, stakeCoin, totalStakeCoin)

		providerScore := ProviderScore{
			Address:         providerAddress,
			CompositeScore:  compositeScore,
			SelectionWeight: compositeScore,
		}
		providerScores = append(providerScores, providerScore)

		// Store detailed scores for selection stats
		scoreDetails = append(scoreDetails, ProviderScoreDetails{
			Address:      providerAddress,
			Availability: availabilityScore,
			Latency:      latencyScore,
			Sync:         syncScore,
			Stake:        stakeScore,
			Composite:    compositeScore,
		})

		// Create QoS report for metrics
		qosReports[providerAddress] = &metrics.OptimizerQoSReport{
			ProviderAddress:   providerAddress,
			SyncScore:         sync,
			AvailabilityScore: availability,
			LatencyScore:      latency,
			GenericScore:      compositeScore,
			// Add selection stats - normalized scores used in selection algorithm
			SelectionAvailability: availabilityScore,
			SelectionLatency:      latencyScore,
			SelectionSync:         syncScore,
			SelectionStake:        stakeScore,
			SelectionComposite:    compositeScore,
		}

		utils.LavaFormatTrace("[WeightedSelector] calculated provider score",
			utils.LogAttr("provider", providerAddress),
			utils.LogAttr("compositeScore", compositeScore),
			utils.LogAttr("availability", availability),
			utils.LogAttr("latency", latency),
			utils.LogAttr("sync", sync),
			utils.LogAttr("stake", stake),
		)
	}

	return providerScores, qosReports, scoreDetails
}

// GetConfig returns the current configuration
func (ws *WeightedSelector) GetConfig() WeightedSelectorConfig {
	return WeightedSelectorConfig{
		AvailabilityWeight:    ws.availabilityWeight,
		LatencyWeight:         ws.latencyWeight,
		SyncWeight:            ws.syncWeight,
		StakeWeight:           ws.stakeWeight,
		MinSelectionChance:    ws.minSelectionChance,
		Strategy:              ws.strategy,
		UseAdaptiveLatencyMax: ws.useAdaptiveLatencyMax,
		AdaptiveLatencyGetter: ws.adaptiveLatencyGetter,
		UseAdaptiveSyncMax:    ws.useAdaptiveSyncMax,
		AdaptiveSyncGetter:    ws.adaptiveSyncGetter,
	}
}

// UpdateStrategy changes the strategy and recalculates any strategy-dependent parameters
func (ws *WeightedSelector) UpdateStrategy(strategy Strategy) {
	ws.strategy = strategy
}

// FormatSelectionStats formats selection stats as a string for the header
// Format: [provider1: availability, latency, sync, stake, composite] [provider2: ...] | RNG: <value> | Selected: <provider>
func (stats *SelectionStats) FormatSelectionStats() string {
	if stats == nil {
		return ""
	}

	var result strings.Builder

	// Format each provider's scores
	for i, ps := range stats.ProviderScores {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString("[")
		result.WriteString(ps.Address)
		result.WriteString(": ")
		result.WriteString(fmt.Sprintf("%.3f, %.3f, %.3f, %.3f, %.3f", ps.Availability, ps.Latency, ps.Sync, ps.Stake, ps.Composite))
		result.WriteString("]")
	}

	// Add RNG value
	result.WriteString(" | RNG: ")
	result.WriteString(fmt.Sprintf("%.6f", stats.RNGValue))

	// Add selected provider
	result.WriteString(" | Selected: ")
	result.WriteString(stats.SelectedProvider)

	return result.String()
}
