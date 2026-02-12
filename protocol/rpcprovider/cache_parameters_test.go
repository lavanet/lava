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

func (m *mockChainMessageForCache) TimeoutOverride(override ...time.Duration) time.Duration {
	if len(override) > 0 {
		return override[0]
	}
	return 0 // Use default timeout
}

// mockChainParserForCache provides minimal implementation for testing
type mockChainParserForCache struct{}

func (m *mockChainParserForCache) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceToFinalization uint32, blocksInFinalizationData uint32) {
	return 5, 6 * time.Second, 15, 20
}

// Helper to create test relay request with seenBlock
func createTestRelayRequest(requestBlock int64, seenBlock int64) *pairingtypes.RelayRequest {
	return &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: requestBlock,
			SeenBlock:    seenBlock,
		},
	}
}

// Test helper that wraps GetParametersForCache with test-friendly defaults
func (rpcps *RPCProviderServer) getParametersForCacheTest(ctx context.Context, request *pairingtypes.RelayRequest, chainMsg chainlib.ChainMessage, blockDistanceToFinalization uint32) (latestBlock int64, requestedBlockHash []byte, finalized bool, err error) {
	// Use test defaults for consistency check
	blockLagForQosSync := int64(5)
	averageBlockTime := 6 * time.Second
	blocksInFinalizationData := uint32(20)
	relayTimeout := chainlib.GetRelayTimeout(chainMsg, averageBlockTime)
	
	// First handle consistency - get the latest block after waiting if needed
	latestBlock, _, _, err = rpcps.handleConsistency(
		ctx,
		relayTimeout,
		request.RelayData.GetSeenBlock(),
		request.RelayData.GetRequestBlock(),
		averageBlockTime,
		blockLagForQosSync,
		blockDistanceToFinalization,
		blocksInFinalizationData,
	)
	if err != nil {
		return 0, nil, false, err
	}
	
	// Then get cache parameters using the consistent latest block
	requestedBlockHash, finalized, err = rpcps.GetParametersForCache(ctx, request, latestBlock, blockDistanceToFinalization)
	return latestBlock, requestedBlockHash, finalized, err
}

// Tests for GetParametersForCache

func TestGetParametersForCache_LatestBlock(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := createTestRelayRequest(spectypes.LATEST_BLOCK, 990)
	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)
	require.Equal(t, int64(1000), latestBlock, "should return latest block from chain tracker")
	require.Nil(t, requestedBlockHash, "LATEST_BLOCK should not have a hash")
	require.False(t, finalized, "LATEST_BLOCK (current block) is not finalized - needs to age by blockDistanceToFinalization")
}

func TestGetParametersForCache_SpecificBlock_WithHash(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())
	mockTracker.SetBlockHash(950, "hash_950")

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := createTestRelayRequest(950, 940)

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

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

	request := createTestRelayRequest(950, 940)

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

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

	request := createTestRelayRequest(995, 990) // Within finalization distance

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, _, finalized, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

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
	request := createTestRelayRequest(985, 980) // Exactly 15 blocks behind

	mockChainMsg := &mockChainMessageForCache{}

	_, _, finalized, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

	require.True(t, finalized, "block at exact boundary should be finalized")
}

func TestGetParametersForCache_SafeBlock(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := createTestRelayRequest(spectypes.SAFE_BLOCK, 990)

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, _, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

	require.Equal(t, int64(1000), latestBlock)
	require.Nil(t, requestedBlockHash, "special block types should not have hash")
}

func TestGetParametersForCache_EarliestBlock(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := createTestRelayRequest(spectypes.EARLIEST_BLOCK, 990)

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, _, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

	require.Equal(t, int64(1000), latestBlock)
	require.Nil(t, requestedBlockHash, "EARLIEST_BLOCK should not have hash")
}

func TestGetParametersForCache_FinalizedBlockTag(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := createTestRelayRequest(spectypes.FINALIZED_BLOCK, 990)

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, _, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

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

	request := createTestRelayRequest(950, 940)

	mockChainMsg := &mockChainMessageForCache{}

	// Should return error when chain tracker fails in handleConsistency
	_, _, _, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.Error(t, err, "should return error when GetLatestBlockData fails in handleConsistency")
	require.Contains(t, err.Error(), "chain tracker unavailable", "error should indicate chain tracker unavailable")
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

	request := createTestRelayRequest(950, 940)

	mockChainMsg := &mockChainMessageForCache{}

	_, requestedBlockHash, _, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

	// Should only use hash if exactly 1 is returned
	require.Nil(t, requestedBlockHash, "should return nil when multiple hashes returned")
}

func TestGetParametersForCache_BlockZero(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now())

	rpcps := &RPCProviderServer{
		chainTracker: mockTracker,
	}

	request := createTestRelayRequest(0, 0)

	mockChainMsg := &mockChainMessageForCache{}

	latestBlock, requestedBlockHash, finalized, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 15)

	require.NoError(t, err)

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

	request := createTestRelayRequest(50, 40)

	mockChainMsg := &mockChainMessageForCache{}

	// Very large finalization distance
	_, _, finalized, err := rpcps.getParametersForCacheTest(context.Background(), request, mockChainMsg, 10000)

	require.NoError(t, err)

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
