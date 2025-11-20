package provideroptimizer

import (
	"math"
	stdrand "math/rand"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
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
	availabilityWeight float64 // Default: 0.4 (40% weight)
	latencyWeight      float64 // Default: 0.3 (30% weight)
	syncWeight         float64 // Default: 0.2 (20% weight)
	stakeWeight        float64 // Default: 0.1 (10% weight)

	// Minimum selection probability to ensure all providers get some traffic
	minSelectionChance float64 // Default: 0.01 (1% minimum)

	// Strategy-specific adjustments for weights
	strategy Strategy

	// Random number generator (defaults to global probabilistic RNG)
	rng Randomizer
}

// ProviderScore represents a provider's calculated scores for selection
type ProviderScore struct {
	Address         string  // Provider address
	CompositeScore  float64 // Normalized 0-1 composite score
	SelectionWeight float64 // Score adjusted by strategy and stake
}

// WeightedSelectorConfig holds configuration options for creating a WeightedSelector
type WeightedSelectorConfig struct {
	AvailabilityWeight float64
	LatencyWeight      float64
	SyncWeight         float64
	StakeWeight        float64
	MinSelectionChance float64
	Strategy           Strategy
}

// DefaultWeightedSelectorConfig returns a configuration with balanced default weights
func DefaultWeightedSelectorConfig() WeightedSelectorConfig {
	return WeightedSelectorConfig{
		AvailabilityWeight: 0.4,  // Availability is most important (40%)
		LatencyWeight:      0.3,  // Latency is second priority (30%)
		SyncWeight:         0.2,  // Sync is third priority (20%)
		StakeWeight:        0.1,  // Stake provides small boost (10%)
		MinSelectionChance: 0.10, // 10% minimum chance to ensure fairness
		Strategy:           StrategyBalanced,
	}
}

// NewWeightedSelector creates a new WeightedSelector with the given configuration
func NewWeightedSelector(config WeightedSelectorConfig) *WeightedSelector {
	// Validate and normalize weights
	totalWeight := config.AvailabilityWeight + config.LatencyWeight + config.SyncWeight + config.StakeWeight
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
		availabilityWeight: config.AvailabilityWeight,
		latencyWeight:      config.LatencyWeight,
		syncWeight:         config.SyncWeight,
		stakeWeight:        config.StakeWeight,
		minSelectionChance: config.MinSelectionChance,
		strategy:           config.Strategy,
		rng:                globalRandomizer{},
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
	// Extract individual scores from QoS report (already normalized 0-1)
	availability, err := qos.Availability.Float64()
	if err != nil {
		utils.LavaFormatWarning("could not parse availability score, using 0", err)
		availability = 0
	}

	latency, err := qos.Latency.Float64()
	if err != nil {
		utils.LavaFormatWarning("could not parse latency score, using worst latency", err)
		latency = 1.0 // Worst latency
	}

	sync, err := qos.Sync.Float64()
	if err != nil {
		utils.LavaFormatWarning("could not parse sync score, using worst sync", err)
		sync = 1.0 // Worst sync
	}

	// Normalize individual metrics to 0-1 range where higher is better
	availabilityScore := availability // Already 0-1, higher is better

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

// normalizeLatency converts latency score to 0-1 range where higher is better
// Input: latency in seconds (lower is better)
// Output: normalized score where 1.0 = best, 0.0 = worst
func (ws *WeightedSelector) normalizeLatency(latency float64) float64 {
	// Maximum expected latency (1 second)
	const maxLatency = 1.0

	if latency <= 0 {
		return 1.0 // Perfect latency
	}

	// Normalize to 0-1 range, inverted so lower latency = higher score
	normalized := 1.0 - (latency / maxLatency)
	if normalized < 0 {
		normalized = 0 // Clamp to 0 for extremely high latency
	}

	return normalized
}

// normalizeSync converts sync lag to 0-1 range where higher is better
// Input: sync lag in seconds (lower is better)
// Output: normalized score where 1.0 = best, 0.0 = worst
func (ws *WeightedSelector) normalizeSync(syncLag float64) float64 {
	// Maximum acceptable sync lag (60 seconds)
	const maxSyncLag = 60.0

	if syncLag <= 0 {
		return 1.0 // Perfect sync
	}

	// Normalize to 0-1 range, inverted so lower sync = higher score
	normalized := 1.0 - (syncLag / maxSyncLag)
	if normalized < 0 {
		normalized = 0 // Clamp to 0 for extremely poor sync
	}

	return normalized
}

// normalizeStake converts stake to 0-1 range relative to total stake
func (ws *WeightedSelector) normalizeStake(stake sdk.Coin, totalStake sdk.Coin) float64 {
	if totalStake.IsZero() || stake.IsZero() {
		return 0.0
	}

	// Calculate stake ratio
	// Use QuoInt64 to get a decimal representation, then convert to float
	stakeFloat := float64(stake.Amount.Int64())
	totalStakeFloat := float64(totalStake.Amount.Int64())

	if totalStakeFloat == 0 {
		return 0.0
	}

	stakeRatio := stakeFloat / totalStakeFloat

	// Cap at 1.0 to prevent single large staker from dominating
	if stakeRatio > 1.0 {
		stakeRatio = 1.0
	}

	return stakeRatio
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
	if len(providerScores) == 0 {
		return ""
	}

	// Handle single provider case
	if len(providerScores) == 1 {
		return providerScores[0].Address
	}

	// Calculate total weighted score
	totalScore := 0.0
	for _, ps := range providerScores {
		totalScore += ps.SelectionWeight
	}

	if totalScore <= 0 {
		// Fallback to uniform random selection if all scores are zero
		utils.LavaFormatWarning("all provider scores are zero, using uniform selection", nil)
		return providerScores[ws.rng.Intn(len(providerScores))].Address
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
			return ps.Address
		}
	}

	// Fallback to last provider (should rarely happen due to floating point precision)
	utils.LavaFormatWarning("weighted selection fallback to last provider", nil,
		utils.LogAttr("totalScore", totalScore),
		utils.LogAttr("randomValue", randomValue),
	)
	return providerScores[len(providerScores)-1].Address
}

// CalculateProviderScores computes scores for all providers
func (ws *WeightedSelector) CalculateProviderScores(
	allAddresses []string,
	ignoredProviders map[string]struct{},
	providerDataGetter func(string) (*pairingtypes.QualityOfServiceReport, time.Time, bool),
	stakeGetter func(string) int64,
) ([]ProviderScore, map[string]*metrics.OptimizerQoSReport) {
	providerScores := make([]ProviderScore, 0, len(allAddresses))
	qosReports := make(map[string]*metrics.OptimizerQoSReport)

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

		// Calculate composite score
		compositeScore := ws.CalculateScore(qos, stakeCoin, totalStakeCoin)

		providerScore := ProviderScore{
			Address:         providerAddress,
			CompositeScore:  compositeScore,
			SelectionWeight: compositeScore,
		}
		providerScores = append(providerScores, providerScore)

		// Create QoS report for metrics
		latency, sync, availability := qos.GetScoresFloat64()
		qosReports[providerAddress] = &metrics.OptimizerQoSReport{
			ProviderAddress:   providerAddress,
			SyncScore:         sync,
			AvailabilityScore: availability,
			LatencyScore:      latency,
			GenericScore:      compositeScore,
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

	return providerScores, qosReports
}

// GetConfig returns the current configuration
func (ws *WeightedSelector) GetConfig() WeightedSelectorConfig {
	return WeightedSelectorConfig{
		AvailabilityWeight: ws.availabilityWeight,
		LatencyWeight:      ws.latencyWeight,
		SyncWeight:         ws.syncWeight,
		StakeWeight:        ws.stakeWeight,
		MinSelectionChance: ws.minSelectionChance,
		Strategy:           ws.strategy,
	}
}

// UpdateStrategy changes the strategy and recalculates any strategy-dependent parameters
func (ws *WeightedSelector) UpdateStrategy(strategy Strategy) {
	ws.strategy = strategy
}
