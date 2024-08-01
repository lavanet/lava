package chaintracker_test

import (
	"context"
	fmt "fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	chaintracker "github.com/lavanet/lava/v2/protocol/chaintracker"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	rand "github.com/lavanet/lava/v2/utils/rand"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	// TimeForPollingMock = (100 * time.Microsecond)
	TimeForPollingMock = (2 * time.Millisecond)
	SleepTime          = TimeForPollingMock * 2
	SleepChunks        = 5
)

type MockTimeUpdater struct {
	callBack func(time.Duration)
}

func (mtu *MockTimeUpdater) UpdateBlockTime(arg time.Duration) {
	mtu.callBack(arg)
}

type MockChainFetcher struct {
	latestBlock int64
	blockHashes []*chaintracker.BlockStore
	mutex       sync.Mutex
	fork        string
	callBack    func()
}

func (mcf *MockChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return lavasession.RPCProviderEndpoint{}
}

func (mcf *MockChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	if mcf.callBack != nil {
		mcf.callBack()
	}
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

func (mcf *MockChainFetcher) FetchChainID(ctx context.Context) (string, string, error) {
	return "", "", utils.LavaFormatError("FetchChainID not supported for lava chain fetcher", nil)
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

func (mcf *MockChainFetcher) Shrink(newSize int) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	currentSize := len(mcf.blockHashes)
	if currentSize <= newSize {
		return
	}
	newHashes := make([]*chaintracker.BlockStore, newSize)
	copy(newHashes, mcf.blockHashes[currentSize-newSize:])
}

func NewMockChainFetcher(startBlock, blocksToSave int64, callback func()) *MockChainFetcher {
	mockCHainFetcher := MockChainFetcher{callBack: callback}
	for i := int64(0); i < blocksToSave; i++ {
		mockCHainFetcher.SetBlock(startBlock + i)
	}
	return &mockCHainFetcher
}

func TestMain(m *testing.M) {
	// This code will run once before any test cases are executed.
	seed := time.Now().Unix()

	rand.SetSpecificSeed(seed)
	// Run the actual tests
	exitCode := m.Run()
	if exitCode != 0 {
		utils.LavaFormatDebug("failed tests seed", utils.Attribute{Key: "seed", Value: seed})
	}
	os.Exit(exitCode)
}

const startedTestStr = "started test "

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
		{name: "ten block memory 4 block fetch", mockBlocks: 20, requestBlocks: 4, fetcherBlocks: 10, advancements: []int64{0, 1, 0, 0, 1, 1, 1, 0, 2, 0, 5, 1, 10, 1, 1, 1}, requestBlockFrom: spectypes.LATEST_BLOCK - 9, requestBlockTo: spectypes.LATEST_BLOCK - 6, specificBlock: spectypes.NOT_APPLICABLE},
		{name: "ten block memory one block fetch", mockBlocks: 20, requestBlocks: 1, fetcherBlocks: 10, advancements: []int64{0, 1, 0, 0, 1, 1, 1, 0, 2, 0, 5, 1, 10, 1, 1, 1}, requestBlockFrom: spectypes.LATEST_BLOCK, requestBlockTo: spectypes.LATEST_BLOCK, specificBlock: spectypes.NOT_APPLICABLE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils.LavaFormatInfo(startedTestStr + tt.name)
			mockChainFetcher := NewMockChainFetcher(1000, tt.mockBlocks, nil)
			currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()

			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(tt.fetcherBlocks), AverageBlockTime: TimeForPollingMock, ServerBlockMemory: uint64(tt.mockBlocks)}
			chainTracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
			require.NoError(t, err)
			for _, advancement := range tt.advancements {
				for i := 0; i < int(advancement); i++ {
					currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
				}
				for sleepChunk := 0; sleepChunk < SleepChunks; sleepChunk++ {
					time.Sleep(SleepTime) // stateTracker polls asynchronously
					latestBlock := chainTracker.GetAtomicLatestBlockNum()
					if latestBlock >= currentLatestBlockInMock {
						break
					}
				}
				latestBlock := chainTracker.GetAtomicLatestBlockNum()
				require.Equal(t, currentLatestBlockInMock, latestBlock)

				latestBlock, requestedHashes, _, err := chainTracker.GetLatestBlockData(tt.requestBlockFrom, tt.requestBlockTo, tt.specificBlock)
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
		{name: "ten block memory + 3 block fetch", mockBlocks: 100, requestBlocks: 3, fetcherBlocks: 10, advancements: []int64{0, 1, 0, 0, 1, 1, 1, 0, 2, 0, 5, 1, 10, 1, 1, 1}, requestBlockFrom: spectypes.LATEST_BLOCK - 6, requestBlockTo: spectypes.LATEST_BLOCK - 4, specificBlock: spectypes.NOT_APPLICABLE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChainFetcher := NewMockChainFetcher(1000, tt.mockBlocks, nil)
			currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()

			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(tt.fetcherBlocks), AverageBlockTime: TimeForPollingMock, ServerBlockMemory: uint64(tt.mockBlocks)}
			chainTracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
			require.NoError(t, err)
			for _, advancement := range tt.advancements {
				for i := 0; i < int(advancement); i++ {
					currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
				}
				for sleepChunk := 0; sleepChunk < SleepChunks; sleepChunk++ {
					time.Sleep(SleepTime) // stateTracker polls asynchronously
					latestBlock := chainTracker.GetAtomicLatestBlockNum()
					if latestBlock >= currentLatestBlockInMock {
						break
					}
				}
				latestBlock := chainTracker.GetAtomicLatestBlockNum()
				require.Equal(t, currentLatestBlockInMock, latestBlock)

				latestBlock, requestedHashes, _, err := chainTracker.GetLatestBlockData(tt.requestBlockFrom, tt.requestBlockTo, tt.specificBlock)
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
	requestBlockTo := spectypes.LATEST_BLOCK - 4
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
	mockChainFetcher := NewMockChainFetcher(1000, mockBlocks, nil)
	currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()

	// used to identify if the fork callback was called
	callbackCalledFork := false
	forkCallback := func(arg int64) {
		utils.LavaFormatDebug("fork callback called")
		callbackCalledFork = true
	}
	// used to identify if the newLatest callback was called
	callbackCalledNewLatest := false
	callbackCalledTimes := 0
	newBlockCallback := func(blockFrom int64, blockTo int64, hash string) {
		callbackCalledNewLatest = true
		for block := blockFrom + 1; block <= blockTo; block++ {
			callbackCalledTimes++
		}
	}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: TimeForPollingMock, ServerBlockMemory: uint64(mockBlocks), ForkCallback: forkCallback, NewLatestCallback: newBlockCallback}
	chainTracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
	require.NoError(t, err)
	totalAdvancement := 0
	t.Run("one long test", func(t *testing.T) {
		for _, tt := range tests {
			totalAdvancement += int(tt.advancement)
			utils.LavaFormatInfo(startedTestStr + tt.name)
			callbackCalledFork = false
			callbackCalledNewLatest = false
			for i := 0; i < int(tt.advancement); i++ {
				currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
			}
			mockChainFetcher.Fork(tt.fork)
			for sleepChunk := 0; sleepChunk < SleepChunks; sleepChunk++ {
				time.Sleep(SleepTime) // stateTracker polls asynchronously
				latestBlock := chainTracker.GetAtomicLatestBlockNum()
				if latestBlock >= currentLatestBlockInMock && tt.shouldFork == false {
					break
				}
			}
			latestBlock := chainTracker.GetAtomicLatestBlockNum()
			require.Equal(t, currentLatestBlockInMock, latestBlock)

			latestBlock, requestedHashes, _, err := chainTracker.GetLatestBlockData(requestBlockFrom, requestBlockTo, specificBlock)
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
			require.Equal(t, totalAdvancement, callbackCalledTimes)
			if tt.advancement > 0 {
				require.True(t, callbackCalledNewLatest)
			} else {
				require.False(t, callbackCalledNewLatest)
			}
		}
	})
}

func TestChainTrackerFetchSpreadAcrossPollingTime(t *testing.T) {
	t.Run("one long test", func(t *testing.T) {
		mockBlocks := int64(50)
		fetcherBlocks := 1
		called := 0
		lastCall := time.Now()
		timeDiff := 0 * time.Millisecond
		localTimeForPollingMock := 500 * time.Millisecond
		callback := func() {
			called++
			timeDiff = time.Since(lastCall)
			lastCall = time.Now()
		}
		mockChainFetcher := NewMockChainFetcher(1000, mockBlocks, callback)
		mockChainFetcher.AdvanceBlock()
		chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: localTimeForPollingMock, ServerBlockMemory: uint64(mockBlocks)}
		tracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
		require.NoError(t, err)
		// fool the tracker so it thinks blocks will come every localTimeForPollingMock (ms), and not adjust it's polling timers
		for i := 0; i < 50; i++ {
			tracker.AddBlockGap(localTimeForPollingMock, 1)
		}
		// initially we start with 1/16 block probing
		time.Sleep(localTimeForPollingMock)                           // we expect 15+init calls
		require.GreaterOrEqual(t, called, 15*8/10)                    // 15 to give a gap, give a 20% margin
		require.Greater(t, timeDiff, localTimeForPollingMock/16*8/10) // give a 20% margin
		fmt.Println(timeDiff, localTimeForPollingMock/16*8/10, localTimeForPollingMock/8*12/10)
		require.Less(t, timeDiff, localTimeForPollingMock/8*12/10) // give a 20% margin
		mockChainFetcher.AdvanceBlock()                            // we advanced a block
		time.Sleep(localTimeForPollingMock / 2)
		require.LessOrEqual(t, called, (3+16)*12/10) // init + 2 new + 16 from first block advancement, give 20% margin
		require.GreaterOrEqual(t, called, 17*8/10)   // give a 20% margin
		fmt.Println(timeDiff, localTimeForPollingMock/2*12/10, localTimeForPollingMock/8*8/10)
		require.Less(t, timeDiff, localTimeForPollingMock/2*12/10)   // give a 20% margin
		require.Greater(t, timeDiff, localTimeForPollingMock/8*8/10) // give a 20% margin
		time.Sleep(localTimeForPollingMock / 2)
		require.GreaterOrEqual(t, called, (6+16)*8/10)
		require.Less(t, timeDiff, localTimeForPollingMock/8*12/10) // give a 20% margin
		fmt.Println(timeDiff, localTimeForPollingMock/8*12/10, localTimeForPollingMock/16*8/10)
		require.Greater(t, timeDiff, localTimeForPollingMock/16*8/10) // give a 20% margin
	})
}

func TestChainTrackerPollingTimeUpdate(t *testing.T) {
	playbook := []struct {
		name                    string
		localTimeForPollingMock time.Duration
		startDelay              time.Duration
		updateTime              time.Duration
	}{
		{
			name:                    "no-delay polling time decrease big",
			localTimeForPollingMock: 16 * time.Millisecond,
			startDelay:              0,
			updateTime:              5 * time.Millisecond,
		},
		{
			name:                    "no-delay polling time decrease small",
			localTimeForPollingMock: 8 * time.Millisecond,
			startDelay:              0,
			updateTime:              5 * time.Millisecond,
		},
		{
			name:                    "half delay polling time decrease big",
			localTimeForPollingMock: 16 * time.Millisecond,
			startDelay:              2500 * time.Microsecond,
			updateTime:              5 * time.Millisecond,
		},
		{
			name:                    "half delay polling time decrease small",
			localTimeForPollingMock: 8 * time.Millisecond,
			startDelay:              2500 * time.Microsecond,
			updateTime:              5 * time.Millisecond,
		},
		{
			name:                    "90Percent  delay polling time decrease big",
			localTimeForPollingMock: 16 * time.Millisecond,
			startDelay:              4500 * time.Microsecond,
			updateTime:              5 * time.Millisecond,
		},
		{
			name:                    "90Percent delay polling time decrease small",
			localTimeForPollingMock: 8 * time.Millisecond,
			startDelay:              4500 * time.Microsecond,
			updateTime:              5 * time.Millisecond,
		},
		{
			name:                    "no-delay polling time increase big",
			localTimeForPollingMock: 5 * time.Millisecond,
			startDelay:              0,
			updateTime:              16 * time.Millisecond,
		},
		{
			name:                    "no-delay polling time increase small",
			localTimeForPollingMock: 5 * time.Millisecond,
			startDelay:              0,
			updateTime:              8 * time.Millisecond,
		},
		{
			name:                    "half delay polling time increase big",
			localTimeForPollingMock: 5 * time.Millisecond,
			startDelay:              8 * time.Millisecond,
			updateTime:              16 * time.Millisecond,
		},
		{
			name:                    "half delay polling time increase small",
			localTimeForPollingMock: 5 * time.Millisecond,
			startDelay:              4 * time.Millisecond,
			updateTime:              8 * time.Millisecond,
		},
		{
			name:                    "90Percent  delay polling time increase big",
			localTimeForPollingMock: 5 * time.Millisecond,
			startDelay:              14 * time.Millisecond,
			updateTime:              16 * time.Millisecond,
		},
		{
			name:                    "90Percent delay polling time increase small",
			localTimeForPollingMock: 5 * time.Millisecond,
			startDelay:              7200 * time.Microsecond,
			updateTime:              8 * time.Millisecond,
		},
	}
	iterations := chaintracker.PollingUpdateLength
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			mockBlocks := int64(25)
			fetcherBlocks := 1
			called := false
			callback := func() {
			}
			updatedTime := 0 * time.Second
			updateCallback := func(arg time.Duration) {
				called = true
				updatedTime = arg
			}
			mockTimeUpdater := MockTimeUpdater{callBack: updateCallback}
			mockChainFetcher := NewMockChainFetcher(1000, mockBlocks, callback)
			mockChainFetcher.AdvanceBlock()
			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: play.localTimeForPollingMock, ServerBlockMemory: uint64(mockBlocks)}
			tracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
			tracker.RegisterForBlockTimeUpdates(&mockTimeUpdater)
			require.NoError(t, err)
			// initial delay
			time.Sleep(play.startDelay)
			// chainTracker will poll every localTimeForPollingMock/16
			for i := 0; i < iterations*2+1; i++ {
				mockChainFetcher.AdvanceBlock()
				time.Sleep(play.updateTime)
			}
			// give it more time to update in case it didn't trigger on slow machines
			if !called {
				for i := 0; i < iterations*2; i++ {
					mockChainFetcher.AdvanceBlock()
					time.Sleep(play.updateTime)
				}
			}
			require.InDelta(t, play.updateTime, updatedTime, float64(play.updateTime)*0.2)
			// if we wait more time we expect this to stay correct
			for i := 0; i < iterations*4; i++ {
				mockChainFetcher.AdvanceBlock()
				time.Sleep(play.updateTime)
			}
			require.InDelta(t, play.updateTime, updatedTime, float64(play.updateTime)*0.2)
		})
	}
}

func TestChainTrackerMaintainMemory(t *testing.T) {
	mockBlocks := int64(100)
	requestBlocks := 4
	fetcherBlocks := 50
	requestBlockFrom := spectypes.LATEST_BLOCK - 6
	requestBlockTo := spectypes.LATEST_BLOCK - 4
	specificBlock := spectypes.LATEST_BLOCK - 30 // needs to be smaller than requestBlockFrom, can't be NOT_APPLICABLE
	tests := []struct {
		name        string
		advancement int64
		shrink      bool
	}{
		{name: "[t00]", shrink: false, advancement: 0},
		{name: "[t01]", shrink: false, advancement: 1},
		{name: "[t02]", shrink: false, advancement: 0},
		{name: "[t03]", shrink: false, advancement: 0},
		{name: "[t04]", shrink: false, advancement: 1},
		{name: "[t05]", shrink: true, advancement: 1},
		{name: "[t06]", shrink: false, advancement: 1},
		{name: "[t07]", shrink: false, advancement: 0},
		{name: "[t08]", shrink: false, advancement: 2},
		{name: "[t09]", shrink: false, advancement: 0},
		{name: "[t10]", shrink: false, advancement: 5},
		{name: "[t11]", shrink: false, advancement: 1},
	}
	mockChainFetcher := NewMockChainFetcher(1000, mockBlocks, nil)
	currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()

	// used to identify if the fork callback was called
	callbackCalledFork := false
	forkCallback := func(arg int64) {
		utils.LavaFormatDebug("fork callback called")
		callbackCalledFork = true
	}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: TimeForPollingMock, ServerBlockMemory: uint64(mockBlocks), ForkCallback: forkCallback}
	chainTracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
	require.NoError(t, err)
	t.Run("one long test", func(t *testing.T) {
		for _, tt := range tests {
			utils.LavaFormatInfo(startedTestStr + tt.name)
			callbackCalledFork = false
			for i := 0; i < int(tt.advancement); i++ {
				currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
			}
			if tt.shrink {
				mockChainFetcher.Shrink(50) // do not allow previous block fetches
			}
			for sleepChunk := 0; sleepChunk < SleepChunks; sleepChunk++ {
				time.Sleep(SleepTime) // stateTracker polls asynchronously
				latestBlock := chainTracker.GetAtomicLatestBlockNum()
				if latestBlock >= currentLatestBlockInMock && tt.shrink == false {
					break
				}
			}
			latestBlock := chainTracker.GetAtomicLatestBlockNum()
			require.Equal(t, currentLatestBlockInMock, latestBlock)

			latestBlock, requestedHashes, _, err := chainTracker.GetLatestBlockData(requestBlockFrom, requestBlockTo, specificBlock)
			require.Equal(t, currentLatestBlockInMock, latestBlock)
			require.NoError(t, err)
			require.Equal(t, requestBlocks, len(requestedHashes))
			if requestBlockFrom != spectypes.NOT_APPLICABLE {
				fromNum := chaintracker.LatestArgToBlockNum(requestBlockFrom, latestBlock)
				// in this test specific is always smaller than requestBlockFrom therefore from starts on index 1
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[1].Hash, fromNum), "incompatible hash %s on block %d", requestedHashes[1].Hash, fromNum)
			}
			if specificBlock != spectypes.NOT_APPLICABLE {
				specificNum := chaintracker.LatestArgToBlockNum(specificBlock, latestBlock)
				// in this test specific is always smaller than requestBlockFrom therefore first
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[0].Hash, specificNum), "latestBlock: %d, blockHashes: %v", latestBlock, requestedHashes)
			}
			for idx := 1; idx < len(requestedHashes)-1; idx++ {
				require.Equal(t, requestedHashes[idx].Block+1, requestedHashes[idx+1].Block, "latestBlock: %d, blockHashes: %v", latestBlock, requestedHashes)
				require.True(t, mockChainFetcher.IsCorrectHash(requestedHashes[idx].Hash, requestedHashes[idx].Block))
			}
			require.False(t, callbackCalledFork)
		}
	})
}

func TestFindRequestedBlockHash(t *testing.T) {
	mockBlocks := int64(100)
	fetcherBlocks := 50
	mockChainFetcher := NewMockChainFetcher(1000, mockBlocks, nil)
	currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()

	chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: TimeForPollingMock, ServerBlockMemory: uint64(mockBlocks)}
	chainTracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
	require.NoError(t, err)
	latestBlock, onlyLatestBlockData, _, err := chainTracker.GetLatestBlockData(spectypes.LATEST_BLOCK, spectypes.LATEST_BLOCK, spectypes.NOT_APPLICABLE)
	require.NoError(t, err)
	require.Equal(t, currentLatestBlockInMock, latestBlock)
	requestedHash, hashesMap := chaintracker.FindRequestedBlockHash(onlyLatestBlockData, latestBlock, spectypes.LATEST_BLOCK, spectypes.LATEST_BLOCK, map[int64]interface{}{})
	require.NotNil(t, requestedHash)
	require.Len(t, hashesMap, 1)

	latestBlock, onlyLatestBlockData, _, err = chainTracker.GetLatestBlockData(spectypes.LATEST_BLOCK-3, spectypes.LATEST_BLOCK, spectypes.NOT_APPLICABLE)
	require.NoError(t, err)
	require.Equal(t, currentLatestBlockInMock, latestBlock)
	requestedHash, hashesMap = chaintracker.FindRequestedBlockHash(onlyLatestBlockData, latestBlock, spectypes.LATEST_BLOCK, spectypes.LATEST_BLOCK-3, map[int64]interface{}{})
	require.NotNil(t, requestedHash)
	require.Len(t, hashesMap, 4)

	latestBlock, onlyLatestBlockData, _, err = chainTracker.GetLatestBlockData(currentLatestBlockInMock-3, currentLatestBlockInMock, currentLatestBlockInMock)
	require.NoError(t, err)
	require.Equal(t, currentLatestBlockInMock, latestBlock)
	requestedHash, hashesMap = chaintracker.FindRequestedBlockHash(onlyLatestBlockData, latestBlock, currentLatestBlockInMock, currentLatestBlockInMock-3, map[int64]interface{}{})
	require.NotNil(t, requestedHash)
	require.Len(t, hashesMap, 4)
}
