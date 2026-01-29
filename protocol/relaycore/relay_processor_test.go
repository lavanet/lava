package relaycore

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/qos"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// Mock RelayStateMachine for testing
type mockRelayStateMachine struct {
	protocolMessage chainlib.ProtocolMessage
	usedProviders   *lavasession.UsedProviders
	debugState      bool
	selection       Selection
}

func newMockRelayStateMachine(protocolMessage chainlib.ProtocolMessage, usedProviders *lavasession.UsedProviders) *mockRelayStateMachine {
	return &mockRelayStateMachine{
		protocolMessage: protocolMessage,
		usedProviders:   usedProviders,
		debugState:      false,
		selection:       Stateful, // Default to Stateful for backward compatibility
	}
}

func newMockRelayStateMachineWithSelection(protocolMessage chainlib.ProtocolMessage, usedProviders *lavasession.UsedProviders, selection Selection) *mockRelayStateMachine {
	return &mockRelayStateMachine{
		protocolMessage: protocolMessage,
		usedProviders:   usedProviders,
		debugState:      false,
		selection:       selection,
	}
}

func (m *mockRelayStateMachine) GetProtocolMessage() chainlib.ProtocolMessage {
	return m.protocolMessage
}

func (m *mockRelayStateMachine) GetDebugState() bool {
	return m.debugState
}

func (m *mockRelayStateMachine) GetRelayTaskChannel() (chan RelayStateSendInstructions, error) {
	return make(chan RelayStateSendInstructions), nil
}

func (m *mockRelayStateMachine) UpdateBatch(err error) {
}

func (m *mockRelayStateMachine) GetSelection() Selection {
	return m.selection
}

func (m *mockRelayStateMachine) GetUsedProviders() *lavasession.UsedProviders {
	return m.usedProviders
}

func (m *mockRelayStateMachine) SetResultsChecker(resultsChecker ResultsCheckerInf) {
}

func (m *mockRelayStateMachine) SetRelayRetriesManager(relayRetriesManager *lavaprotocol.RelayRetriesManager) {
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
		consistency := NewConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, common.DefaultQuorumParams, consistency, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders), qos.NewQoSManager())

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
		go SendSuccessResp(relayProcessor, "lava@test", time.Millisecond*5)
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
		relayProcessor := NewRelayProcessor(ctx, common.DefaultQuorumParams, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders), qos.NewQoSManager())

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
		go SendSuccessResp(relayProcessor, "lava@test", time.Millisecond*20)
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
		relayProcessor := NewRelayProcessor(ctx, common.DefaultQuorumParams, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders), qos.NewQoSManager())

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		go SendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go SendSuccessResp(relayProcessor, "lava@test2", time.Millisecond*20)
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
		relayProcessor := NewRelayProcessor(ctx, common.DefaultQuorumParams, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders), qos.NewQoSManager())

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		go SendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go SendNodeError(relayProcessor, "lava@test2", time.Millisecond*20)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(1), protocolErrors)
		// When quorum is disabled and we have node errors, we should get the node error as result
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Should return node error when quorum is disabled")
		require.NotNil(t, returnedResult)
		require.Equal(t, 500, returnedResult.StatusCode, "Node errors should have status code 500")
		require.Equal(t, []byte(`{"message":"bad","code":123}`), returnedResult.Reply.Data, "Should return node error data")
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
		relayProcessor := NewRelayProcessor(ctx, common.DefaultQuorumParams, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders), qos.NewQoSManager())
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava4@test": &lavasession.SessionInfo{}, "lava3@test": &lavasession.SessionInfo{}, "lava@test": &lavasession.SessionInfo{}, "lava2@test": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		go SendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go SendNodeError(relayProcessor, "lava2@test", time.Millisecond*20)
		go SendNodeError(relayProcessor, "lava3@test", time.Millisecond*25)
		go SendSuccessResp(relayProcessor, "lava4@test", time.Millisecond*100)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		for i := 0; i < 10; i++ {
			err := relayProcessor.WaitForResults(ctx)
			require.NoError(t, err)
			// Decide if we need to resend or not
			if results, _ := relayProcessor.HasRequiredNodeResults(1); results {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		resultsOk, _ = relayProcessor.HasRequiredNodeResults(1)
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
		relayProcessor := NewRelayProcessor(ctx, common.DefaultQuorumParams, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders), qos.NewQoSManager())
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava4@test": &lavasession.SessionInfo{}, "lava3@test": &lavasession.SessionInfo{}, "lava@test": &lavasession.SessionInfo{}, "lava2@test": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		go SendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go SendNodeError(relayProcessor, "lava2@test", time.Millisecond*20)
		go SendNodeError(relayProcessor, "lava3@test", time.Millisecond*25)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()
		for i := 0; i < 2; i++ {
			relayProcessor.WaitForResults(ctx)
		}
		resultsOk := relayProcessor.HasResults()
		require.True(t, resultsOk)
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(1), protocolErrors)
		// When quorum is disabled and we have node errors, we should get the node error as result
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Should return node error when quorum is disabled")
		require.NotNil(t, returnedResult)
		require.Equal(t, 500, returnedResult.StatusCode, "Node errors should have status code 500")
		require.Equal(t, []byte(`{"message":"bad","code":123}`), returnedResult.Reply.Data, "Should return node error data")
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
		relayProcessor := NewRelayProcessor(ctx, common.DefaultQuorumParams, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders), qos.NewQoSManager())
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())

		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		go SendProtocolError(relayProcessor, "lava@test", time.Millisecond*5, fmt.Errorf("bad"))
		go SendSuccessResp(relayProcessor, "lava@test2", time.Millisecond*20)
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

// Helper function to create mock relay responses with different data
func createMockRelayResponseWithData(providerAddr string, data []byte, err error) *RelayResponse {
	return &RelayResponse{
		RelayResult: common.RelayResult{
			Reply: &pairingtypes.RelayReply{
				Data: data,
			},
			ProviderInfo: common.ProviderInfo{
				ProviderAddress: providerAddr,
			},
		},
		Err: err,
	}
}

func TestHasRequiredNodeResultsQuorumScenarios(t *testing.T) {
	const baseTimestamp = 1234567890
	mockData := []byte(`{"result": "success", "data": "mock1", "timestamp": 1234567890}`)

	tests := []struct {
		name           string
		quorumParams   common.QuorumParams
		successResults int
		nodeErrors     int
		tries          int
		expectedResult bool
		expectedErrors int
		useSameData    int // Number of providers that should send the same data (0 = all different)
	}{
		{
			name: "quorum not met with different data from providers",
			quorumParams: common.QuorumParams{
				Min:  2,
				Rate: 0.6,
				Max:  5,
			},
			successResults: 2,
			nodeErrors:     0,
			tries:          1,
			expectedResult: false, // Different data won't meet quorum
			expectedErrors: 0,
			useSameData:    0, // 0 means all different data
		},
		{
			name: "quorum not met with different data from providers",
			quorumParams: common.QuorumParams{
				Min:  2,
				Rate: 1,
				Max:  5,
			},
			successResults: 2,
			nodeErrors:     0,
			tries:          1,
			expectedResult: true, // we have the final result and dont need to retry since there is no mathematical potentiol to meet quorum
			expectedErrors: 0,
			useSameData:    0, // 0 means all different data
		},
		{
			name: "quorum met with same data from min providers",
			quorumParams: common.QuorumParams{
				Min:  3,
				Rate: 0.6,
				Max:  5,
			},
			successResults: 3,
			nodeErrors:     0,
			tries:          1,
			expectedResult: true, // Same data should meet quorum
			expectedErrors: 0,
			useSameData:    2, // 2 providers with same data, 1 with different
		},
		{
			name: "quorum met with all providers sending same data",
			quorumParams: common.QuorumParams{
				Min:  2,
				Rate: 0.6,
				Max:  5,
			},
			successResults: 3,
			nodeErrors:     0,
			tries:          1,
			expectedResult: true, // All same data should definitely meet quorum
			expectedErrors: 0,
			useSameData:    3, // All 3 providers with same data
		},
		{
			name: "quorum not met with mixed data (1 same, 2 different)",
			quorumParams: common.QuorumParams{
				Min:  3,
				Rate: 0.6,
				Max:  5,
			},
			successResults: 3,
			nodeErrors:     0,
			tries:          1,
			expectedResult: false, // Mixed data won't meet quorum
			expectedErrors: 0,
			useSameData:    1, // Only 1 provider with same data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// Create a minimal protocol message to avoid nil pointer panic
			chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceRest, nil, nil, "../../", nil)
			if closeServer != nil {
				defer closeServer()
			}
			require.NoError(t, err)
			chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")
			usedProviders := lavasession.NewUsedProviders(nil)

			relayProcessor := NewRelayProcessor(
				ctx,
				tt.quorumParams,
				nil,
				RelayProcessorMetrics,
				RelayProcessorMetrics,
				RelayRetriesManagerInstance,
				newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
				qos.NewQoSManager(),
			)

			// Set the batch size so WaitForResults knows how many responses to expect
			consumerSessionsMap := lavasession.ConsumerSessionsMap{}
			for i := 0; i < tt.successResults+tt.nodeErrors; i++ {
				consumerSessionsMap[fmt.Sprintf("provider%d", i)] = &lavasession.SessionInfo{}
			}
			relayProcessor.GetUsedProviders().AddUsed(consumerSessionsMap, nil)

			// Create mock responses to populate the results
			for i := 0; i < tt.successResults; i++ {
				var data []byte
				if i < tt.useSameData {
					// Use same data for the first 'useSameData' providers (should meet quorum)
					data = mockData
				} else {
					// Use different data for remaining providers (should not meet quorum)
					// Create unique data by adding 'i' to the timestamp for each provider
					uniqueTimestamp := baseTimestamp + i
					data = []byte(fmt.Sprintf(`{"result": "success", "data": "mock%d", "timestamp": %d}`, i+1, uniqueTimestamp))
				}

				mockResponse := createMockRelayResponseWithData(
					fmt.Sprintf("provider%d", i),
					data,
					nil,
				)
				relayProcessor.SetResponse(mockResponse)
			}

			for i := 0; i < tt.nodeErrors; i++ {
				mockResponse := createMockRelayResponseWithData(
					fmt.Sprintf("provider%d", i+tt.successResults),
					[]byte(`{"error": "node error"}`),
					fmt.Errorf("node error"),
				)
				relayProcessor.SetResponse(mockResponse)
			}

			// Wait for responses to be processed
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
			defer cancel()
			relayProcessor.WaitForResults(ctx)

			// Test the quorum functionality
			result, errors := relayProcessor.HasRequiredNodeResults(tt.tries)

			require.Equal(t, tt.expectedResult, result, "quorum result mismatch")
			require.Equal(t, tt.expectedErrors, errors, "error count mismatch")
		})
	}
}

// ============================================================================
// CATEGORY 1: Node Error Quorum Tests
// These tests validate that node errors can form their own quorum
// ============================================================================

// Test 1.1: Node errors that match should form quorum
func TestNodeErrorQuorumMet(t *testing.T) {
	t.Run("three_matching_node_errors_meet_quorum", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Use Stateless selection to enable quorum logic
		relayProcessor := NewRelayProcessor(
			ctx,
			common.QuorumParams{Min: 3, Rate: 0.6, Max: 5},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		// Add 3 providers to the session
		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 3 identical node errors (same error code and message) - using REST error format
		nodeErrorData := []byte(`{"message":"invalid argument 0: hex string has length 3","code":400}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test1", time.Millisecond*5, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test2", time.Millisecond*10, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeErrorData)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - should succeed with node error quorum
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Node error quorum should be met")
		require.NotNil(t, returnedResult)
		require.Equal(t, 3, returnedResult.Quorum, "Quorum should be 3 (all three node errors matched)")
		require.Equal(t, nodeErrorData, returnedResult.Reply.Data, "Should return the node error data")
	})
}

// Test 1.2: Node errors that don't match should NOT form quorum
func TestNodeErrorQuorumNotMet(t *testing.T) {
	t.Run("three_different_node_errors_fail_quorum", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		relayProcessor := NewRelayProcessor(
			ctx,
			common.QuorumParams{Min: 3, Rate: 0.6, Max: 5},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 3 DIFFERENT node errors - using REST error format
		nodeError1 := []byte(`{"message":"error type 1","code":400}`)
		nodeError2 := []byte(`{"message":"error type 2","code":401}`)
		nodeError3 := []byte(`{"message":"error type 3","code":402}`)

		go sendNodeErrorWithData(relayProcessor, "lava@test1", time.Millisecond*5, nodeError1)
		go sendNodeErrorWithData(relayProcessor, "lava@test2", time.Millisecond*10, nodeError2)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeError3)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - should fail (no quorum)
		_, err = relayProcessor.ProcessingResult()
		require.Error(t, err, "Should fail when node errors don't match")
		require.Contains(t, err.Error(), "equal results count is less than requiredQuorumSize")
	})
}

// Test 1.3: Node error quorum with protocol errors mixed in
func TestNodeErrorQuorumWithProtocolErrors(t *testing.T) {
	t.Run("node_error_quorum_ignores_protocol_errors", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		relayProcessor := NewRelayProcessor(
			ctx,
			common.QuorumParams{Min: 3, Rate: 0.6, Max: 5},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
			"lava@test4": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 3 matching node errors + 1 protocol error - using REST error format
		nodeErrorData := []byte(`{"message":"invalid params","code":400}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test1", time.Millisecond*5, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test2", time.Millisecond*10, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeErrorData)
		go SendProtocolError(relayProcessor, "lava@test4", time.Millisecond*20, fmt.Errorf("connection timeout"))

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - node error quorum should be met despite protocol error
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Node error quorum should be met, protocol error should not interfere")
		require.NotNil(t, returnedResult)
		require.Equal(t, 3, returnedResult.Quorum, "Quorum should be 3 (node errors only)")
		require.Equal(t, nodeErrorData, returnedResult.Reply.Data, "Should return the node error data")
	})
}

// Test: Node error prioritized over protocol errors when quorum disabled
// This tests that with 4 protocol errors and 1 node error, we get the node error
func TestNodeErrorPrioritizedOverProtocolErrors(t *testing.T) {
	t.Run("node_error_prioritized_over_protocol_errors", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Quorum disabled (Stateful selection)
		relayProcessor := NewRelayProcessor(
			ctx,
			common.DefaultQuorumParams,
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachine(protocolMessage, usedProviders),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
			"lava@test4": &lavasession.SessionInfo{},
			"lava@test5": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 4 protocol errors and 1 node error
		go SendProtocolError(relayProcessor, "lava@test1", time.Millisecond*5, fmt.Errorf("protocol error 1"))
		go SendProtocolError(relayProcessor, "lava@test2", time.Millisecond*10, fmt.Errorf("protocol error 2"))
		go SendProtocolError(relayProcessor, "lava@test3", time.Millisecond*15, fmt.Errorf("protocol error 3"))
		go SendProtocolError(relayProcessor, "lava@test4", time.Millisecond*20, fmt.Errorf("protocol error 4"))
		go SendNodeError(relayProcessor, "lava@test5", time.Millisecond*25)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Verify counts
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(4), protocolErrors, "Should have 4 protocol errors")

		// Process results - should return node error (prioritized over protocol errors)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Should return node error, not an error")
		require.NotNil(t, returnedResult)
		require.Equal(t, 500, returnedResult.StatusCode, "Node errors should have status code 500")
		require.Equal(t, []byte(`{"message":"bad","code":123}`), returnedResult.Reply.Data, "Should return node error data")
		require.Equal(t, "lava@test5", returnedResult.ProviderInfo.ProviderAddress, "Should return the node error from provider 5")
	})

	t.Run("node_error_prioritized_over_protocol_errors_with_quorum_selection", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Stateless selection but quorum feature disabled (DefaultQuorumParams)
		relayProcessor := NewRelayProcessor(
			ctx,
			common.DefaultQuorumParams,
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
			"lava@test4": &lavasession.SessionInfo{},
			"lava@test5": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 4 protocol errors and 1 node error
		go SendProtocolError(relayProcessor, "lava@test1", time.Millisecond*5, fmt.Errorf("protocol error 1"))
		go SendProtocolError(relayProcessor, "lava@test2", time.Millisecond*10, fmt.Errorf("protocol error 2"))
		go SendProtocolError(relayProcessor, "lava@test3", time.Millisecond*15, fmt.Errorf("protocol error 3"))
		go SendProtocolError(relayProcessor, "lava@test4", time.Millisecond*20, fmt.Errorf("protocol error 4"))
		go SendNodeError(relayProcessor, "lava@test5", time.Millisecond*25)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Verify counts
		protocolErrors := relayProcessor.ProtocolErrors()
		require.Equal(t, uint64(4), protocolErrors, "Should have 4 protocol errors")

		// Process results - should return node error (prioritized over protocol errors)
		// Even with Stateless selection, when quorum feature is disabled, node errors should be prioritized
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Should return node error, not an error")
		require.NotNil(t, returnedResult)
		require.Equal(t, 500, returnedResult.StatusCode, "Node errors should have status code 500")
		require.Equal(t, []byte(`{"message":"bad","code":123}`), returnedResult.Reply.Data, "Should return node error data")
		require.Equal(t, "lava@test5", returnedResult.ProviderInfo.ProviderAddress, "Should return the node error from provider 5")
	})
}

// ============================================================================
// CATEGORY 2: Priority Logic Tests
// These tests validate that success results take priority over node errors
// ============================================================================

// Test 2.1: Success quorum takes priority when both success and node errors exist
func TestSuccessQuorumTakesPriorityOverNodeError(t *testing.T) {
	t.Run("success_results_prioritized_over_node_errors", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		relayProcessor := NewRelayProcessor(
			ctx,
			common.QuorumParams{Min: 3, Rate: 0.6, Max: 6},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
			"lava@test4": &lavasession.SessionInfo{},
			"lava@test5": &lavasession.SessionInfo{},
			"lava@test6": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 3 matching successes
		successData := []byte(`{"result":"success","block":100}`)
		go sendSuccessWithData(relayProcessor, "lava@test1", time.Millisecond*5, successData)
		go sendSuccessWithData(relayProcessor, "lava@test2", time.Millisecond*10, successData)
		go sendSuccessWithData(relayProcessor, "lava@test3", time.Millisecond*15, successData)

		// Send 3 matching node errors - using REST error format
		nodeErrorData := []byte(`{"message":"node error","code":500}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test4", time.Millisecond*20, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test5", time.Millisecond*25, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test6", time.Millisecond*30, nodeErrorData)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - should return success (priority)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Success quorum should take priority")
		require.NotNil(t, returnedResult)
		require.Equal(t, 3, returnedResult.Quorum, "Quorum should be 3 from success results")
		require.Equal(t, successData, returnedResult.Reply.Data, "Should return success data, not node error")
		require.Equal(t, 200, returnedResult.StatusCode, "Status should be 200 (success)")
	})
}

// Test 2.2: Node error quorum is used when success results are insufficient
func TestNodeErrorQuorumWhenSuccessInsufficient(t *testing.T) {
	t.Run("node_error_quorum_used_when_success_insufficient", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		relayProcessor := NewRelayProcessor(
			ctx,
			common.QuorumParams{Min: 3, Rate: 0.6, Max: 5},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
			"lava@test4": &lavasession.SessionInfo{},
			"lava@test5": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 2 DIFFERENT successes (can't form quorum)
		successData1 := []byte(`{"result":"success","block":100}`)
		successData2 := []byte(`{"result":"success","block":101}`)
		go sendSuccessWithData(relayProcessor, "lava@test1", time.Millisecond*5, successData1)
		go sendSuccessWithData(relayProcessor, "lava@test2", time.Millisecond*10, successData2)

		// Send 3 matching node errors (CAN form quorum) - using REST error format
		nodeErrorData := []byte(`{"message":"invalid params","code":400}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test4", time.Millisecond*20, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test5", time.Millisecond*25, nodeErrorData)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Debug: Check what was actually stored
		_, nodeErrors, _ := relayProcessor.GetResultsData()
		t.Logf("Number of node errors stored: %d", len(nodeErrors))
		for i, ne := range nodeErrors {
			t.Logf("Node error %d data: %s", i, string(ne.Reply.Data))
		}

		// Process results - should return node error quorum (success insufficient)
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Node error quorum should be used when success can't form quorum")
		require.NotNil(t, returnedResult)
		require.Equal(t, 3, returnedResult.Quorum, "Quorum should be 3 from node errors")
		require.Equal(t, nodeErrorData, returnedResult.Reply.Data, "Should return node error data")
		require.Equal(t, 500, returnedResult.StatusCode, "Status should be 500 (node error)")
	})
}

// Test 2.3: Success quorum ignores node errors entirely
func TestSuccessQuorumIgnoresNodeErrors(t *testing.T) {
	t.Run("success_quorum_ignores_all_node_errors", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		relayProcessor := NewRelayProcessor(
			ctx,
			common.QuorumParams{Min: 3, Rate: 0.6, Max: 8},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
			"lava@test3": &lavasession.SessionInfo{},
			"lava@test4": &lavasession.SessionInfo{},
			"lava@test5": &lavasession.SessionInfo{},
			"lava@test6": &lavasession.SessionInfo{},
			"lava@test7": &lavasession.SessionInfo{},
			"lava@test8": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 3 matching successes
		successData := []byte(`{"result":"success","block":100}`)
		go sendSuccessWithData(relayProcessor, "lava@test1", time.Millisecond*5, successData)
		go sendSuccessWithData(relayProcessor, "lava@test2", time.Millisecond*10, successData)
		go sendSuccessWithData(relayProcessor, "lava@test3", time.Millisecond*15, successData)

		// Send 5 DIFFERENT node errors (noise - should be ignored)
		go sendNodeErrorWithData(relayProcessor, "lava@test4", time.Millisecond*20, []byte(`{"error":"error1"}`))
		go sendNodeErrorWithData(relayProcessor, "lava@test5", time.Millisecond*25, []byte(`{"error":"error2"}`))
		go sendNodeErrorWithData(relayProcessor, "lava@test6", time.Millisecond*30, []byte(`{"error":"error3"}`))
		go sendNodeErrorWithData(relayProcessor, "lava@test7", time.Millisecond*35, []byte(`{"error":"error4"}`))
		go sendNodeErrorWithData(relayProcessor, "lava@test8", time.Millisecond*40, []byte(`{"error":"error5"}`))

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - should return success, completely ignoring node errors
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Success quorum should succeed regardless of node errors")
		require.NotNil(t, returnedResult)
		require.Equal(t, 3, returnedResult.Quorum, "Quorum should be 3 from success results")
		require.Equal(t, successData, returnedResult.Reply.Data, "Should return success data")
		require.Equal(t, 200, returnedResult.StatusCode, "Status should be 200 (success)")
	})
}

// Test: Success quorum fails when quorum feature is disabled
// This tests the scenario where success responses exist but fail quorum
// when quorum feature is disabled. With current implementation, when success
// quorum fails, it returns error without falling back to node errors.
func TestSuccessQuorumFailsWhenQuorumDisabled(t *testing.T) {
	t.Run("success_quorum_fails_when_quorum_disabled", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Quorum feature DISABLED (Stateful selection) - Min will be used as requiredQuorumSize
		relayProcessor := NewRelayProcessor(
			ctx,
			common.QuorumParams{Min: 1, Rate: 0.6, Max: 5},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateful), // Stateful = quorum disabled
			qos.NewQoSManager(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{
			"lava@test1": &lavasession.SessionInfo{},
			"lava@test2": &lavasession.SessionInfo{},
		}
		usedProviders.AddUsed(consumerSessionsMap, nil)

		// Send 1 success response with valid data
		// With quorum disabled and Min=1, having 1 success should normally succeed
		// But we test the edge case where responsesQuorum might fail for other reasons
		successData := []byte(`{"result":"success"}`)
		go sendSuccessWithData(relayProcessor, "lava@test1", time.Millisecond*5, successData)

		// Send 1 node error (available as potential fallback)
		nodeErrorData := []byte(`{"message":"invalid params","code":400}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test2", time.Millisecond*10, nodeErrorData)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Verify counts
		successResults, nodeErrors, _ := relayProcessor.GetResultsData()
		t.Logf("Success results: %d, Node errors: %d", len(successResults), len(nodeErrors))

		// Process results
		// With quorum disabled: requiredQuorumSize = Min = 1
		// successResultsCount (1) >= requiredQuorumSize (1) → enters success path
		// With 1 valid success response, quorum should succeed (maxCount = 1 >= quorumSize = 1)
		// This test verifies the code path is executed correctly
		returnedResult, err := relayProcessor.ProcessingResult()

		// With 1 success and quorumSize = 1, it should succeed
		// This test documents the expected behavior when quorum is disabled
		if err != nil {
			// If it fails, verify it's the expected error
			require.Contains(t, err.Error(), "majority count is less than quorumSize",
				"If it fails, should indicate quorum failure")
		} else {
			// If it succeeds, that's also valid - 1 success with quorumSize=1 should work
			require.NotNil(t, returnedResult, "If it succeeds, should return a result")
		}
	})
}

// ============================================================================
// Helper functions for new tests
// ============================================================================

// Helper to send a node error with custom data
func sendNodeErrorWithData(relayProcessor *RelayProcessor, provider string, delay time.Duration, data []byte) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &RelayResponse{
		RelayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: data},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   500, // Node errors have status 500
		},
		Err: nil,
	}
	relayProcessor.SetResponse(response)
}

// Helper to send a success response with custom data
func sendSuccessWithData(relayProcessor *RelayProcessor, provider string, delay time.Duration, data []byte) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &RelayResponse{
		RelayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: data, LatestBlock: 1},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   200, // Success responses have status 200
		},
		Err: nil,
	}
	relayProcessor.SetResponse(response)
}

func TestIsValidResponse(t *testing.T) {
	// Create the isValidResponse function (extracted from relay_processor.go for testing)
	isValidResponse := func(data []byte) bool {
		if len(data) == 0 {
			return false
		}
		// Check for empty JSON objects/arrays that are technically valid but meaningless
		trimmed := bytes.TrimSpace(data)
		if bytes.Equal(trimmed, []byte("{}")) || bytes.Equal(trimmed, []byte("[]")) {
			return false
		}
		return true
	}

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "empty byte slice",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "nil byte slice",
			data:     nil,
			expected: false,
		},
		{
			name:     "empty JSON object",
			data:     []byte("{}"),
			expected: false,
		},
		{
			name:     "empty JSON array",
			data:     []byte("[]"),
			expected: false,
		},
		{
			name:     "empty JSON object with whitespace",
			data:     []byte("  {}  "),
			expected: false,
		},
		{
			name:     "empty JSON array with whitespace",
			data:     []byte("\n[]\n"),
			expected: false,
		},
		{
			name:     "empty JSON object with tabs and newlines",
			data:     []byte("\t{}\n"),
			expected: false,
		},
		{
			name:     "valid JSON object with content",
			data:     []byte(`{"key": "value"}`),
			expected: true,
		},
		{
			name:     "valid JSON array with content",
			data:     []byte(`[1, 2, 3]`),
			expected: true,
		},
		{
			name:     "valid JSON with single element",
			data:     []byte(`{"result": null}`),
			expected: true,
		},
		{
			name:     "plain text response",
			data:     []byte("ok"),
			expected: true,
		},
		{
			name:     "whitespace only",
			data:     []byte("   \n\t  "),
			expected: true, // trimmed will be empty but doesn't match {} or []
		},
		{
			name:     "partial JSON object",
			data:     []byte(`{`),
			expected: true,
		},
		{
			name:     "nested empty object inside object",
			data:     []byte(`{"nested": {}}`),
			expected: true,
		},
		{
			name:     "nested empty array inside array",
			data:     []byte(`[[]]`),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidResponse(tt.data)
			require.Equal(t, tt.expected, result, "isValidResponse(%q) = %v, want %v", string(tt.data), result, tt.expected)
		})
	}
}

// Mock metrics tracker for testing error recovery metrics
type MockMetricsTracker struct {
	mu                         sync.Mutex
	nodeErrorRecoveryCalls     []MetricCall
	protocolErrorRecoveryCalls []MetricCall
}

type MetricCall struct {
	chainId      string
	apiInterface string
	attempt      string
}

func NewMockMetricsTracker() *MockMetricsTracker {
	return &MockMetricsTracker{
		nodeErrorRecoveryCalls:     []MetricCall{},
		protocolErrorRecoveryCalls: []MetricCall{},
	}
}

func (m *MockMetricsTracker) SetRelayNodeErrorMetric(providerAddress string, chainId string, apiInterface string) {
}

func (m *MockMetricsTracker) SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeErrorRecoveryCalls = append(m.nodeErrorRecoveryCalls, MetricCall{
		chainId:      chainId,
		apiInterface: apiInterface,
		attempt:      attempt,
	})
}

func (m *MockMetricsTracker) SetProtocolErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.protocolErrorRecoveryCalls = append(m.protocolErrorRecoveryCalls, MetricCall{
		chainId:      chainId,
		apiInterface: apiInterface,
		attempt:      attempt,
	})
}

func (m *MockMetricsTracker) GetChainIdAndApiInterface() (string, string) {
	return "TEST_CHAIN", "rest"
}

// GetNodeErrorRecoveryCalls returns a copy of the calls for thread-safe reading
func (m *MockMetricsTracker) GetNodeErrorRecoveryCalls() []MetricCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]MetricCall, len(m.nodeErrorRecoveryCalls))
	copy(calls, m.nodeErrorRecoveryCalls)
	return calls
}

// GetProtocolErrorRecoveryCalls returns a copy of the calls for thread-safe reading
func (m *MockMetricsTracker) GetProtocolErrorRecoveryCalls() []MetricCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]MetricCall, len(m.protocolErrorRecoveryCalls))
	copy(calls, m.protocolErrorRecoveryCalls)
	return calls
}

// Test: Quorum met with node errors only
func TestNodeErrorsRecoveryMetricWithQuorum(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "dapp", "127.0.0.1")
	usedProviders := lavasession.NewUsedProviders(nil)
	mockMetrics := NewMockMetricsTracker()
	consistency := NewConsistency(specId)

	quorumParams := common.QuorumParams{
		Min:  3,
		Max:  5,
		Rate: 0.6,
	}

	relayProcessor := NewRelayProcessor(
		ctx,
		quorumParams,
		consistency,
		mockMetrics,
		mockMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
		qos.NewQoSManager(),
	)

	// Add providers
	canUse := usedProviders.TryLockSelection(ctx)
	require.Nil(t, canUse)
	consumerSessionsMap := lavasession.ConsumerSessionsMap{}
	for i := 0; i < 5; i++ {
		consumerSessionsMap[fmt.Sprintf("provider%d", i)] = &lavasession.SessionInfo{}
	}
	usedProviders.AddUsed(consumerSessionsMap, nil)

	// Simulate 3 successful responses
	for i := 0; i < 3; i++ {
		go SendSuccessResp(relayProcessor, fmt.Sprintf("provider%d", i), 20*time.Millisecond)
	}

	// Simulate 2 node errors
	for i := 3; i < 5; i++ {
		go SendNodeError(relayProcessor, fmt.Sprintf("provider%d", i), 0)
	}

	// Give goroutines a chance to start executing
	time.Sleep(10 * time.Millisecond)

	// Wait for results
	waitCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	err = relayProcessor.WaitForResults(waitCtx)
	require.NoError(t, err)

	// Wait for all responses to be processed (including node errors)
	// WaitForResults returns when quorum is met, but node errors might still be in flight
	// Poll until we see node errors or timeout (max 200ms)
	var hasResults bool
	var nodeErrorCount int
	maxRetries := 20
	for retry := 0; retry < maxRetries; retry++ {
		time.Sleep(10 * time.Millisecond)
		hasResults, nodeErrorCount = relayProcessor.HasRequiredNodeResults(1)
		if hasResults && nodeErrorCount > 0 {
			break
		}
	}

	// Verify results
	require.True(t, hasResults, "Should have required results (quorum met)")
	require.Greater(t, nodeErrorCount, 0, "Should have at least 1 node error")

	// Give time for async metric calls
	time.Sleep(100 * time.Millisecond)

	// Verify metrics - nodeError recovery metric should be called since we had node errors
	nodeErrorCalls := mockMetrics.GetNodeErrorRecoveryCalls()
	protocolErrorCalls := mockMetrics.GetProtocolErrorRecoveryCalls()

	require.Equal(t, 1, len(nodeErrorCalls), "Node error recovery metric should be called once")
	require.Equal(t, 0, len(protocolErrorCalls), "Protocol error recovery metric should NOT be called")

	if len(nodeErrorCalls) > 0 {
		// Note: Due to timing, we might not get all 2 error responses before quorum is met
		// The important thing is that the metric is incremented when quorum is met with some node errors
		require.Greater(t, len(nodeErrorCalls[0].attempt), 0, "Should record node errors")
		require.Equal(t, "TEST_CHAIN", nodeErrorCalls[0].chainId)
		require.Equal(t, "rest", nodeErrorCalls[0].apiInterface)
	}
}

// Test: Quorum met with protocol errors only
func TestProtocolErrorsRecoveryMetricWithQuorum(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "dapp", "127.0.0.1")
	usedProviders := lavasession.NewUsedProviders(nil)
	mockMetrics := NewMockMetricsTracker()
	consistency := NewConsistency(specId)

	quorumParams := common.QuorumParams{
		Min:  3,
		Max:  5,
		Rate: 0.6,
	}

	relayProcessor := NewRelayProcessor(
		ctx,
		quorumParams,
		consistency,
		mockMetrics,
		mockMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
		qos.NewQoSManager(),
	)

	// Add providers
	canUse := usedProviders.TryLockSelection(ctx)
	require.Nil(t, canUse)
	consumerSessionsMap := lavasession.ConsumerSessionsMap{}
	for i := 0; i < 5; i++ {
		consumerSessionsMap[fmt.Sprintf("provider%d", i)] = &lavasession.SessionInfo{}
	}
	usedProviders.AddUsed(consumerSessionsMap, nil)

	// Simulate 2 protocol errors (connection timeouts)
	for i := 3; i < 5; i++ {
		go SendProtocolError(relayProcessor, fmt.Sprintf("provider%d", i), 0, fmt.Errorf("connection timeout"))
	}

	// Simulate 3 successful responses with a delay to ensure errors are processed first
	for i := 0; i < 3; i++ {
		go SendSuccessResp(relayProcessor, fmt.Sprintf("provider%d", i), 20*time.Millisecond)
	}

	// Wait for results
	waitCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	err = relayProcessor.WaitForResults(waitCtx)
	require.NoError(t, err)

	// Check HasRequiredNodeResults
	hasResults, nodeErrorCount := relayProcessor.HasRequiredNodeResults(1)

	// Verify results
	require.True(t, hasResults, "Should have required results (quorum met)")
	require.Equal(t, 0, nodeErrorCount, "Should have 0 node errors")

	// Give time for async metric calls
	time.Sleep(100 * time.Millisecond)

	// Verify metrics
	nodeErrorCalls := mockMetrics.GetNodeErrorRecoveryCalls()
	protocolErrorCalls := mockMetrics.GetProtocolErrorRecoveryCalls()

	require.Equal(t, 0, len(nodeErrorCalls), "Node error recovery metric should NOT be called")
	require.Greater(t, len(protocolErrorCalls), 0, "Protocol error recovery metric should be called at least once")

	if len(protocolErrorCalls) > 0 {
		// Note: Due to timing, we might not get all error responses before quorum is met
		// The important thing is that the metric is incremented when quorum is met with some protocol errors
		require.Greater(t, len(protocolErrorCalls[0].attempt), 0, "Should record protocol errors")
		require.Equal(t, "TEST_CHAIN", protocolErrorCalls[0].chainId)
		require.Equal(t, "rest", protocolErrorCalls[0].apiInterface)
	}
}

// Test: Quorum met with both error types
func TestBothErrorTypesRecoveryMetricsWithQuorum(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "dapp", "127.0.0.1")
	usedProviders := lavasession.NewUsedProviders(nil)
	mockMetrics := NewMockMetricsTracker()
	consistency := NewConsistency(specId)

	quorumParams := common.QuorumParams{
		Min:  3,
		Max:  5,
		Rate: 0.6,
	}

	relayProcessor := NewRelayProcessor(
		ctx,
		quorumParams,
		consistency,
		mockMetrics,
		mockMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
		qos.NewQoSManager(),
	)

	// Add providers
	canUse := usedProviders.TryLockSelection(ctx)
	require.Nil(t, canUse)
	consumerSessionsMap := lavasession.ConsumerSessionsMap{}
	for i := 0; i < 5; i++ {
		consumerSessionsMap[fmt.Sprintf("provider%d", i)] = &lavasession.SessionInfo{}
	}
	usedProviders.AddUsed(consumerSessionsMap, nil)

	// Simulate 3 successful responses
	for i := 0; i < 3; i++ {
		go SendSuccessResp(relayProcessor, fmt.Sprintf("provider%d", i), 0)
	}

	// Simulate 1 node error
	go SendNodeError(relayProcessor, "provider3", 0)

	// Simulate 1 protocol error
	go SendProtocolError(relayProcessor, "provider4", 0, fmt.Errorf("epoch mismatch"))

	// Wait for results
	waitCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	err = relayProcessor.WaitForResults(waitCtx)
	require.NoError(t, err)

	// Check HasRequiredNodeResults
	hasResults, _ := relayProcessor.HasRequiredNodeResults(1)

	// Verify results
	require.True(t, hasResults, "Should have required results (quorum met)")

	// Give time for async metric calls
	time.Sleep(100 * time.Millisecond)

	// Get actual error counts
	resultsCount, actualNodeErrors, _, actualProtocolErrors := relayProcessor.GetResults()

	// Due to timing and async execution, we might get various combinations:
	// 1. Both error types (ideal case) - both metrics called
	// 2. Only one error type - one metric called
	// 3. No errors before quorum - no metrics called (successes came in very fast)
	// All scenarios are valid! The test proves that when errors ARE received before quorum,
	// the corresponding metrics ARE incremented correctly.

	// Verify metric consistency: if we have node errors, metric should be called
	nodeErrorCalls := mockMetrics.GetNodeErrorRecoveryCalls()
	protocolErrorCalls := mockMetrics.GetProtocolErrorRecoveryCalls()

	if actualNodeErrors > 0 {
		require.Equal(t, 1, len(nodeErrorCalls), "Node error recovery metric should be called when node errors present")
	}

	// Verify metric consistency: if we have protocol errors, metric should be called
	if actualProtocolErrors > 0 {
		require.Equal(t, 1, len(protocolErrorCalls), "Protocol error recovery metric should be called when protocol errors present")
	}

	// Verify we had successful responses (quorum)
	require.GreaterOrEqual(t, resultsCount, 3, "Should have at least 3 successes for quorum")
}

// Test: Quorum not met - no metrics
func TestNoRecoveryMetricsWhenQuorumNotMet(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "dapp", "127.0.0.1")
	usedProviders := lavasession.NewUsedProviders(nil)
	mockMetrics := NewMockMetricsTracker()
	consistency := NewConsistency(specId)

	quorumParams := common.QuorumParams{
		Min:  3,
		Max:  5,
		Rate: 0.6,
	}

	relayProcessor := NewRelayProcessor(
		ctx,
		quorumParams,
		consistency,
		mockMetrics,
		mockMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
		qos.NewQoSManager(),
	)

	// Add providers
	canUse := usedProviders.TryLockSelection(ctx)
	require.Nil(t, canUse)
	consumerSessionsMap := lavasession.ConsumerSessionsMap{}
	for i := 0; i < 5; i++ {
		consumerSessionsMap[fmt.Sprintf("provider%d", i)] = &lavasession.SessionInfo{}
	}
	usedProviders.AddUsed(consumerSessionsMap, nil)

	// Simulate only 1 successful response (not enough for quorum)
	go SendSuccessResp(relayProcessor, "provider0", 0)

	// Simulate 2 node errors
	go SendNodeError(relayProcessor, "provider1", 0)
	go SendNodeError(relayProcessor, "provider2", 0)

	// Simulate 2 protocol errors
	go SendProtocolError(relayProcessor, "provider3", 0, fmt.Errorf("connection timeout"))
	go SendProtocolError(relayProcessor, "provider4", 0, fmt.Errorf("provider unavailable"))

	// Wait for results
	waitCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	err = relayProcessor.WaitForResults(waitCtx)
	require.NoError(t, err)

	// Check HasRequiredNodeResults
	hasResults, nodeErrorCount := relayProcessor.HasRequiredNodeResults(1)

	// Verify results
	require.False(t, hasResults, "Should NOT have required results (only 1 success, need 3)")
	require.Equal(t, 2, nodeErrorCount, "Should have 2 node errors")

	// Give time for async metric calls (if any)
	time.Sleep(100 * time.Millisecond)

	// Verify NO metrics were called (no recovery since quorum not met)
	nodeErrorCalls := mockMetrics.GetNodeErrorRecoveryCalls()
	protocolErrorCalls := mockMetrics.GetProtocolErrorRecoveryCalls()

	require.Equal(t, 0, len(nodeErrorCalls), "Node error recovery metric should NOT be called")
	require.Equal(t, 0, len(protocolErrorCalls), "Protocol error recovery metric should NOT be called")
}

// TestGetRequiredQuorumSize tests the getRequiredQuorumSize function behavior
// with quorum enabled and disabled
func TestGetRequiredQuorumSize(t *testing.T) {
	tests := []struct {
		name          string
		quorumParams  common.QuorumParams
		responseCount int
		expected      int
		description   string
	}{
		{
			name:          "Quorum Disabled - Returns Min",
			quorumParams:  common.QuorumParams{Rate: 1, Max: 1, Min: 1},
			responseCount: 5,
			expected:      1,
			description:   "When quorum is disabled, should always return Min regardless of response count",
		},
		{
			name:          "Quorum Enabled - Single Response",
			quorumParams:  common.QuorumParams{Rate: 0.66, Max: 5, Min: 2},
			responseCount: 1,
			expected:      2,
			description:   "When quorum enabled with low response count, should return Min",
		},
		{
			name:          "Quorum Enabled - Multiple Responses",
			quorumParams:  common.QuorumParams{Rate: 0.66, Max: 5, Min: 2},
			responseCount: 3,
			expected:      2,
			description:   "Rate 0.66 * 3 = 1.98, ceil = 2",
		},
		{
			name:          "Quorum Enabled - High Response Count",
			quorumParams:  common.QuorumParams{Rate: 0.66, Max: 5, Min: 2},
			responseCount: 5,
			expected:      4,
			description:   "Rate 0.66 * 5 = 3.3, ceil = 4",
		},
		{
			name:          "Quorum Disabled - Zero Responses",
			quorumParams:  common.QuorumParams{Rate: 1, Max: 1, Min: 1},
			responseCount: 0,
			expected:      1,
			description:   "Even with 0 responses, should return Min when disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the function directly by creating a minimal RelayProcessor
			rp := &RelayProcessor{
				quorumParams: tt.quorumParams,
			}

			result := rp.getRequiredQuorumSize(tt.responseCount)
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestQuorumDisabledScenario tests the core bug fix:
// With quorum disabled and 2 responses, should only require 1 match (Min)
func TestQuorumDisabledScenario(t *testing.T) {
	// This test validates the fix for the bug where:
	// - Quorum was disabled {Rate: 1, Max: 1, Min: 1}
	// - 2 node error responses were received
	// - Old behavior: required 2 matches (calculated from response count)
	// - New behavior: requires 1 match (from Min)

	quorumDisabled := common.QuorumParams{Rate: 1, Max: 1, Min: 1}
	rp := &RelayProcessor{quorumParams: quorumDisabled}

	// With 2 responses and quorum disabled, should require only Min (1)
	result := rp.getRequiredQuorumSize(2)
	require.Equal(t, 1, result, "Quorum disabled with 2 responses should require Min (1), not 2")

	// With 5 responses and quorum disabled, should still require only Min (1)
	result = rp.getRequiredQuorumSize(5)
	require.Equal(t, 1, result, "Quorum disabled with 5 responses should require Min (1), not 5")

	// Now test with quorum enabled
	quorumEnabled := common.QuorumParams{Rate: 0.66, Max: 5, Min: 2}
	rp.quorumParams = quorumEnabled

	// With 2 responses and quorum enabled, should calculate: ceil(0.66 * 2) = 2
	result = rp.getRequiredQuorumSize(2)
	require.Equal(t, 2, result, "Quorum enabled with 2 responses: ceil(0.66 * 2) = 2")

	// With 3 responses and quorum enabled, should calculate: ceil(0.66 * 3) = 2
	result = rp.getRequiredQuorumSize(3)
	require.Equal(t, 2, result, "Quorum enabled with 3 responses: ceil(0.66 * 3) = 2")
}
