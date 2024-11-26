package rpcconsumer

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

type relayProcessorMetricsMock struct{}

func (romm *relayProcessorMetricsMock) SetRelayNodeErrorMetric(providerAddress, chainId, apiInterface string) {
}

func (romm *relayProcessorMetricsMock) SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
}

func (romm *relayProcessorMetricsMock) SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string) {
}

func (romm *relayProcessorMetricsMock) SetNodeErrorAttemptMetric(chainId string, apiInterface string) {
}

func (romm *relayProcessorMetricsMock) GetChainIdAndApiInterface() (string, string) {
	return "testId", "testInterface"
}

var (
	relayRetriesManagerInstance = lavaprotocol.NewRelayRetriesManager()
	relayProcessorMetrics       = &relayProcessorMetricsMock{}
)

func sendSuccessRespJsonRpc(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	id, _ := json.Marshal(1)
	resultBody, _ := json.Marshal(map[string]string{"result": "success"})
	res := rpcclient.JsonrpcMessage{
		Version: "2.0",
		ID:      id,
		Result:  resultBody,
	}
	resBytes, _ := json.Marshal(res)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &relayResponse{
		relayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: resBytes, LatestBlock: 1},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   http.StatusOK,
		},
		err: nil,
	}
	relayProcessor.SetResponse(response)
}

func sendSuccessResp(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &relayResponse{
		relayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: []byte("ok"), LatestBlock: 1},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   http.StatusOK,
		},
		err: nil,
	}
	relayProcessor.SetResponse(response)
}

func sendProtocolError(relayProcessor *RelayProcessor, provider string, delay time.Duration, err error) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), err)
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
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
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

func sendNodeErrorJsonRpc(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	id, _ := json.Marshal(1)
	res := rpcclient.JsonrpcMessage{
		Version: "2.0",
		ID:      id,
		Error:   &rpcclient.JsonError{Code: 1, Message: "test"},
	}
	resBytes, _ := json.Marshal(res)

	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &relayResponse{
		relayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: resBytes},
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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		dappId := "dapp"
		consumerIp := "123.11"
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, dappId, consumerIp)
		consistency := NewConsumerConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, consistency, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

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
		var seenBlock int64
		var found bool
		// wait for cache to be added asynchronously
		for i := 0; i < 10; i++ {
			seenBlock, found = consistency.GetSeenBlock(protocolMessage.GetUserData())
			if found {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		require.True(t, found)
		require.Equal(t, seenBlock, int64(1))
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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
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
		requiredNodeResults, _ := relayProcessor.HasRequiredNodeResults()
		require.False(t, requiredNodeResults)
		// check first retry
		go sendNodeError(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk = relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults, _ = relayProcessor.HasRequiredNodeResults()
		require.False(t, requiredNodeResults)

		// check first second retry
		go sendNodeError(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk = relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults, _ = relayProcessor.HasRequiredNodeResults()
		require.True(t, requiredNodeResults)

		// 2nd relay, same inputs
		// check hash map flow:
		chainMsg, err = chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage = chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders = lavasession.NewUsedProviders(nil)
		relayProcessor = NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

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
		requiredNodeResults, _ = relayProcessor.HasRequiredNodeResults()
		require.True(t, requiredNodeResults)

		// 3nd relay, different inputs
		// check hash map flow:
		chainMsg, err = chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/18", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage = chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders = lavasession.NewUsedProviders(nil)
		relayProcessor = NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

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
		requiredNodeResults, _ = relayProcessor.HasRequiredNodeResults()
		// check our hashing mechanism works with different inputs
		require.False(t, requiredNodeResults)

		// 4th relay, same inputs, this time a successful relay, should remove the hash from the map
		chainMsg, err = chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage = chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders = lavasession.NewUsedProviders(nil)
		relayProcessor = NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))
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
		require.True(t, relayProcessor.relayRetriesManager.CheckHashInCache(hash))
		go sendSuccessResp(relayProcessor, "lava@test", time.Millisecond*5)
		err = relayProcessor.WaitForResults(context.Background())
		require.NoError(t, err)
		resultsOk = relayProcessor.HasResults()
		require.True(t, resultsOk)
		requiredNodeResults, _ = relayProcessor.HasRequiredNodeResults()
		require.True(t, requiredNodeResults)

		// A way for us to break early from sleep, just waiting up to 5 seconds and breaking as soon as the value we expect is there.
		// After 5 seconds if its not there test will fail
		for i := 0; i < 100; i++ {
			if !relayProcessor.relayRetriesManager.CheckHashInCache(hash) {
				break
			}
			time.Sleep(time.Millisecond * 50) // sleep up to 5 seconds
		}
		// after the sleep we should not have the hash anymore in the map as it was removed by a successful relay.
		require.False(t, relayProcessor.relayRetriesManager.CheckHashInCache(hash))
	})

	t.Run("retry_flow_disabled", func(t *testing.T) {
		ctx := context.Background()
		relayCountOnNodeError = 0
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

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
		requiredNodeResults, _ := relayProcessor.HasRequiredNodeResults()
		require.True(t, requiredNodeResults)
		relayCountOnNodeError = 2
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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))

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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/tx/v1beta1/txs", []byte("data"), http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))
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
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		for i := 0; i < 10; i++ {
			err := relayProcessor.WaitForResults(ctx)
			require.NoError(t, err)
			// Decide if we need to resend or not
			if results, _ := relayProcessor.HasRequiredNodeResults(); results {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		resultsOk, _ = relayProcessor.HasRequiredNodeResults()
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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/tx/v1beta1/txs", []byte("data"), http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))
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
		for i := 0; i < 2; i++ {
			relayProcessor.WaitForResults(ctx)
		}
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
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/latest", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, nil, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &RPCConsumerServer{}, protocolMessage, nil, false, relayProcessorMetrics))
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
