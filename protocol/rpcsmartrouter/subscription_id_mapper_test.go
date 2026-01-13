package rpcsmartrouter

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionIDMapper_GenerateRouterID(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	clientKey := "dapp123:192.168.1.1:ws-uid-456"
	id1 := mapper.GenerateRouterID(clientKey)
	id2 := mapper.GenerateRouterID(clientKey)
	id3 := mapper.GenerateRouterID(clientKey)

	// Check format: rs_{hash}_{counter}
	assert.True(t, strings.HasPrefix(id1, "rs_"), "should have rs_ prefix")
	assert.Len(t, strings.Split(id1, "_"), 3, "should have 3 parts separated by _")

	// Check uniqueness
	assert.NotEqual(t, id1, id2, "IDs should be unique")
	assert.NotEqual(t, id2, id3, "IDs should be unique")

	// Check monotonic counter
	assert.Contains(t, id1, "_00001", "first ID should have counter 00001")
	assert.Contains(t, id2, "_00002", "second ID should have counter 00002")
	assert.Contains(t, id3, "_00003", "third ID should have counter 00003")
}

func TestSubscriptionIDMapper_GenerateRouterID_DifferentClients(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	client1 := "client1"
	client2 := "client2"

	id1a := mapper.GenerateRouterID(client1)
	id1b := mapper.GenerateRouterID(client1)
	id2a := mapper.GenerateRouterID(client2)
	id2b := mapper.GenerateRouterID(client2)

	// Each client has independent counter
	assert.Contains(t, id1a, "_00001")
	assert.Contains(t, id1b, "_00002")
	assert.Contains(t, id2a, "_00001")
	assert.Contains(t, id2b, "_00002")

	// Different hash prefixes for different clients
	parts1 := strings.Split(id1a, "_")
	parts2 := strings.Split(id2a, "_")
	assert.NotEqual(t, parts1[1], parts2[1], "different clients should have different hash")
}

func TestSubscriptionIDMapper_RegisterAndGetMapping(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	routerID := "rs_abc123_00001"
	upstreamID := "0x1a2b3c4d"

	// Register mapping
	mapper.RegisterMapping(routerID, upstreamID)

	// Get upstream ID
	gotUpstream, found := mapper.GetUpstreamID(routerID)
	require.True(t, found)
	assert.Equal(t, upstreamID, gotUpstream)

	// Get router IDs
	routerIDs := mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 1)
	assert.Equal(t, routerID, routerIDs[0])
}

func TestSubscriptionIDMapper_MultipleRoutersToOneUpstream(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	upstreamID := "0x1a2b3c4d"
	routerID1 := "rs_client1_00001"
	routerID2 := "rs_client2_00001"
	routerID3 := "rs_client3_00001"

	// Multiple router IDs map to same upstream (subscription sharing)
	mapper.RegisterMapping(routerID1, upstreamID)
	mapper.RegisterMapping(routerID2, upstreamID)
	mapper.RegisterMapping(routerID3, upstreamID)

	// All should resolve to same upstream
	up1, _ := mapper.GetUpstreamID(routerID1)
	up2, _ := mapper.GetUpstreamID(routerID2)
	up3, _ := mapper.GetUpstreamID(routerID3)
	assert.Equal(t, upstreamID, up1)
	assert.Equal(t, upstreamID, up2)
	assert.Equal(t, upstreamID, up3)

	// Upstream should have all router IDs
	routerIDs := mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 3)
	assert.Contains(t, routerIDs, routerID1)
	assert.Contains(t, routerIDs, routerID2)
	assert.Contains(t, routerIDs, routerID3)
}

func TestSubscriptionIDMapper_RemoveMapping(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	upstreamID := "0x1a2b3c4d"
	routerID1 := "rs_client1_00001"
	routerID2 := "rs_client2_00001"

	mapper.RegisterMapping(routerID1, upstreamID)
	mapper.RegisterMapping(routerID2, upstreamID)

	// Remove first router - should not be last client
	gotUpstream, lastClient := mapper.RemoveMapping(routerID1)
	assert.Equal(t, upstreamID, gotUpstream)
	assert.False(t, lastClient, "should not be last client")

	// routerID1 should be gone
	_, found := mapper.GetUpstreamID(routerID1)
	assert.False(t, found)

	// routerID2 should still exist
	_, found = mapper.GetUpstreamID(routerID2)
	assert.True(t, found)

	// Upstream should have one router left
	routerIDs := mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 1)
	assert.Equal(t, routerID2, routerIDs[0])

	// Remove second router - should be last client
	gotUpstream, lastClient = mapper.RemoveMapping(routerID2)
	assert.Equal(t, upstreamID, gotUpstream)
	assert.True(t, lastClient, "should be last client")

	// Upstream should have no routers
	routerIDs = mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 0)
}

func TestSubscriptionIDMapper_RemoveMapping_NotFound(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	gotUpstream, lastClient := mapper.RemoveMapping("nonexistent")
	assert.Equal(t, "", gotUpstream)
	assert.False(t, lastClient)
}

func TestSubscriptionIDMapper_RemoveAllForUpstream(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	upstreamID := "0x1a2b3c4d"
	routerID1 := "rs_client1_00001"
	routerID2 := "rs_client2_00001"
	routerID3 := "rs_client3_00001"

	mapper.RegisterMapping(routerID1, upstreamID)
	mapper.RegisterMapping(routerID2, upstreamID)
	mapper.RegisterMapping(routerID3, upstreamID)

	// Remove all for upstream
	removed := mapper.RemoveAllForUpstream(upstreamID)
	assert.Len(t, removed, 3)
	assert.Contains(t, removed, routerID1)
	assert.Contains(t, removed, routerID2)
	assert.Contains(t, removed, routerID3)

	// All should be gone
	_, found := mapper.GetUpstreamID(routerID1)
	assert.False(t, found)
	_, found = mapper.GetUpstreamID(routerID2)
	assert.False(t, found)
	_, found = mapper.GetUpstreamID(routerID3)
	assert.False(t, found)

	// Upstream should have no routers
	routerIDs := mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 0)
}

func TestSubscriptionIDMapper_HasMethods(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	routerID := "rs_abc123_00001"
	upstreamID := "0x1a2b3c4d"

	assert.False(t, mapper.HasRouterID(routerID))
	assert.False(t, mapper.HasUpstreamID(upstreamID))

	mapper.RegisterMapping(routerID, upstreamID)

	assert.True(t, mapper.HasRouterID(routerID))
	assert.True(t, mapper.HasUpstreamID(upstreamID))
}

func TestSubscriptionIDMapper_Stats(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	routerCount, upstreamCount := mapper.Stats()
	assert.Equal(t, 0, routerCount)
	assert.Equal(t, 0, upstreamCount)

	// Add mappings
	mapper.RegisterMapping("rs_1", "up_a")
	mapper.RegisterMapping("rs_2", "up_a") // Same upstream
	mapper.RegisterMapping("rs_3", "up_b") // Different upstream

	routerCount, upstreamCount = mapper.Stats()
	assert.Equal(t, 3, routerCount)
	assert.Equal(t, 2, upstreamCount)
}

func TestSubscriptionIDMapper_ConcurrentAccess(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(clientNum int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				routerID := mapper.GenerateRouterID("client" + string(rune(clientNum)))
				mapper.RegisterMapping(routerID, "upstream_"+routerID)
			}
		}(i)
	}

	wg.Wait()

	// Verify integrity
	routerCount, upstreamCount := mapper.Stats()
	expectedCount := numGoroutines * numOperations
	assert.Equal(t, expectedCount, routerCount)
	assert.Equal(t, expectedCount, upstreamCount) // Each router has unique upstream in this test
}

func TestSubscriptionIDMapper_GetRouterIDs_ReturnsCopy(t *testing.T) {
	mapper := NewSubscriptionIDMapper()

	upstreamID := "0x1a2b3c4d"
	mapper.RegisterMapping("rs_1", upstreamID)
	mapper.RegisterMapping("rs_2", upstreamID)

	// Get router IDs
	ids1 := mapper.GetRouterIDs(upstreamID)

	// Modify the returned slice
	ids1[0] = "modified"

	// Get again - should not be affected
	ids2 := mapper.GetRouterIDs(upstreamID)
	assert.NotEqual(t, "modified", ids2[0], "returned slice should be a copy")
}

func TestSha256Short(t *testing.T) {
	hash1 := sha256Short("test")
	hash2 := sha256Short("test")
	hash3 := sha256Short("different")

	assert.Len(t, hash1, 6, "should return 6 characters")
	assert.Equal(t, hash1, hash2, "same input should produce same hash")
	assert.NotEqual(t, hash1, hash3, "different input should produce different hash")
}
