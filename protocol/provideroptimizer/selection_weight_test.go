package provideroptimizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProviderStakeCache(t *testing.T) {
	cache := NewProviderStakeCache()
	assert.NotNil(t, cache)
}

func TestGetStake(t *testing.T) {
	cache := NewProviderStakeCache()
	stakes := map[string]int64{
		"address1": 10,
		"address2": 20,
	}
	cache.UpdateStakes(stakes)

	assert.Equal(t, int64(10), cache.GetStake("address1"))
	assert.Equal(t, int64(20), cache.GetStake("address2"))
	assert.Equal(t, int64(1), cache.GetStake("address3")) // address not set, returns default

	// Update stakes
	stakes = map[string]int64{
		"address1": 25,
		"address3": 30,
	}
	cache.UpdateStakes(stakes)

	assert.Equal(t, int64(25), cache.GetStake("address1"))
	assert.Equal(t, int64(20), cache.GetStake("address2")) // not updated, retains previous value
	assert.Equal(t, int64(30), cache.GetStake("address3"))
}

// Note: WeightedChoice method was part of tier-based selection and has been removed.
// Weighted selection is now handled by WeightedSelector in weighted_selector.go
