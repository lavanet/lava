package rpcconsumer

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/v2/utils"
)

// entries ttl duration
const RetryEntryTTL = 6 * time.Hour

// On node errors we try to send a relay again.
// If this relay failed all retries we ban it from retries to avoid spam and save resources
type RelayRetriesManager struct {
	cache *ristretto.Cache
}

func NewRelayRetriesManager() *RelayRetriesManager {
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for consumer consistency", err)
	}
	return &RelayRetriesManager{
		cache: cache,
	}
}

// Check if we already have this hash so we don't retry.
func (rrm *RelayRetriesManager) CheckHashInCache(hash string) bool {
	_, found := rrm.cache.Get(hash)
	return found
}

// Add hash to the retry cache.
func (rrm *RelayRetriesManager) AddHashToCache(hash string) {
	rrm.cache.SetWithTTL(hash, struct{}{}, 1, RetryEntryTTL)
}

// Remove hash from cache if it exists
func (rrm *RelayRetriesManager) RemoveHashFromCache(hash string) {
	rrm.cache.Del(hash)
}
