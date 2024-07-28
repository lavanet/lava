package rpcconsumer

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type relayProcessorMetricsMock struct{}

func (romm *relayProcessorMetricsMock) SetRelayNodeErrorMetric(chainId string, apiInterface string) {}

func (romm *relayProcessorMetricsMock) GetChainIdAndApiInterface() (string, string) {
	return "testId", "testInterface"
}

var (
	relayRetriesManagerInstance = NewRelayRetriesManager()
	relayProcessorMetrics       = &relayProcessorMetricsMock{}
)

func sendSuccessResp(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, nil)
	response := &relayResponse{
		relayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: []byte("ok")},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   http.StatusOK,
		},
		err: nil,
	}
	relayProcessor.SetResponse(response)
}

func sendProtocolError(relayProcessor *RelayProcessor, provider string, delay time.Duration, err error) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, err)
	response := &relayResponse{
		relayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: []byte(`{"message":"bad","code":123}`)},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   0,
		},
		err: err,
	}
	relayProcessor.SetResponse(response)
}

func sendNodeError(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, nil)
	response := &relayResponse{
		relayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: []byte(`{"message":"bad","code":123}`)},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   http.StatusInternalServerError,
		},
		err: nil,
	}
	relayProcessor.SetResponse(response)
}

func TestRelayProcessorHappyFlow(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)

		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		go sendSuccessResp(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Zero(t, protocolErrors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err)
		require.Equal(t, string(returnedResult.Reply.Data), "ok")
	})
}

func TestRelayProcessorNodeErrorRetryFlow(t *testing.T) {
	t.Run("retry_flow", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)

		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		// check first reply
		go sendNodeError(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults := relayProcessor.HasRequiredNodeResults()
		require.False(t, requiredNodeResults)
		// check first retry
		go sendNodeError(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk = relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults = relayProcessor.HasRequiredNodeResults()
		require.False(t, requiredNodeResults)

		// check first second retry
		go sendNodeError(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk = relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults = relayProcessor.HasRequiredNodeResults()
		require.True(t, requiredNodeResults)

		// 2nd relay, same inputs
		// check hash map flow:
		chainMsg, err = chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor = NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)
		usedProviders = relayProcessor.GetUsedProviders()
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse = usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap = lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		// check first reply, this time we have hash in map, so we don't retry node errors.
		go sendNodeError(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk = relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults = relayProcessor.HasRequiredNodeResults()
		require.True(t, requiredNodeResults)

		// 3nd relay, same inputs, this time a successful relay, should remove the hash from the map
		chainMsg, err = chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor = NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)
		usedProviders = relayProcessor.GetUsedProviders()
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse = usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap = lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		// check first reply, this time we have hash in map, so we don't retry node errors.
		hash, err := relayProcessor.getInputMsgInfoHashString()
		require.NoError(t, err)
		require.True(t, relayProcessor.relayRetriesManager.CheckHashInMap(hash))
		go sendSuccessResp(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk = relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults = relayProcessor.HasRequiredNodeResults()
		require.True(t, requiredNodeResults)

		// A way for us to break early from sleep, just waiting up to 5 seconds and breaking as soon as the value we expect is there.
		// After 5 seconds if its not there test will fail
		for i := 0; i < 100; i++ {
			if !relayProcessor.relayRetriesManager.CheckHashInMap(hash) {
				break
			}
			time.Sleep(time.Millisecond * 50) // sleep up to 5 seconds
		}
		// after the sleep we should not have the hash anymore in the map as it was removed by a successful relay.
		require.False(t, relayProcessor.relayRetriesManager.CheckHashInMap(hash))
	})
}

func TestRelayProcessorTimeout(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)

		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		go func() {
			time.Sleep(time.Millisecond * 5)
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
			defer cancel()
			canUse := usedProviders.TryLockSelection(ctx)
			require.NoError(t, ctx.Err())
			require.Nil(t, canUse)
			consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test3": &lavasession.SessionInfo{}, "lava@test4": &lavasession.SessionInfo{}}
			usedProviders.AddUsed(consumerSessionsMap, nil)
		}()
		go sendSuccessResp(relayProcessor, "lava@test", time.Millisecond*20)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Zero(t, protocolErrors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err)
		require.Equal(t, string(returnedResult.Reply.Data), "ok")
	})
}

func TestRelayProcessorRetry(t *testing.T) {
	t.Run("retry", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)

		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		go sendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go sendSuccessResp(relayProcessor, "lava@test2", time.Millisecond*20)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(1), protocolErrors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err)
		require.Equal(t, string(returnedResult.Reply.Data), "ok")
	})
}

func TestRelayProcessorRetryNodeError(t *testing.T) {
	t.Run("retry", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)

		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		go sendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go sendNodeError(relayProcessor, "lava@test2", time.Millisecond*20)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(1), protocolErrors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err)
		require.Equal(t, string(returnedResult.Reply.Data), `{"message":"bad","code":123}`)
		require.Equal(t, returnedResult.StatusCode, http.StatusInternalServerError)
	})
}

func TestRelayProcessorStatefulApi(t *testing.T) {
	t.Run("stateful", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/tx/v1beta1/txs", []byte("data"), http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)
		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava4@test": &lavasession.SessionInfo{}, "lava3@test": &lavasession.SessionInfo{}, "lava@test": &lavasession.SessionInfo{}, "lava2@test": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		go sendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go sendNodeError(relayProcessor, "lava2@test", time.Millisecond*20)
		go sendNodeError(relayProcessor, "lava3@test", time.Millisecond*25)
		go sendSuccessResp(relayProcessor, "lava4@test", time.Millisecond*100)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(1), protocolErrors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err)
		require.Equal(t, string(returnedResult.Reply.Data), "ok")
		require.Equal(t, http.StatusOK, returnedResult.StatusCode)
	})
}

func TestRelayProcessorStatefulApiErr(t *testing.T) {
	t.Run("stateful", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/tx/v1beta1/txs", []byte("data"), http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)
		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava4@test": &lavasession.SessionInfo{}, "lava3@test": &lavasession.SessionInfo{}, "lava@test": &lavasession.SessionInfo{}, "lava2@test": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		go sendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go sendNodeError(relayProcessor, "lava2@test", time.Millisecond*20)
		go sendNodeError(relayProcessor, "lava3@test", time.Millisecond*25)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.Error(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(1), protocolErrors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err)
		require.Equal(t, string(returnedResult.Reply.Data), `{"message":"bad","code":123}`)
		require.Equal(t, returnedResult.StatusCode, http.StatusInternalServerError)
	})
}

func TestRelayProcessorLatest(t *testing.T) {
	t.Run("latest req", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/latest", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMsg, nil, "", "", false, relayProcessorMetrics, relayProcessorMetrics, false, relayRetriesManagerInstance)
		usedProviders := relayProcessor.GetUsedProviders()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())

		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		go sendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go sendSuccessResp(relayProcessor, "lava@test2", time.Millisecond*20)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(1), protocolErrors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err)
		require.Equal(t, string(returnedResult.Reply.Data), "ok")
		// reqBlock, _ := chainMsg.RequestedBlock()
		// require.NotEqual(t, spectypes.LATEST_BLOCK, reqBlock) // disabled until we enable requested block modification again
	})
}
