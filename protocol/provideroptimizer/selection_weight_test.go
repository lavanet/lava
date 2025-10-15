package provideroptimizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSelectionWeighter(t *testing.T) {
	sw := NewSelectionWeighter()
	assert.NotNil(t, sw)
}

func TestWeight(t *testing.T) {
	sw := NewSelectionWeighter()
	weights := map[string]int64{
		"address1": 10,
		"address2": 20,
	}
	sw.SetWeights(weights)

	assert.Equal(t, int64(10), sw.Weight("address1"))
	assert.Equal(t, int64(20), sw.Weight("address2"))
	assert.Equal(t, int64(1), sw.Weight("address3")) // address not set

	weights = map[string]int64{
		"address1": 25,
		"address3": 30,
	}
	sw.SetWeights(weights)

	assert.Equal(t, int64(25), sw.Weight("address1"))
	assert.Equal(t, int64(20), sw.Weight("address2"))
	assert.Equal(t, int64(30), sw.Weight("address3")) // address not set
}

// Removed: TestWeightedChoice
// WeightedChoice method was part of tier-based selection and has been removed
// Weighted selection is now handled by WeightedSelector in weighted_selector.go
