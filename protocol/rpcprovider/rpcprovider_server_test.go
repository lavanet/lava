package rpcprovider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/rpcprovider/reliabilitymanager"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

type MockChainTracker struct {
	latestBlock int64
	changeTime  time.Time
}

func (mct *MockChainTracker) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	return mct.latestBlock, nil, mct.changeTime, nil
}

func (mct *MockChainTracker) GetLatestBlockNum() (int64, time.Time) {
	return mct.latestBlock, mct.changeTime
}

func (mct *MockChainTracker) SetLatestBlock(newLatest int64, changeTime time.Time) {
	mct.latestBlock = newLatest
	mct.changeTime = changeTime
}

func TestHandleConsistency(t *testing.T) {
	plays := []struct {
		seenBlock          int64
		requestBlock       int64
		name               string
		specId             string
		err                error
		timeout            time.Duration
		changeTime         time.Duration
		chainTrackerBlocks []int64
		sleep              bool
	}{
		{
			name:               "success",
			seenBlock:          0,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "success with seen eq",
			seenBlock:          100,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "success with seen less",
			seenBlock:          99,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "success with seen more",
			seenBlock:          99,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "latest success",
			seenBlock:          0,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond,
			sleep:              false,
		},
		{
			name:               "latest success with seen lower",
			seenBlock:          99,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond,
			sleep:              false,
		},
		{
			name:               "latest with seen higher - sleep, fail",
			seenBlock:          101,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "latest success with seen higher - sleep, succeed",
			seenBlock:          101,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                nil,
			timeout:            25 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100, 101},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "latest with seen higher - immediate fail",
			seenBlock:          100000,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         1 * time.Millisecond, // 1 ms ago
			sleep:              false,
		},
		{
			name:               "requested higher - immediate fail",
			seenBlock:          10001,
			requestBlock:       10001,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         1 * time.Millisecond, // 1 ms ago,
			sleep:              false,
		},
		{
			name:               "requested higher seen 0 - success",
			seenBlock:          0,
			requestBlock:       10001,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         1 * time.Millisecond, // 1 ms ago,
			sleep:              false,
		},
		{
			name:               "requested higher seen lower- success",
			seenBlock:          99,
			requestBlock:       101,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         100 * time.Second,
			sleep:              false,
		},
		{
			name:               "requested higher seen higher- sleep fail",
			seenBlock:          101,
			requestBlock:       101,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "requested much higher seen higher - sleep success",
			seenBlock:          101,
			requestBlock:       10001,
			specId:             "LAV1",
			err:                nil,
			timeout:            50 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100, 101},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "requested higher seen higher - sleep success",
			seenBlock:          101,
			requestBlock:       101,
			specId:             "LAV1",
			err:                nil,
			timeout:            50 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100, 101},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
	}
	for _, play := range plays {
		t.Run(play.name, func(t *testing.T) {
			specId := play.specId
			ts := chainlib.SetupForTests(t, 1, specId, "../../")
			replyDataBuf := []byte(`{"reply": "REPLY-STUB"}`)
			serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle the incoming request and provide the desired response
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, string(replyDataBuf))
			})
			chainParser, chainProxy, _, closeServer, _, err := chainlib.CreateChainLibMocks(ts.Ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
			if closeServer != nil {
				defer closeServer()
			}
			mockChainTracker := &MockChainTracker{}
			require.GreaterOrEqual(t, len(play.chainTrackerBlocks), 1)
			calls := 1                                                                                      // how many times we have setLatestBlock in the mock
			mockChainTracker.SetLatestBlock(play.chainTrackerBlocks[0], time.Now().Add(-1*play.changeTime)) // change time is only in the past
			require.NoError(t, err)
			reliabilityManager := reliabilitymanager.NewReliabilityManager(mockChainTracker, nil, ts.Providers[0].Addr.String(), chainProxy, chainParser)
			rpcproviderServer := RPCProviderServer{
				reliabilityManager: reliabilityManager,
				rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{
					ChainID: specId,
				},
			}
			seenBlock := play.seenBlock
			requestBlock := play.requestBlock
			blockLagForQosSync, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
			go func() {
				// advance mockChainTracker
				if len(play.chainTrackerBlocks) > 1 {
					nextBlock := play.chainTrackerBlocks[1]
					time.Sleep(6 * time.Millisecond)
					mockChainTracker.SetLatestBlock(nextBlock, time.Now())
					calls += 1
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), play.timeout)
			latestBlock, _, timeSlept, err := rpcproviderServer.handleConsistency(ctx, play.timeout, seenBlock, requestBlock, averageBlockTime, blockLagForQosSync, blocksInFinalizationData, blockDistanceToFinalization)
			cancel()
			if play.err != nil {
				require.Error(t, err, strconv.Itoa(calls))
			} else {
				require.NoError(t, err, strconv.Itoa(calls))
			}
			require.Less(t, timeSlept, play.timeout)
			if play.sleep {
				require.NotZero(t, timeSlept)
			} else {
				require.Zero(t, timeSlept)
			}
			if play.err == nil {
				require.LessOrEqual(t, seenBlock, latestBlock)
			}
		})
	}
}
