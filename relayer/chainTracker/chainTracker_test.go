package chaintracker_test

import (
	"context"
	fmt "fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	chaintracker "github.com/lavanet/lava/relayer/chainTracker"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type MockChainFetcher struct {
	latestBlock int64
	blockHashes []*chaintracker.BlockStore
	mutex       sync.Mutex
}

func (mcf *MockChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	return mcf.latestBlock, nil
}
func (mcf *MockChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	for _, blockStore := range mcf.blockHashes {
		if blockStore.Block == blockNum {
			return blockStore.Hash, nil
		}
	}
	return "", fmt.Errorf("invalid block num requested %d, latestBlockSaved: %d, MockChainFetcher blockHashes: %+v", blockNum, mcf.latestBlock, mcf.blockHashes)
}

func (mcf *MockChainFetcher) hashKey(latestBlock int64) string {
	return "stubHash-" + strconv.FormatInt(latestBlock, 10)
}

func (mcf *MockChainFetcher) IsCorrectHash(hash string, hashBlock int64) bool {
	return hash == mcf.hashKey(hashBlock)
}

func (mcf *MockChainFetcher) AdvanceBlock() int64 {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	mcf.latestBlock += 1
	newHash := mcf.hashKey(mcf.latestBlock)
	mcf.blockHashes = append(mcf.blockHashes[1:], &chaintracker.BlockStore{Block: mcf.latestBlock, Hash: newHash})
	return mcf.latestBlock
}
func (mcf *MockChainFetcher) SetBlock(latestBlock int64) {
	mcf.latestBlock = latestBlock
	newHash := mcf.hashKey(mcf.latestBlock)
	mcf.blockHashes = append(mcf.blockHashes, &chaintracker.BlockStore{Block: latestBlock, Hash: newHash})
}

func NewMockChainFetcher(startBlock int64, blocksToSave int64) *MockChainFetcher {
	mockCHainFetcher := MockChainFetcher{}
	for i := int64(0); i < blocksToSave; i++ {
		mockCHainFetcher.SetBlock(startBlock + i)
	}
	return &mockCHainFetcher
}

func TestChainTracker(t *testing.T) {

	tests := []struct {
		name             string
		requestBlocks    int64
		fetcherBlocks    int64
		mockBlocks       int64
		advancements     []int64
		requestBlockFrom int64
		requestBlockTo   int64
		specificBlock    int64
	}{
		{name: "one block memory + fetch", mockBlocks: 20, requestBlocks: 1, fetcherBlocks: 1, advancements: []int64{0, 1, 0, 0, 1, 1, 1, 0, 2, 0, 5, 1, 10, 1, 1, 1}, requestBlockFrom: spectypes.NOT_APPLICABLE, requestBlockTo: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK},
		{name: "ten block memory 4 block fetch", mockBlocks: 20, requestBlocks: 4, fetcherBlocks: 10, advancements: []int64{0, 1, 0, 0, 1, 1, 1, 0, 2, 0, 5, 1, 10, 1, 1, 1}, requestBlockFrom: spectypes.LATEST_BLOCK - 9, requestBlockTo: spectypes.LATEST_BLOCK - 6, specificBlock: spectypes.LATEST_BLOCK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChainFetcher := NewMockChainFetcher(1000, tt.mockBlocks)
			currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()
			timeForPollingMock := time.Millisecond * 5

			waitForPolling := func() {
				time.Sleep(timeForPollingMock)
			}

			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(tt.fetcherBlocks), AverageBlockTime: timeForPollingMock - time.Microsecond, ServerBlockMemory: uint64(tt.mockBlocks)}
			chainTracker, err := chaintracker.New(context.Background(), mockChainFetcher, chainTrackerConfig)
			require.NoError(t, err)

			for _, advancement := range tt.advancements {
				waitForPolling() // statTracker polls asynchronously
				for i := 0; i < int(advancement); i++ {
					currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
				}
				latestBlock := chainTracker.GetLatestBlockNum()
				require.Equal(t, currentLatestBlockInMock, latestBlock)

				latestBlock, requestedHashes, err := chainTracker.GetLatestBlockData(tt.requestBlockFrom, tt.requestBlockTo, tt.specificBlock)
				require.Equal(t, currentLatestBlockInMock, latestBlock)
				require.NoError(t, err)
				require.Equal(t, tt.requestBlocks, len(requestedHashes))
				fromNum := chaintracker.LatestArgToBlockNum(tt.requestBlockFrom, latestBlock)
				specificNum := chaintracker.LatestArgToBlockNum(tt.specificBlock, latestBlock)
				// in this test specific hash is always latest and always last in the requested blocks
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[0].Hash, fromNum))
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[len(requestedHashes)-1].Hash, specificNum))
				for idx := 0; idx < len(requestedHashes)-1; idx++ {
					require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[idx].Hash, requestedHashes[idx].Block))
				}
			}
		})
	}
}

func TestChainTrackerRangeOnly(t *testing.T) {

	tests := []struct {
		name             string
		requestBlocks    int64
		fetcherBlocks    int64
		mockBlocks       int64
		advancements     []int64
		requestBlockFrom int64
		requestBlockTo   int64
		specificBlock    int64
	}{
		{name: "ten block memory + 3 block fetch", mockBlocks: 100, requestBlocks: 4, fetcherBlocks: 10, advancements: []int64{0, 1, 0, 0, 1, 1, 1, 0, 2, 0, 5, 1, 10, 1, 1, 1}, requestBlockFrom: spectypes.LATEST_BLOCK - 6, requestBlockTo: spectypes.LATEST_BLOCK - 3, specificBlock: spectypes.NOT_APPLICABLE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChainFetcher := NewMockChainFetcher(1000, tt.mockBlocks)
			currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()
			timeForPollingMock := time.Millisecond * 5

			waitForPolling := func() {
				time.Sleep(timeForPollingMock)
			}

			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(tt.fetcherBlocks), AverageBlockTime: timeForPollingMock - time.Microsecond, ServerBlockMemory: uint64(tt.mockBlocks)}
			chainTracker, err := chaintracker.New(context.Background(), mockChainFetcher, chainTrackerConfig)
			require.NoError(t, err)

			for _, advancement := range tt.advancements {
				waitForPolling() // statTracker polls asynchronously
				for i := 0; i < int(advancement); i++ {
					currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
				}
				latestBlock := chainTracker.GetLatestBlockNum()
				require.Equal(t, currentLatestBlockInMock, latestBlock)

				latestBlock, requestedHashes, err := chainTracker.GetLatestBlockData(tt.requestBlockFrom, tt.requestBlockTo, tt.specificBlock)
				require.Equal(t, currentLatestBlockInMock, latestBlock)
				require.NoError(t, err)
				require.Equal(t, tt.requestBlocks, len(requestedHashes))
				fromNum := chaintracker.LatestArgToBlockNum(tt.requestBlockFrom, latestBlock)
				// in this test specific hash is always latest and always last in the requested blocks
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[0].Hash, fromNum))
				for idx := 0; idx < len(requestedHashes)-1; idx++ {
					require.Equal(t, requestedHashes[idx].Block+1, requestedHashes[idx+1].Block)
					require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[idx].Hash, requestedHashes[idx].Block))
				}
			}
		})
	}
}
