package rpcprovider

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	spectypes "github.com/lavanet/lava/x/spec/types"
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

func (mct *MockChainTracker) SetLatestBlock(newLatest int64) {
	mct.latestBlock = newLatest
	mct.changeTime = time.Now()
}

func TestHandleConsistency(t *testing.T) {

	plays := []struct {
		seenBlock          int64
		requestBlock       int64
		name               string
		specId             string
		err                error
		timeout            time.Duration
		chainTrackerBlocks []int64
	}{
		{
			seenBlock:          0,
			requestBlock:       100,
			name:               "success",
			specId:             "LAV1",
			err:                nil,
			timeout:            10 * time.Millisecond,
			chainTrackerBlocks: []int64{100},
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
			chainParser, chainProxy, _, closeServer, err := chainlib.CreateChainLibMocks(ts.Ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../../", nil)
			if closeServer != nil {
				defer closeServer()
			}
			mockChainTracker := &MockChainTracker{}
			require.GreaterOrEqual(t, len(play.chainTrackerBlocks), 1)
			mockChainTracker.SetLatestBlock(play.chainTrackerBlocks[0])
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
					mockChainTracker.SetLatestBlock(play.chainTrackerBlocks[1])
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), play.timeout)
			latestBlock, _, err := rpcproviderServer.handleConsistency(ctx, seenBlock, requestBlock, averageBlockTime, blockLagForQosSync, blocksInFinalizationData, blockDistanceToFinalization)
			cancel()
			require.Equal(t, err == nil, play.err == nil)
			if play.err == nil {
				require.LessOrEqual(t, seenBlock, latestBlock)
				require.LessOrEqual(t, requestBlock, latestBlock)
			}
		})
	}
}
