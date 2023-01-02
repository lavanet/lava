package chaintracker_test

import (
	"context"
	fmt "fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	chaintracker "github.com/lavanet/lava/relayer/chainTracker"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type MockChainFetcher struct {
	latestBlock int64
	blockHashes []*chaintracker.BlockStore
	mutex       sync.Mutex
	fork        string
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
	return "stubHash-" + strconv.FormatInt(latestBlock, 10) + mcf.fork
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

func (mcf *MockChainFetcher) Fork(fork string) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	if mcf.fork == fork {
		// nothing to do
		return
	}
	mcf.fork = fork
	for _, blockStore := range mcf.blockHashes {
		blockStore.Hash = mcf.hashKey(blockStore.Block)
	}
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
			utils.LavaFormatInfo("started test "+tt.name, nil)
			mockChainFetcher := NewMockChainFetcher(1000, tt.mockBlocks)
			currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()
			timeForPollingMock := time.Millisecond * 2

			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(tt.fetcherBlocks), AverageBlockTime: timeForPollingMock - time.Microsecond, ServerBlockMemory: uint64(tt.mockBlocks)}
			chainTracker, err := chaintracker.New(context.Background(), mockChainFetcher, chainTrackerConfig)
			require.NoError(t, err)

			for _, advancement := range tt.advancements {
				for i := 0; i < int(advancement); i++ {
					currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
				}
				time.Sleep(timeForPollingMock * 2) // statTracker polls asynchronously
				latestBlock := chainTracker.GetLatestBlockNum()
				require.Equal(t, currentLatestBlockInMock, latestBlock)

				latestBlock, requestedHashes, err := chainTracker.GetLatestBlockData(tt.requestBlockFrom, tt.requestBlockTo, tt.specificBlock)
				require.GreaterOrEqual(t, latestBlock, int64(0))
				require.Equal(t, currentLatestBlockInMock, latestBlock)
				require.NoError(t, err)
				require.Equal(t, tt.requestBlocks, int64(len(requestedHashes)))
				if tt.requestBlockFrom != spectypes.NOT_APPLICABLE {
					fromNum := chaintracker.LatestArgToBlockNum(tt.requestBlockFrom, latestBlock)
					require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[0].Hash, fromNum), "incompatible hash %s on block %d", requestedHashes[0].Hash, fromNum)
				}
				if tt.specificBlock != spectypes.NOT_APPLICABLE {
					specificNum := chaintracker.LatestArgToBlockNum(tt.specificBlock, latestBlock)
					// in this test specific hash is always latest and always last in the requested blocks
					require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[len(requestedHashes)-1].Hash, specificNum))
				}
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
		{name: "ten block memory + 3 block fetch", mockBlocks: 100, requestBlocks: 3, fetcherBlocks: 10, advancements: []int64{0, 1, 0, 0, 1, 1, 1, 0, 2, 0, 5, 1, 10, 1, 1, 1}, requestBlockFrom: spectypes.LATEST_BLOCK - 6, requestBlockTo: spectypes.LATEST_BLOCK - 3, specificBlock: spectypes.NOT_APPLICABLE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChainFetcher := NewMockChainFetcher(1000, tt.mockBlocks)
			currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()
			timeForPollingMock := time.Millisecond * 2

			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(tt.fetcherBlocks), AverageBlockTime: timeForPollingMock - time.Microsecond, ServerBlockMemory: uint64(tt.mockBlocks)}
			chainTracker, err := chaintracker.New(context.Background(), mockChainFetcher, chainTrackerConfig)
			require.NoError(t, err)

			for _, advancement := range tt.advancements {
				for i := 0; i < int(advancement); i++ {
					currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
				}
				time.Sleep(timeForPollingMock * 2) // statTracker polls asynchronously
				latestBlock := chainTracker.GetLatestBlockNum()
				require.Equal(t, currentLatestBlockInMock, latestBlock)

				latestBlock, requestedHashes, err := chainTracker.GetLatestBlockData(tt.requestBlockFrom, tt.requestBlockTo, tt.specificBlock)
				require.Equal(t, currentLatestBlockInMock, latestBlock)
				require.NoError(t, err)
				require.Equal(t, tt.requestBlocks, int64(len(requestedHashes)))
				if tt.requestBlockFrom != spectypes.NOT_APPLICABLE {
					fromNum := chaintracker.LatestArgToBlockNum(tt.requestBlockFrom, latestBlock)
					require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[0].Hash, fromNum), "incompatible hash %s on block %d", requestedHashes[0].Hash, fromNum)
				}
				for idx := 0; idx < len(requestedHashes)-1; idx++ {
					require.Equal(t, requestedHashes[idx].Block+1, requestedHashes[idx+1].Block)
					require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[idx].Hash, requestedHashes[idx].Block))
				}
			}
		})
	}
}

func TestChainTrackerCallbacks(t *testing.T) {
	mockBlocks := int64(100)
	requestBlocks := 3
	fetcherBlocks := 10
	requestBlockFrom := spectypes.LATEST_BLOCK - 6
	requestBlockTo := spectypes.LATEST_BLOCK - 3
	specificBlock := spectypes.NOT_APPLICABLE
	tests := []struct {
		name        string
		advancement int64
		fork        string
		shouldFork  bool
	}{
		{name: "[t00]", advancement: 0, shouldFork: false, fork: ""},
		{name: "[t01]", advancement: 1, shouldFork: false, fork: ""},
		{name: "[t02]", advancement: 0, shouldFork: true, fork: "fork"},
		{name: "[t03]", advancement: 0, shouldFork: false, fork: "fork"},
		{name: "[t04]", advancement: 1, shouldFork: true, fork: "another-fork"},
		{name: "[t05]", advancement: 1, shouldFork: false, fork: "another-fork"},
		{name: "[t06]", advancement: 1, shouldFork: true, fork: "fork"},
		{name: "[t07]", advancement: 0, shouldFork: false, fork: "fork"},
		{name: "[t08]", advancement: 2, shouldFork: true, fork: ""},
		{name: "[t09]", advancement: 0, shouldFork: false, fork: ""},
		{name: "[t10]", advancement: 5, shouldFork: true, fork: "another-fork"},
		{name: "[t11]", advancement: 1, shouldFork: false, fork: "another-fork"},
		{name: "[t12]", advancement: 10, shouldFork: true, fork: ""},
		{name: "[t13]", advancement: 15, shouldFork: false, fork: ""},
		{name: "[t14]", advancement: 1, shouldFork: true, fork: "fork"},
		{name: "[t15]", advancement: 1, shouldFork: false, fork: "fork"},
		{name: "[t16]", advancement: 0, shouldFork: true, fork: "another-fork"},
	}
	mockChainFetcher := NewMockChainFetcher(1000, mockBlocks)
	currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()
	timeForPollingMock := time.Millisecond * 2

	//used to identify if the fork callback was called
	callbackCalledFork := false
	forkCallback := func() {
		utils.LavaFormatDebug("fork callback called", nil)
		callbackCalledFork = true
	}
	//used to identify if the newLatest callback was called
	callbackCalledNewLatest := false
	newBlockCallback := func() {
		utils.LavaFormatDebug("new latest callback called", nil)
		callbackCalledNewLatest = true
	}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: timeForPollingMock - time.Microsecond, ServerBlockMemory: uint64(mockBlocks), ForkCallback: forkCallback, NewLatestCallback: newBlockCallback}
	chainTracker, err := chaintracker.New(context.Background(), mockChainFetcher, chainTrackerConfig)
	require.NoError(t, err)
	t.Run("banana", func(t *testing.T) {
		for _, tt := range tests {
			utils.LavaFormatInfo("started test "+tt.name, nil)
			callbackCalledFork = false
			callbackCalledNewLatest = false
			for i := 0; i < int(tt.advancement); i++ {
				currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
			}
			mockChainFetcher.Fork(tt.fork)
			time.Sleep(timeForPollingMock * 2) // statTracker polls asynchronously
			latestBlock := chainTracker.GetLatestBlockNum()
			require.Equal(t, currentLatestBlockInMock, latestBlock)

			latestBlock, requestedHashes, err := chainTracker.GetLatestBlockData(requestBlockFrom, requestBlockTo, specificBlock)
			require.Equal(t, currentLatestBlockInMock, latestBlock)
			require.NoError(t, err)
			require.Equal(t, requestBlocks, len(requestedHashes))
			if requestBlockFrom != spectypes.NOT_APPLICABLE {
				fromNum := chaintracker.LatestArgToBlockNum(requestBlockFrom, latestBlock)
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[0].Hash, fromNum), "incompatible hash %s on block %d", requestedHashes[0].Hash, fromNum)
			}
			for idx := 0; idx < len(requestedHashes)-1; idx++ {
				require.Equal(t, requestedHashes[idx].Block+1, requestedHashes[idx+1].Block)
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[idx].Hash, requestedHashes[idx].Block))
			}
			if tt.shouldFork {
				require.True(t, callbackCalledFork)
			} else {
				require.False(t, callbackCalledFork)
			}
			if tt.advancement > 0 {
				require.True(t, callbackCalledNewLatest)
			} else {
				require.False(t, callbackCalledNewLatest)
			}
		}
	})
}
