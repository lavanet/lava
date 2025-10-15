package provideroptimizer

import (
	"sync"
)

// SelectionWeighter is a utility to select an address based on a weight.
type SelectionWeighter interface {
	Weight(address string) int64
	SetWeights(weights map[string]int64)
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
