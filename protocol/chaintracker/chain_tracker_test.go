package chaintracker_test

import (
	"context"
	fmt "fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	chaintracker "github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	rand "github.com/lavanet/lava/v5/utils/rand"
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

func (mcf *MockChainFetcher) CustomMessage(ctx context.Context, path string, data []byte, connectionType string, apiName string) ([]byte, error) {
	return nil, utils.LavaFormatError("Not Implemented CustomMessage for MockChainFetcher", nil)
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

func TestChainTrackerCallbacks(t *testing.T) {
	mockBlocks := int64(100)
	fetcherBlocks := 10
	tests := []struct {
		name        string
		advancement int64
	}{
		{name: "[t00]", advancement: 0},
		{name: "[t01]", advancement: 1},
		{name: "[t02]", advancement: 0},
		{name: "[t03]", advancement: 0},
		{name: "[t04]", advancement: 1},
		{name: "[t05]", advancement: 1},
		{name: "[t06]", advancement: 1},
		{name: "[t07]", advancement: 0},
		{name: "[t08]", advancement: 2},
		{name: "[t09]", advancement: 0},
		{name: "[t10]", advancement: 5},
		{name: "[t11]", advancement: 1},
		{name: "[t12]", advancement: 10},
		{name: "[t13]", advancement: 15},
		{name: "[t14]", advancement: 1},
		{name: "[t15]", advancement: 1},
		{name: "[t16]", advancement: 0},
	}
	mockChainFetcher := NewMockChainFetcher(1000, mockBlocks, nil)
	currentLatestBlockInMock := mockChainFetcher.AdvanceBlock()

	// used to identify if the newLatest callback was called
	callbackCalledNewLatest := false
	callbackCalledTimes := 0
	newBlockCallback := func(blockFrom int64, blockTo int64) {
		callbackCalledNewLatest = true
		for block := blockFrom + 1; block <= blockTo; block++ {
			callbackCalledTimes++
		}
	}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: TimeForPollingMock, ServerBlockMemory: uint64(mockBlocks), NewLatestCallback: newBlockCallback, ParseDirectiveEnabled: true}
	chainTracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
	require.NoError(t, err)
	chainTracker.StartAndServe(context.Background())
	totalAdvancement := 0
	t.Run("one long test", func(t *testing.T) {
		for _, tt := range tests {
			totalAdvancement += int(tt.advancement)
			utils.LavaFormatInfo(startedTestStr + tt.name)
			callbackCalledNewLatest = false
			for i := 0; i < int(tt.advancement); i++ {
				currentLatestBlockInMock = mockChainFetcher.AdvanceBlock()
			}
			for sleepChunk := 0; sleepChunk < SleepChunks; sleepChunk++ {
				time.Sleep(SleepTime) // stateTracker polls asynchronously
				latestBlock := chainTracker.GetAtomicLatestBlockNum()
				if latestBlock > currentLatestBlockInMock {
					break
				}
			}
			latestBlock := chainTracker.GetAtomicLatestBlockNum()
			require.Equal(t, currentLatestBlockInMock, latestBlock)

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
		chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: localTimeForPollingMock, ServerBlockMemory: uint64(mockBlocks), ParseDirectiveEnabled: true}
		tracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
		require.NoError(t, err)
		tracker.StartAndServe(context.Background())
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
			chainTrackerConfig := chaintracker.ChainTrackerConfig{BlocksToSave: uint64(fetcherBlocks), AverageBlockTime: play.localTimeForPollingMock, ServerBlockMemory: uint64(mockBlocks), ParseDirectiveEnabled: true}
			tracker, err := chaintracker.NewChainTracker(context.Background(), mockChainFetcher, chainTrackerConfig)
			tracker.StartAndServe(context.Background())
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
			require.InDelta(t, play.updateTime, updatedTime, float64(play.updateTime)*0.3)
			// if we wait more time we expect this to stay correct
			for i := 0; i < iterations*4; i++ {
				mockChainFetcher.AdvanceBlock()
				time.Sleep(play.updateTime)
			}
			require.InDelta(t, play.updateTime, updatedTime, float64(play.updateTime)*0.3)
		})
	}
}
