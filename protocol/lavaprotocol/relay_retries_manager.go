package lavaprotocol

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/utils"
)

// entries ttl duration
const (
	CacheMaxCost     = 10 * 1024 // each item cost would be 1
	CacheNumCounters = 20000     // expect 2000 items
	RetryEntryTTL    = 6 * time.Hour
)

type RelayRetriesManagerInf interface {
	AddHashToCache(hash string)
	CheckHashInCache(hash string) bool
	RemoveHashFromCache(hash string)
}

// On node errors we try to send a relay again.
// If this relay failed all retries we ban it from retries to avoid spam and save resources
type RelayRetriesManager struct {
	cache *ristretto.Cache[string, any]
}

func NewRelayRetriesManager() *RelayRetriesManager {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
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
	// Force cache to process buffered writes for consistency
	rrm.cache.Wait()
}

// Remove hash from cache if it exists
func (rrm *RelayRetriesManager) RemoveHashFromCache(hash string) {
	rrm.cache.Del(hash)
}
