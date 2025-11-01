package relaycore

import (
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
