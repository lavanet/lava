package rpcconsumer

import "sync"

// On node errors we try to send a relay again.
// If this relay failed all retries we ban it from retries to avoid spam and save resources
type RelayRetriesManager struct {
	hashMap map[string]struct{} // a map of relay requests that failed retries.
	lock    sync.RWMutex
}

func NewRelayRetriesManager() *RelayRetriesManager {
	return &RelayRetriesManager{
		hashMap: make(map[string]struct{}),
	}
}

// Check if we already have this hash so we don't retry.
func (rrm *RelayRetriesManager) CheckHashInMap(hash string) bool {
	rrm.lock.RLock()
	defer rrm.lock.RUnlock()
	_, found := rrm.hashMap[hash]
	return found
}

// Remove hash from map.
func (rrm *RelayRetriesManager) AddHashToMap(hash string) {
	rrm.lock.Lock()
	defer rrm.lock.Unlock()
	rrm.hashMap[hash] = struct{}{}
}

// Add failed retries attempt.
func (rrm *RelayRetriesManager) RemoveHashFromMap(hash string) {
	rrm.lock.Lock()
	defer rrm.lock.Unlock()
	delete(rrm.hashMap, hash)
}

// On epoch updates we reset the state of the hashes.
func (rrm *RelayRetriesManager) UpdateEpoch(epoch uint64) {
	rrm.lock.Lock()
	defer rrm.lock.Unlock()
	rrm.hashMap = make(map[string]struct{})
}
