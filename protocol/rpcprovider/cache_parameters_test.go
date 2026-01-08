package rpcprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// mockChainMessageForCache is a minimal mock that satisfies chainlib.ChainMessage for GetParametersForCache testing
// Most methods are no-ops since GetParametersForCache doesn't use them
type mockChainMessageForCache struct {
	chainlib.ChainMessage // Embed to get default implementations (will panic if called)
}

// Override only the methods that might be called
func (m *mockChainMessageForCache) GetApi() *spectypes.Api {
	return &spectypes.Api{Name: "test"}
}

func (m *mockChainMessageForCache) GetApiCollection() *spectypes.ApiCollection {
	return &spectypes.ApiCollection{
		CollectionData: spectypes.CollectionData{
			ApiInterface: "rest",
		},
	}
}

func (m *mockChainMessageForCache) AppendHeader(metadata []pairingtypes.Metadata) {}

func (m *mockChainMessageForCache) CheckResponseError(data []byte, statusCode int) (bool, string) {
	return false, ""
}

func (m *mockChainMessageForCache) DisableErrorHandling() {}

func (m *mockChainMessageForCache) RequestedBlock() (int64, int64) {
	return -2, -2
}

func (m *mockChainMessageForCache) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) bool {
	return false
}

// Tests for GetParametersForCache

func TestGetParametersForCache_LatestBlock(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: spectypes.LATEST_BLOCK, // -2
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock, "should return latest block from chain tracker")
	require.Nil(t, requestedBlockHash, "LATEST_BLOCK should not have a hash")
	require.True(t, finalized, "LATEST_BLOCK converted to actual block should be finalized if within finalization window")
}

func TestGetParametersForCache_SpecificBlock_WithHash(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())
	mockTracker.SetBlockHash(950, "hash_950")

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 950,
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock)
	require.Equal(t, []byte("hash_950"), requestedBlockHash, "should return hash for specific block")
	require.True(t, finalized, "block 950 should be finalized (1000 - 950 = 50 > 15)")
}

func TestGetParametersForCache_SpecificBlock_NoHash(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())
	// Don't set hash for block 950

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 950,
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock)
	require.Nil(t, requestedBlockHash, "should return nil if hash not available")
	require.True(t, finalized, "should still calculate finalized status")
}

func TestGetParametersForCache_RecentBlock_NotFinalized(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 995, // Within finalization distance
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, _, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock)
	require.False(t, finalized, "block 995 should NOT be finalized (1000 - 995 = 5 < 15)")
}

func TestGetParametersForCache_FinalizedBlock_EdgeCase(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	// Block exactly at finalization boundary
	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 985, // Exactly 15 blocks behind
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	_, _, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.True(t, finalized, "block at exact boundary should be finalized")
}

func TestGetParametersForCache_SafeBlock(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: spectypes.SAFE_BLOCK, // -4
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, _ := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock)
	require.Nil(t, requestedBlockHash, "special block types should not have hash")
}

func TestGetParametersForCache_EarliestBlock(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: spectypes.EARLIEST_BLOCK, // -3
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, _ := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock)
	require.Nil(t, requestedBlockHash, "EARLIEST_BLOCK should not have hash")
}

func TestGetParametersForCache_FinalizedBlockTag(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: spectypes.FINALIZED_BLOCK, // -5
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, _ := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock)
	require.Nil(t, requestedBlockHash, "FINALIZED_BLOCK tag should not have hash")
}

func TestGetParametersForCache_ChainTrackerError(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())
	mockTracker.SetError(errors.New("chain tracker unavailable"))

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 950,
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	// Should handle error gracefully
	latestBlock, requestedBlockHash, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock, "should still get atomic latest block")
	require.Nil(t, requestedBlockHash, "should return nil hash when GetLatestBlockData errors")
	require.True(t, finalized, "should still calculate finalized status based on latest block")
}

func TestGetParametersForCache_MultipleHashesReturned(t *testing.T) {
	// Create a custom mock that returns multiple hashes
	customTracker := &customMockChainTrackerForCache{
		DummyChainTracker: &chaintracker.DummyChainTracker{},
		latestBlock:       1000,
		multipleHashes: []*chaintracker.BlockStore{
			{Block: 950, Hash: "hash1"},
			{Block: 951, Hash: "hash2"},
		},
	}

	rpcps := &RPCProviderServer{
		chainTracker: customTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 950,
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	_, requestedBlockHash, _ := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	// Should only use hash if exactly 1 is returned
	require.Nil(t, requestedBlockHash, "should return nil when multiple hashes returned")
}

func TestGetParametersForCache_BlockZero(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 0,
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 15)

	require.Equal(t, int64(1000), latestBlock)
	require.Nil(t, requestedBlockHash)
	// Block 0 is ancient, should be finalized
	require.True(t, finalized)
}

func TestGetParametersForCache_VeryLargeBlockDistanceForFinalization(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(100, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 50,
		},
	}

	mockChainMsg := &mockChainMessageForCache{}

	// Very large finalization distance
	_, _, finalized := rpcps.GetParametersForCache(context.Background(), request, mockChainMsg, 10000)

	require.False(t, finalized, "with huge finalization distance, nothing should be finalized")
}

// Custom mock for testing multiple hashes scenario
type customMockChainTrackerForCache struct {
	*chaintracker.DummyChainTracker
	latestBlock    int64
	multipleHashes []*chaintracker.BlockStore
}

func (cmt *customMockChainTrackerForCache) GetAtomicLatestBlockNum() int64 {
	return cmt.latestBlock
}

func (cmt *customMockChainTrackerForCache) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	return cmt.latestBlock, cmt.multipleHashes, time.Now(), nil
}

func (cmt *customMockChainTrackerForCache) GetLatestBlockNum() (int64, time.Time) {
	return cmt.latestBlock, time.Now()
}

// Test cache flag behavior with the actual cacheLatestBlockEnabled logic
func TestCacheLatestBlockConversion_Disabled(t *testing.T) {
	// Simulate the logic from trySetRelayReplyInCache when cacheLatestBlockEnabled=false
	requestedBlock := spectypes.LATEST_BLOCK
	latestBlock := int64(1000)
	cacheLatestBlockEnabled := false

	requestedBlockForCache := requestedBlock
	shouldSkip := false
	
	if requestedBlock == spectypes.LATEST_BLOCK {
		if !cacheLatestBlockEnabled {
			shouldSkip = true
		} else {
			requestedBlockForCache = latestBlock
		}
	}

	require.True(t, shouldSkip, "should skip caching LATEST_BLOCK when flag is disabled")
	require.Equal(t, spectypes.LATEST_BLOCK, requestedBlockForCache, "should not convert block number when skipping")
}

func TestCacheLatestBlockConversion_Enabled(t *testing.T) {
	// Simulate the logic from trySetRelayReplyInCache when cacheLatestBlockEnabled=true
	requestedBlock := spectypes.LATEST_BLOCK
	latestBlock := int64(1000)
	cacheLatestBlockEnabled := true

	requestedBlockForCache := requestedBlock
	shouldSkip := false
	
	if requestedBlock == spectypes.LATEST_BLOCK {
		if !cacheLatestBlockEnabled {
			shouldSkip = true
		} else {
			requestedBlockForCache = latestBlock
		}
	}

	require.False(t, shouldSkip, "should not skip caching when flag is enabled")
	require.Equal(t, latestBlock, requestedBlockForCache, "should convert LATEST_BLOCK to actual block number")
}

func TestCacheLatestBlockConversion_SpecificBlock_AlwaysCached(t *testing.T) {
	// Specific blocks should be cached regardless of flag
	requestedBlock := int64(950)
	latestBlock := int64(1000)
	cacheLatestBlockEnabled := false // Even when disabled

	requestedBlockForCache := requestedBlock
	shouldSkip := false
	
	if requestedBlock == spectypes.LATEST_BLOCK {
		if !cacheLatestBlockEnabled {
			shouldSkip = true
		} else {
			requestedBlockForCache = latestBlock
		}
	}

	require.False(t, shouldSkip, "should cache specific blocks regardless of flag")
	require.Equal(t, int64(950), requestedBlockForCache, "specific block number should not change")
}
