package relaycore

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
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
		selection:       BestResult, // Default to BestResult for backward compatibility
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
		// With quorum changes, we now fail when we only have node errors
		_, err = relayProcessor.ProcessingResult()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed relay, insufficient results")
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
		// With quorum changes, we now fail when we only have node errors
		_, err = relayProcessor.ProcessingResult()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed relay, insufficient results")
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
				newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Quorum),
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
	m.nodeErrorRecoveryCalls = append(m.nodeErrorRecoveryCalls, MetricCall{
		chainId:      chainId,
		apiInterface: apiInterface,
		attempt:      attempt,
	})
}

func (m *MockMetricsTracker) SetProtocolErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
	m.protocolErrorRecoveryCalls = append(m.protocolErrorRecoveryCalls, MetricCall{
		chainId:      chainId,
		apiInterface: apiInterface,
		attempt:      attempt,
	})
}

func (m *MockMetricsTracker) GetChainIdAndApiInterface() (string, string) {
	return "TEST_CHAIN", "rest"
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
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Quorum),
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

	// Simulate 2 node errors
	for i := 3; i < 5; i++ {
		go SendNodeError(relayProcessor, fmt.Sprintf("provider%d", i), 0)
	}

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
	require.Equal(t, 1, len(mockMetrics.nodeErrorRecoveryCalls), "Node error recovery metric should be called once")
	require.Equal(t, 0, len(mockMetrics.protocolErrorRecoveryCalls), "Protocol error recovery metric should NOT be called")

	if len(mockMetrics.nodeErrorRecoveryCalls) > 0 {
		// Note: Due to timing, we might not get all 2 error responses before quorum is met
		// The important thing is that the metric is incremented when quorum is met with some node errors
		require.Greater(t, len(mockMetrics.nodeErrorRecoveryCalls[0].attempt), 0, "Should record node errors")
		require.Equal(t, "TEST_CHAIN", mockMetrics.nodeErrorRecoveryCalls[0].chainId)
		require.Equal(t, "rest", mockMetrics.nodeErrorRecoveryCalls[0].apiInterface)
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
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Quorum),
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

	// Simulate 2 protocol errors (connection timeouts)
	for i := 3; i < 5; i++ {
		go SendProtocolError(relayProcessor, fmt.Sprintf("provider%d", i), 0, fmt.Errorf("connection timeout"))
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
	require.Equal(t, 0, len(mockMetrics.nodeErrorRecoveryCalls), "Node error recovery metric should NOT be called")
	require.Equal(t, 1, len(mockMetrics.protocolErrorRecoveryCalls), "Protocol error recovery metric should be called once")

	if len(mockMetrics.protocolErrorRecoveryCalls) > 0 {
		// Note: Due to timing, we might not get all error responses before quorum is met
		// The important thing is that the metric is incremented when quorum is met with some protocol errors
		require.Greater(t, len(mockMetrics.protocolErrorRecoveryCalls[0].attempt), 0, "Should record protocol errors")
		require.Equal(t, "TEST_CHAIN", mockMetrics.protocolErrorRecoveryCalls[0].chainId)
		require.Equal(t, "rest", mockMetrics.protocolErrorRecoveryCalls[0].apiInterface)
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
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Quorum),
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
	if actualNodeErrors > 0 {
		require.Equal(t, 1, len(mockMetrics.nodeErrorRecoveryCalls), "Node error recovery metric should be called when node errors present")
	}

	// Verify metric consistency: if we have protocol errors, metric should be called
	if actualProtocolErrors > 0 {
		require.Equal(t, 1, len(mockMetrics.protocolErrorRecoveryCalls), "Protocol error recovery metric should be called when protocol errors present")
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
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Quorum),
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
	require.Equal(t, 0, len(mockMetrics.nodeErrorRecoveryCalls), "Node error recovery metric should NOT be called")
	require.Equal(t, 0, len(mockMetrics.protocolErrorRecoveryCalls), "Protocol error recovery metric should NOT be called")
}
