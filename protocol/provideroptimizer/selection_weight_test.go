package provideroptimizer

import (
	"testing"

	"github.com/lavanet/lava/v3/utils/rand"
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

func TestWeightedChoice(t *testing.T) {
	sw := NewSelectionWeighter()
	rand.InitRandomSeed()
	weights := map[string]int64{
		"address1": 10,
		"address2": 20,
		"address3": 30,
	}
	sw.SetWeights(weights)

	// Create entries based on weights
	entries := []Entry{
		{Address: "address1", Part: 1},
		{Address: "address2", Part: 1},
		{Address: "address3", Part: 1},
	}

	// Run the weighted choice multiple times to check distribution
	results := make(map[string]int)
	for i := 0; i < 1000; i++ {
		choice := sw.WeightedChoice(entries)
		results[choice]++
	}

	// Check that each address was chosen at least once
	assert.Greater(t, results["address1"], 0)
	assert.Greater(t, results["address2"], 0)
	assert.Greater(t, results["address3"], 0)

	weights = map[string]int64{
		"address1": 800,
		"address2": 100,
		"address3": 100,
	}
	sw.SetWeights(weights)
	results = make(map[string]int)
	for i := 0; i < 10000; i++ {
		choice := sw.WeightedChoice(entries)
		results[choice]++
	}
	// Check that address1 is chosen most of the time
	assert.Greater(t, results["address1"], 7000)
	assert.InDelta(t, 1000, results["address2"], 400)
	assert.InDelta(t, 1000, results["address3"], 400)

	weights = map[string]int64{
		"address1": 100,
		"address2": 800,
		"address3": 100,
	}
	sw.SetWeights(weights)
	results = make(map[string]int)
	for i := 0; i < 10000; i++ {
		choice := sw.WeightedChoice(entries)
		results[choice]++
	}
	// Check that address1 is chosen most of the time
	assert.Greater(t, results["address2"], 7000)
	assert.InDelta(t, 1000, results["address1"], 400)
	assert.InDelta(t, 1000, results["address3"], 400)
}
