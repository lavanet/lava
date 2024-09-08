package provideroptimizer

import (
	"sync"

	"github.com/lavanet/lava/v3/utils"
	"github.com/lavanet/lava/v3/utils/rand"
)

// SelectionWeighter is a utility to select an address based on a weight.
type SelectionWeighter interface {
	Weight(address string) int64
	SetWeights(weights map[string]int64)
	WeightedChoice(possibilities []Entry) string
}

type selectionWeighterInst struct {
	lock        sync.RWMutex
	weights     map[string]int64
	totalWeight int64
}

func NewSelectionWeighter() SelectionWeighter {
	return &selectionWeighterInst{
		weights: make(map[string]int64),
	}
}

func (sw *selectionWeighterInst) Weight(address string) int64 {
	sw.lock.RLock()
	defer sw.lock.RUnlock()
	return sw.weights[address]
}

func (sw *selectionWeighterInst) SetWeights(weights map[string]int64) {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	for address, weight := range weights {
		sw.weights[address] = weight
	}
	totalWeight := int64(0)
	for _, weight := range sw.weights {
		totalWeight += weight
	}
	sw.totalWeight = totalWeight
}

func (sw *selectionWeighterInst) WeightedChoice(entries []Entry) string {
	if len(entries) == 0 {
		return ""
	}
	sw.lock.RLock()
	defer sw.lock.RUnlock()
	randWeight := rand.Int63n(sw.totalWeight)
	currentWeight := int64(0)
	for _, entry := range entries {
		currentWeight += sw.Weight(entry.Address)
		if currentWeight > randWeight {
			return entry.Address
		}
	}
	utils.LavaFormatError("invalid weighted choice, no address chosen, fallback to last one", nil, utils.LogAttr("addresses", entries),
		utils.LogAttr("totalWeight", sw.totalWeight))
	// Fallback to the last address if no address is selected
	return entries[len(entries)-1].Address
}
