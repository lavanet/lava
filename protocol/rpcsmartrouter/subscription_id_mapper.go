package rpcsmartrouter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
)

// SubscriptionIDMapper manages bidirectional mapping between router-generated
// subscription IDs and upstream (node) subscription IDs.
//
// The router generates stable IDs for clients (format: "rs_{hash}_{counter}"),
// while upstream nodes return their own IDs. This mapper maintains the translation.
//
// Key invariants:
//   - Router IDs are unique per client (counter ensures this)
//   - Upstream IDs never exposed to clients
//   - Multiple router IDs can map to same upstream ID (subscription sharing)
//   - All operations are thread-safe
type SubscriptionIDMapper struct {
	// routerToUpstream maps routerID -> upstreamID (for unsubscribe translation)
	routerToUpstream map[string]string

	// upstreamToRouters maps upstreamID -> []routerID (for event fanout)
	// One upstream subscription can serve multiple router subscriptions (dedup)
	upstreamToRouters map[string][]string

	// clientCounters tracks per-client monotonic counters for ID generation
	clientCounters map[string]*atomic.Uint64

	lock sync.RWMutex
}

// NewSubscriptionIDMapper creates a new subscription ID mapper
func NewSubscriptionIDMapper() *SubscriptionIDMapper {
	return &SubscriptionIDMapper{
		routerToUpstream:  make(map[string]string),
		upstreamToRouters: make(map[string][]string),
		clientCounters:    make(map[string]*atomic.Uint64),
	}
}

// GenerateRouterID creates a new router subscription ID for a client.
// Format: "rs_{clientHash6}_{counter5}" - e.g., "rs_a1b2c3_00042"
// The ID is unique per client and monotonically increasing.
func (m *SubscriptionIDMapper) GenerateRouterID(clientKey string) string {
	m.lock.Lock()
	defer m.lock.Unlock()

	counter, exists := m.clientCounters[clientKey]
	if !exists {
		counter = &atomic.Uint64{}
		m.clientCounters[clientKey] = counter
	}

	// Format: rs_{first 6 chars of client hash}_{zero-padded counter}
	clientHash := sha256Short(clientKey)
	count := counter.Add(1)

	return fmt.Sprintf("rs_%s_%05d", clientHash, count)
}

// RegisterMapping creates the bidirectional mapping between router and upstream IDs.
// Call this after receiving the upstream subscription confirmation.
func (m *SubscriptionIDMapper) RegisterMapping(routerID, upstreamID string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.routerToUpstream[routerID] = upstreamID
	m.upstreamToRouters[upstreamID] = append(m.upstreamToRouters[upstreamID], routerID)
}

// GetUpstreamID translates a router ID to its upstream ID (for unsubscribe).
// Returns the upstream ID and true if found, or ("", false) if not found.
func (m *SubscriptionIDMapper) GetUpstreamID(routerID string) (string, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	upstreamID, exists := m.routerToUpstream[routerID]
	return upstreamID, exists
}

// GetRouterIDs returns all router IDs mapped to an upstream ID (for event fanout).
// Returns a copy to avoid race conditions.
func (m *SubscriptionIDMapper) GetRouterIDs(upstreamID string) []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	ids := m.upstreamToRouters[upstreamID]
	if ids == nil {
		return []string{}
	}

	// Return copy to avoid race
	result := make([]string, len(ids))
	copy(result, ids)
	return result
}

// RemoveMapping removes a router ID mapping.
// Returns the upstream ID and whether this was the last client for that upstream.
// If lastClient is true, the caller should unsubscribe from upstream.
func (m *SubscriptionIDMapper) RemoveMapping(routerID string) (upstreamID string, lastClient bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	upstreamID, exists := m.routerToUpstream[routerID]
	if !exists {
		return "", false
	}

	// Remove from router→upstream mapping
	delete(m.routerToUpstream, routerID)

	// Remove from upstream→router mapping
	routers := m.upstreamToRouters[upstreamID]
	for i, rid := range routers {
		if rid == routerID {
			// Remove element at index i
			m.upstreamToRouters[upstreamID] = append(routers[:i], routers[i+1:]...)
			break
		}
	}

	// Check if this was the last client for this upstream subscription
	lastClient = len(m.upstreamToRouters[upstreamID]) == 0
	if lastClient {
		delete(m.upstreamToRouters, upstreamID)
	}

	return upstreamID, lastClient
}

// RemoveAllForUpstream removes all router mappings for an upstream ID.
// Call this when an upstream connection is lost or the subscription is terminated.
// Returns the list of removed router IDs.
func (m *SubscriptionIDMapper) RemoveAllForUpstream(upstreamID string) []string {
	m.lock.Lock()
	defer m.lock.Unlock()

	routerIDs := m.upstreamToRouters[upstreamID]
	if routerIDs == nil {
		return []string{}
	}

	// Remove all router→upstream mappings
	for _, routerID := range routerIDs {
		delete(m.routerToUpstream, routerID)
	}

	// Remove the upstream→router mapping
	delete(m.upstreamToRouters, upstreamID)

	// Return copy
	result := make([]string, len(routerIDs))
	copy(result, routerIDs)
	return result
}

// HasRouterID checks if a router ID exists in the mapper.
func (m *SubscriptionIDMapper) HasRouterID(routerID string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, exists := m.routerToUpstream[routerID]
	return exists
}

// HasUpstreamID checks if an upstream ID has any router mappings.
func (m *SubscriptionIDMapper) HasUpstreamID(upstreamID string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, exists := m.upstreamToRouters[upstreamID]
	return exists
}

// Stats returns current mapping statistics for monitoring.
func (m *SubscriptionIDMapper) Stats() (routerCount, upstreamCount int) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.routerToUpstream), len(m.upstreamToRouters)
}

// sha256Short returns the first 6 characters of the SHA-256 hash of input.
func sha256Short(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:6]
}
