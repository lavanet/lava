package provideroptimizer

import (
	"sync"

	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
)

// SelectionWeighter is a utility to select an address based on a weight.
type SelectionWeighter interface {
	Weight(address string) int64
	SetWeights(weights map[string]int64)
	WeightedChoice(possibilities []Entry) string
	WeightedChoiceByQoS(possibilities []Entry) string
}

type selectionWeighterInst struct {
	lock    sync.RWMutex
	weights map[string]int64
}

func NewSelectionWeighter() SelectionWeighter {
	return &selectionWeighterInst{
		weights: make(map[string]int64),
	}
}

func (sw *selectionWeighterInst) Weight(address string) int64 {
	sw.lock.RLock()
	defer sw.lock.RUnlock()
	return sw.weightInner(address)
}

// assumes lock is held
func (sw *selectionWeighterInst) weightInner(address string) int64 {
	weight, ok := sw.weights[address]
	if !ok {
		// default weight is 1
		return 1
	}
	return weight
}

func (sw *selectionWeighterInst) SetWeights(weights map[string]int64) {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	for address, weight := range weights {
		sw.weights[address] = weight
	}
}

// WeightedChoiceByQoS selects provider based on QoS score using inverse score weighting.
// Lower scores (better QoS) result in higher selection probability.
// This method should be used when QoS-based selection is enabled via the optimizer flag.
func (sw *selectionWeighterInst) WeightedChoiceByQoS(entries []Entry) string {
	if len(entries) == 0 {
		return ""
	}

	const epsilon = 0.0001 // Small value to prevent division by zero

	// Calculate total weighted sum using inverse QoS scores
	totalWeight := float64(0)
	for _, entry := range entries {
		// Inverse score: lower score (better QoS) = higher weight
		weight := (1.0 / (entry.Score + epsilon)) * entry.Part
		totalWeight += weight
	}

	// Generate random number in range [0, totalWeight)
	randWeight := rand.Float64() * totalWeight

	// Find which provider the random number falls into using cumulative distribution
	currentWeight := float64(0)
	for _, entry := range entries {
		weight := (1.0 / (entry.Score + epsilon)) * entry.Part
		currentWeight += weight
		if currentWeight > randWeight {
			utils.LavaFormatInfo("selected provider by QoS",
				utils.LogAttr("address", entry.Address),
				utils.LogAttr("score", entry.Score),
				utils.LogAttr("part", entry.Part))
			return entry.Address
		}
	}

	// Error case: should never reach here, fallback to first provider (best QoS)
	utils.LavaFormatError("invalid QoS weighted choice, no address chosen, fallback to first one", nil,
		utils.LogAttr("addresses", entries),
		utils.LogAttr("total_weight", totalWeight))
	return entries[0].Address // First entry has best QoS (lowest score)
}

// Choose the selected provider address from the entries using stake-weighted random selection,
// the weight is the stake of the provider, the more stake the more likely to be selected
func (sw *selectionWeighterInst) WeightedChoice(entries []Entry) string {
	if len(entries) == 0 {
		return ""
	}
	sw.lock.RLock()
	defer sw.lock.RUnlock()

	// Calculate total weighted sum of all providers (stake * fractional part)
	totalWeight := int64(0)
	for _, entry := range entries {
		totalWeight += int64(float64(sw.weightInner(entry.Address)) * entry.Part)
	}

	// Generate random number in range [0, totalWeight)
	randWeight := rand.Int63n(totalWeight)

	// Find which provider the random number falls into using cumulative distribution
	currentWeight := int64(0)
	for _, entry := range entries {
		currentWeight += int64(float64(sw.weightInner(entry.Address)) * entry.Part)
		if currentWeight > randWeight {
			utils.LavaFormatInfo("selected provider by stake",
				utils.LogAttr("address", entry.Address),
				utils.LogAttr("stake", sw.weightInner(entry.Address)),
				utils.LogAttr("part", entry.Part))
			return entry.Address // return selected provider address
		}
	}

	// Error case: should never reach here, fallback to last provider
	utils.LavaFormatError("invalid weighted choice, no address chosen, fallback to last one", nil, utils.LogAttr("addresses", entries),
		utils.LogAttr("totalWeight", totalWeight))
	// Fallback to the last address if no address is selected
	return entries[len(entries)-1].Address
}
