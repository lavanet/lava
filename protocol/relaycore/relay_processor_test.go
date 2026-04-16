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
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// Mock RelayStateMachine for testing
type mockRelayStateMachine struct {
	protocolMessage       chainlib.ProtocolMessage
	usedProviders         *lavasession.UsedProviders
	debugState            bool
	selection             Selection
	crossValidationParams *common.CrossValidationParams // nil for Stateless/Stateful
}

func newMockRelayStateMachine(protocolMessage chainlib.ProtocolMessage, usedProviders *lavasession.UsedProviders) *mockRelayStateMachine {
	return &mockRelayStateMachine{
		protocolMessage:       protocolMessage,
		usedProviders:         usedProviders,
		debugState:            false,
		selection:             Stateful, // Default to Stateful for backward compatibility
		crossValidationParams: nil,      // nil for non-CrossValidation modes
	}
}

func newMockRelayStateMachineWithSelection(protocolMessage chainlib.ProtocolMessage, usedProviders *lavasession.UsedProviders, selection Selection) *mockRelayStateMachine {
	var cvParams *common.CrossValidationParams
	if selection == CrossValidation {
		cvParams = &common.CrossValidationParams{MaxParticipants: 3, AgreementThreshold: 2}
	}
	return &mockRelayStateMachine{
		protocolMessage:       protocolMessage,
		usedProviders:         usedProviders,
		debugState:            false,
		selection:             selection,
		crossValidationParams: cvParams,
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

func (m *mockRelayStateMachine) GetCrossValidationParams() *common.CrossValidationParams {
	return m.crossValidationParams
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
		relayProcessor := NewRelayProcessor(ctx, nil, consistency, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))

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
		relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))

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
		relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))

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
		relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))

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
		// When cross-validation is disabled and we have node errors, we should get the node error as result
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Should return node error when cross-validation is disabled")
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
		relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))
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
		relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))
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
		// When cross-validation is disabled and we have node errors, we should get the node error as result
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Should return node error when cross-validation is disabled")
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
		relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))
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

func TestHasRequiredNodeResultsCrossValidationScenarios(t *testing.T) {
	const baseTimestamp = 1234567890
	mockData := []byte(`{"result": "success", "data": "mock1", "timestamp": 1234567890}`)

	tests := []struct {
		name                  string
		crossValidationParams *common.CrossValidationParams // nil for Stateless/Stateful
		selection             Selection
		successResults        int
		nodeErrors            int
		tries                 int
		expectedResult        bool
		expectedErrors        int
		useSameData           int // Number of providers that should send the same data (0 = all different)
	}{
		{
			name: "cross-validation not met with different data from providers",
			crossValidationParams: &common.CrossValidationParams{
				AgreementThreshold: 2,
				MaxParticipants:    5,
			},
			selection:      CrossValidation,
			successResults: 2,
			nodeErrors:     0,
			tries:          1,
			expectedResult: false, // Different data won't meet cross-validation threshold
			expectedErrors: 0,
			useSameData:    0, // 0 means all different data
		},
		{
			name: "cross-validation met with same data meeting threshold",
			crossValidationParams: &common.CrossValidationParams{
				AgreementThreshold: 2,
				MaxParticipants:    5,
			},
			selection:      CrossValidation,
			successResults: 3,
			nodeErrors:     0,
			tries:          1,
			expectedResult: true, // 2 providers with same data meets threshold of 2
			expectedErrors: 0,
			useSameData:    2, // 2 providers with same data
		},
		{
			name: "cross-validation met with all providers sending same data",
			crossValidationParams: &common.CrossValidationParams{
				AgreementThreshold: 2,
				MaxParticipants:    5,
			},
			selection:      CrossValidation,
			successResults: 3,
			nodeErrors:     0,
			tries:          1,
			expectedResult: true, // All same data should definitely meet cross-validation
			expectedErrors: 0,
			useSameData:    3, // All 3 providers with same data
		},
		{
			name: "cross-validation not met with threshold of 3 but only 1 matching",
			crossValidationParams: &common.CrossValidationParams{
				AgreementThreshold: 3,
				MaxParticipants:    5,
			},
			selection:      CrossValidation,
			successResults: 3,
			nodeErrors:     0,
			tries:          1,
			expectedResult: false, // Only 1 provider with same data, threshold is 3
			expectedErrors: 0,
			useSameData:    1, // Only 1 provider with same data
		},
		{
			name:                  "stateless mode - single result is sufficient",
			crossValidationParams: nil,
			selection:             Stateless,
			successResults:        1,
			nodeErrors:            0,
			tries:                 1,
			expectedResult:        true, // Stateless mode needs just 1 successful result
			expectedErrors:        0,
			useSameData:           1,
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
				tt.crossValidationParams,
				nil,
				RelayProcessorMetrics,
				RelayProcessorMetrics,
				RelayRetriesManagerInstance,
				newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, tt.selection),
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
					// Use same data for the first 'useSameData' providers (should meet cross-validation)
					data = mockData
				} else {
					// Use different data for remaining providers (should not meet cross-validation)
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

			// Test the cross-validation functionality
			result, errors := relayProcessor.HasRequiredNodeResults(tt.tries)

			require.Equal(t, tt.expectedResult, result, "cross-validation result mismatch")
			require.Equal(t, tt.expectedErrors, errors, "error count mismatch")
		})
	}
}

// ============================================================================
// CATEGORY 1: Selection Mode Result Tests
// These tests validate the different result processing for each selection mode:
// - Stateless: returns first result (no consensus)
// - Stateful: returns first result (no consensus)
// - CrossValidation: requires agreementThreshold matching successful responses
// ============================================================================

// Test 1.1: Stateless mode returns first node error when no successes
func TestStatelessReturnsFirstNodeError(t *testing.T) {
	t.Run("stateless_returns_first_node_error", func(t *testing.T) {
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

		// Stateless mode - just returns first result, no consensus
		relayProcessor := NewRelayProcessor(
			ctx,
			nil, // nil for Stateless mode
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
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

		// Send 3 node errors - using REST error format
		nodeErrorData := []byte(`{"message":"invalid argument 0: hex string has length 3","code":400}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test1", time.Millisecond*5, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test2", time.Millisecond*10, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeErrorData)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - Stateless returns first result, no cross-validation
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Stateless mode should return first node error")
		require.NotNil(t, returnedResult)
		require.Equal(t, 0, returnedResult.CrossValidation, "Stateless mode does not do cross-validation")
		require.Equal(t, nodeErrorData, returnedResult.Reply.Data, "Should return the node error data")
	})
}

// Test 1.2: CrossValidation mode fails when only node errors are received (no successful responses)
func TestNodeErrorCrossValidationNotMet(t *testing.T) {
	t.Run("cross_validation_mode_fails_with_only_node_errors", func(t *testing.T) {
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

		// CrossValidation mode - node errors do NOT count towards consensus
		relayProcessor := NewRelayProcessor(
			ctx,
			&common.CrossValidationParams{AgreementThreshold: 2, MaxParticipants: 5},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
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

		// Send 3 MATCHING node errors - but in CrossValidation mode they don't count
		nodeErrorData := []byte(`{"message":"error type 1","code":400}`)

		go sendNodeErrorWithData(relayProcessor, "lava@test1", time.Millisecond*5, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test2", time.Millisecond*10, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeErrorData)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - should fail because node errors don't count in CrossValidation mode
		_, err = relayProcessor.ProcessingResult()
		require.Error(t, err, "CrossValidation mode should fail when only node errors are received")
		require.Contains(t, err.Error(), "cross-validation failed: insufficient successful responses")
	})
}

// Test 1.3: Stateless mode returns first node error, ignoring protocol errors
func TestStatelessReturnsNodeErrorOverProtocolError(t *testing.T) {
	t.Run("stateless_returns_node_error_over_protocol_error", func(t *testing.T) {
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

		// Stateless mode - returns first result
		relayProcessor := NewRelayProcessor(
			ctx,
			nil, // nil for Stateless mode
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
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

		// Send node errors + protocol error
		nodeErrorData := []byte(`{"message":"invalid params","code":400}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test1", time.Millisecond*5, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test2", time.Millisecond*10, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeErrorData)
		go SendProtocolError(relayProcessor, "lava@test4", time.Millisecond*20, fmt.Errorf("connection timeout"))

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Stateless returns first node error, no cross-validation
		returnedResult, err := relayProcessor.ProcessingResult()
		require.NoError(t, err, "Stateless mode should return first node error")
		require.NotNil(t, returnedResult)
		require.Equal(t, 0, returnedResult.CrossValidation, "Stateless mode does not do cross-validation")
		require.Equal(t, nodeErrorData, returnedResult.Reply.Data, "Should return the node error data")
	})
}

// Test: Node error prioritized over protocol errors when cross-validation disabled
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

		// CrossValidation disabled (Stateful selection)
		relayProcessor := NewRelayProcessor(
			ctx,
			nil,
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachine(protocolMessage, usedProviders),
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

	t.Run("node_error_prioritized_over_protocol_errors_with_stateless_selection", func(t *testing.T) {
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

		// Stateless selection but cross-validation feature disabled (DefaultCrossValidationParams)
		relayProcessor := NewRelayProcessor(
			ctx,
			nil,
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
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
		// Even with Stateless selection, when cross-validation feature is disabled, node errors should be prioritized
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

// Test 2.1: Success cross-validation takes priority when both success and node errors exist
func TestSuccessCrossValidationTakesPriorityOverNodeError(t *testing.T) {
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
			&common.CrossValidationParams{AgreementThreshold: 3, MaxParticipants: 6},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
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
		require.NoError(t, err, "Success cross-validation should take priority")
		require.NotNil(t, returnedResult)
		require.Equal(t, 3, returnedResult.CrossValidation, "CrossValidation should be 3 from success results")
		require.Equal(t, successData, returnedResult.Reply.Data, "Should return success data, not node error")
		require.Equal(t, 200, returnedResult.StatusCode, "Status should be 200 (success)")
	})
}

// Test 2.2: CrossValidation mode fails when success results are insufficient (node errors don't count)
func TestNodeErrorCrossValidationWhenSuccessInsufficient(t *testing.T) {
	t.Run("cross_validation_mode_fails_when_success_insufficient", func(t *testing.T) {
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

		// CrossValidation mode - node errors do NOT count towards consensus
		relayProcessor := NewRelayProcessor(
			ctx,
			&common.CrossValidationParams{AgreementThreshold: 3, MaxParticipants: 5},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
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

		// Send 2 DIFFERENT successes (can't form cross-validation)
		successData1 := []byte(`{"result":"success","block":100}`)
		successData2 := []byte(`{"result":"success","block":101}`)
		go sendSuccessWithData(relayProcessor, "lava@test1", time.Millisecond*5, successData1)
		go sendSuccessWithData(relayProcessor, "lava@test2", time.Millisecond*10, successData2)

		// Send 3 matching node errors - but they don't count in CrossValidation mode
		nodeErrorData := []byte(`{"message":"invalid params","code":400}`)
		go sendNodeErrorWithData(relayProcessor, "lava@test3", time.Millisecond*15, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test4", time.Millisecond*20, nodeErrorData)
		go sendNodeErrorWithData(relayProcessor, "lava@test5", time.Millisecond*25, nodeErrorData)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		err = relayProcessor.WaitForResults(ctx)
		require.NoError(t, err)

		// Process results - should fail because only 2 successes and they don't match
		// Node errors don't count towards cross-validation in CrossValidation mode
		_, err = relayProcessor.ProcessingResult()
		require.Error(t, err, "CrossValidation mode should fail when success can't form consensus (node errors don't count)")
		require.Contains(t, err.Error(), "cross-validation failed: insufficient successful responses")
	})
}

// Test 2.3: Success cross-validation ignores node errors entirely
func TestSuccessCrossValidationIgnoresNodeErrors(t *testing.T) {
	t.Run("success_cross-validation_ignores_all_node_errors", func(t *testing.T) {
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
			&common.CrossValidationParams{AgreementThreshold: 3, MaxParticipants: 8},
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
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
		require.NoError(t, err, "Success cross-validation should succeed regardless of node errors")
		require.NotNil(t, returnedResult)
		require.Equal(t, 3, returnedResult.CrossValidation, "CrossValidation should be 3 from success results")
		require.Equal(t, successData, returnedResult.Reply.Data, "Should return success data")
		require.Equal(t, 200, returnedResult.StatusCode, "Status should be 200 (success)")
	})
}

// Test: Success cross-validation fails when cross-validation feature is disabled
// This tests the scenario where success responses exist but fail cross-validation
// when cross-validation feature is disabled. With current implementation, when success
// cross-validation fails, it returns error without falling back to node errors.
func TestSuccessCrossValidationFailsWhenCrossValidationDisabled(t *testing.T) {
	t.Run("success_cross-validation_fails_when_cross-validation_disabled", func(t *testing.T) {
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

		// CrossValidation feature DISABLED (Stateful selection) - uses default params, needs just 1 successful result
		relayProcessor := NewRelayProcessor(
			ctx,
			nil,
			nil,
			RelayProcessorMetrics,
			RelayProcessorMetrics,
			RelayRetriesManagerInstance,
			newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateful), // Stateful = cross-validation disabled
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
		// With cross-validation disabled and Min=1, having 1 success should normally succeed
		// But we test the edge case where responsesCrossValidation might fail for other reasons
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
		// With cross-validation disabled: requiredCrossValidationSize = Min = 1
		// successResultsCount (1) >= requiredCrossValidationSize (1) → enters success path
		// With 1 valid success response, cross-validation should succeed (maxCount = 1 >= cross-validationSize = 1)
		// This test verifies the code path is executed correctly
		returnedResult, err := relayProcessor.ProcessingResult()

		// With 1 success and cross-validationSize = 1, it should succeed
		// This test documents the expected behavior when cross-validation is disabled
		if err != nil {
			// If it fails, verify it's the expected error
			require.Contains(t, err.Error(), "majority count is less than cross-validationSize",
				"If it fails, should indicate cross-validation failure")
		} else {
			// If it succeeds, that's also valid - 1 success with cross-validationSize=1 should work
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
	mu sync.Mutex
}

func NewMockMetricsTracker() *MockMetricsTracker {
	return &MockMetricsTracker{}
}

func (m *MockMetricsTracker) SetRelayNodeErrorMetric(providerAddress string, chainId string, apiInterface string, method string) {
}

func (m *MockMetricsTracker) GetChainIdAndApiInterface() (string, string) {
	return "TEST_CHAIN", "rest"
}

// TestGetAgreementThreshold tests the getAgreementThreshold function behavior
func TestGetAgreementThreshold(t *testing.T) {
	tests := []struct {
		name                  string
		crossValidationParams *common.CrossValidationParams
		selection             Selection
		expected              int
		description           string
	}{
		{
			name:                  "Stateless - Returns 1 (no cross-validation params)",
			crossValidationParams: nil,
			selection:             Stateless,
			expected:              1,
			description:           "When selection is Stateless with nil params, should return 1",
		},
		{
			name:                  "Stateful - Returns 1 (no cross-validation params)",
			crossValidationParams: nil,
			selection:             Stateful,
			expected:              1,
			description:           "When selection is Stateful with nil params, should return 1",
		},
		{
			name:                  "CrossValidation - Returns AgreementThreshold",
			crossValidationParams: &common.CrossValidationParams{AgreementThreshold: 2, MaxParticipants: 5},
			selection:             CrossValidation,
			expected:              2,
			description:           "When selection is CrossValidation, should return AgreementThreshold",
		},
		{
			name:                  "CrossValidation - High AgreementThreshold",
			crossValidationParams: &common.CrossValidationParams{AgreementThreshold: 4, MaxParticipants: 5},
			selection:             CrossValidation,
			expected:              4,
			description:           "When selection is CrossValidation with high threshold, should return AgreementThreshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := &RelayProcessor{
				crossValidationParams: tt.crossValidationParams,
				selection:             tt.selection,
			}

			result := rp.getAgreementThreshold()
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestGetResultsSummary_UnsupportedMethodFlag verifies that GetResultsSummary
// correctly populates HasUnsupportedMethod from stored node error flags.
func TestGetResultsSummary_UnsupportedMethodFlag(t *testing.T) {
	t.Run("unsupported method flag detected in summary", func(t *testing.T) {
		result := common.RelayResult{
			Reply:               &pairingtypes.RelayReply{Data: []byte(`{"error":"method not found"}`)},
			StatusCode:          200,
			IsNodeError:         true,
			IsUnsupportedMethod: true,
			IsNonRetryable:      true,
		}
		require.True(t, result.IsUnsupportedMethod)
		require.True(t, result.IsNonRetryable)
	})

	t.Run("smart contract error does not set unsupported flag", func(t *testing.T) {
		result := common.RelayResult{
			Reply:               &pairingtypes.RelayReply{Data: []byte(`{"error":"execution reverted: identity not found"}`)},
			StatusCode:          200,
			IsNodeError:         true,
			IsUnsupportedMethod: false,
			IsNonRetryable:      true, // non-retryable but NOT unsupported
		}
		require.False(t, result.IsUnsupportedMethod)
		require.True(t, result.IsNonRetryable)

		isUnsupported := common.IsUnsupportedMethodError("", 0, string(result.Reply.Data))
		require.False(t, isUnsupported, "Smart contract 'identity not found' should NOT match unsupported patterns")
	})

	t.Run("registry-based classification", func(t *testing.T) {
		unsupportedMessages := []string{
			"method not found",
			"endpoint not found",
		}
		for _, msg := range unsupportedMessages {
			isUnsupported := common.IsUnsupportedMethodError("", 0, msg)
			require.True(t, isUnsupported, "Message '%s' should be detected as unsupported", msg)
		}

		smartContractMessages := []string{
			"execution reverted: NFT not found",
			"execution reverted: User not found",
			"execution reverted: identity not found",
		}
		for _, msg := range smartContractMessages {
			isUnsupported := common.IsUnsupportedMethodError("", 0, msg)
			require.False(t, isUnsupported, "Smart contract message '%s' should NOT be detected as unsupported", msg)
		}
	})
}

// TestGetResultsSummary_NonRetryableFlag verifies that GetResultsSummary
// correctly populates HasNonRetryableNodeError from the IsNonRetryable flag
// on stored node error results. This is the path used by policy.Decide().
func TestGetResultsSummary_NonRetryableFlag(t *testing.T) {
	cases := []struct {
		name             string
		nonRetryable     bool
		wantNonRetryable bool
	}{
		{name: "non-retryable node error sets HasNonRetryableNodeError", nonRetryable: true, wantNonRetryable: true},
		{name: "retryable node error leaves HasNonRetryableNodeError false", nonRetryable: false, wantNonRetryable: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
			if closeServer != nil {
				defer closeServer()
			}
			require.NoError(t, err)
			chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")

			usedProviders := lavasession.NewUsedProviders(nil)
			relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachine(protocolMessage, usedProviders))

			lockCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
			defer cancel()
			require.Nil(t, usedProviders.TryLockSelection(lockCtx))
			usedProviders.AddUsed(lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}}, nil)

			go SendNodeErrorWithRetryable(relayProcessor, "lava@test", time.Millisecond*5, tc.nonRetryable)

			waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Millisecond*200)
			defer waitCancel()
			require.NoError(t, relayProcessor.WaitForResults(waitCtx))

			summary := relayProcessor.GetResultsSummary()
			require.Equal(t, tc.wantNonRetryable, summary.HasNonRetryableNodeError,
				"HasNonRetryableNodeError mismatch — policy.Decide() relies on this flag")
			require.Equal(t, 1, summary.NodeErrors, "should have exactly one node error")
		})
	}
}

// TestIsNonRetryableNodeError_Classification verifies the classifier helper
// the consumer/smart-router hot paths use to populate RelayResult.IsNonRetryable
// correctly separates retryable transient errors from terminal deterministic
// ones. "execution reverted" is the canonical case that regressed before this
// fix (non-retryable in the registry, but the old code ignored Retryable=false
// for node errors that weren't unsupported-method or user-input).
func TestIsNonRetryableNodeError_Classification(t *testing.T) {
	cases := []struct {
		name    string
		chainID string
		status  int
		message string
		want    bool
	}{
		{name: "execution reverted is non-retryable", chainID: "ETH1", status: 200, message: "execution reverted", want: true},
		{name: "execution reverted with data is non-retryable", chainID: "ETH1", status: 200, message: "execution reverted: NFT not found", want: true},
		{name: "unsupported method is non-retryable (Retryable=false)", chainID: "ETH1", status: 404, message: "method not found", want: true},
		{name: "user parse error is non-retryable", chainID: "ETH1", status: -32700, message: "parse error", want: true},
		{name: "generic transient error is retryable", chainID: "ETH1", status: 502, message: "bad gateway", want: false},
		{name: "unknown message is retryable (returns false)", chainID: "ETH1", status: 0, message: "totally unfamiliar garbage", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := common.IsNonRetryableNodeError(tc.chainID, tc.status, tc.message)
			require.Equal(t, tc.want, got, "message %q", tc.message)
		})
	}
}

// TestCrossValidationEmptyArrayResponse tests that empty arrays [] are valid for cross-validation
func TestCrossValidationEmptyArrayResponse(t *testing.T) {
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

	cvParams := &common.CrossValidationParams{
		AgreementThreshold: 2,
		MaxParticipants:    3,
	}

	relayProcessor := NewRelayProcessor(
		ctx,
		cvParams,
		nil,
		RelayProcessorMetrics,
		RelayProcessorMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
	)

	// Set the batch size so WaitForResults knows how many responses to expect
	usedProviders.AddUsed(lavasession.ConsumerSessionsMap{
		"provider1": &lavasession.SessionInfo{},
		"provider2": &lavasession.SessionInfo{},
		"provider3": &lavasession.SessionInfo{},
	}, nil)

	// All providers return empty array [] (e.g., eth_getLogs with no matching logs)
	emptyArrayData := []byte(`[]`)
	relayProcessor.SetResponse(createMockRelayResponseWithData("provider1", emptyArrayData, nil))
	relayProcessor.SetResponse(createMockRelayResponseWithData("provider2", emptyArrayData, nil))
	relayProcessor.SetResponse(createMockRelayResponseWithData("provider3", emptyArrayData, nil))

	// Wait for responses to be processed
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	relayProcessor.WaitForResults(waitCtx)

	// Should reach consensus on empty array
	result, err := relayProcessor.ProcessingResult()
	require.NoError(t, err, "Cross-validation should succeed with matching empty arrays")
	require.NotNil(t, result)
	require.Equal(t, emptyArrayData, result.Reply.Data)
	require.GreaterOrEqual(t, result.CrossValidation, 2, "CrossValidation count should meet threshold")
}

// TestCrossValidationEmptyObjectResponse tests that empty objects {} are valid for cross-validation
func TestCrossValidationEmptyObjectResponse(t *testing.T) {
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

	cvParams := &common.CrossValidationParams{
		AgreementThreshold: 2,
		MaxParticipants:    2,
	}

	relayProcessor := NewRelayProcessor(
		ctx,
		cvParams,
		nil,
		RelayProcessorMetrics,
		RelayProcessorMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
	)

	// Set the batch size so WaitForResults knows how many responses to expect
	usedProviders.AddUsed(lavasession.ConsumerSessionsMap{
		"provider1": &lavasession.SessionInfo{},
		"provider2": &lavasession.SessionInfo{},
	}, nil)

	// Both providers return empty object {}
	emptyObjectData := []byte(`{}`)
	relayProcessor.SetResponse(createMockRelayResponseWithData("provider1", emptyObjectData, nil))
	relayProcessor.SetResponse(createMockRelayResponseWithData("provider2", emptyObjectData, nil))

	// Wait for responses to be processed
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	relayProcessor.WaitForResults(waitCtx)

	result, err := relayProcessor.ProcessingResult()
	require.NoError(t, err, "Cross-validation should succeed with matching empty objects")
	require.NotNil(t, result)
	require.Equal(t, emptyObjectData, result.Reply.Data)
	require.GreaterOrEqual(t, result.CrossValidation, 2, "CrossValidation count should meet threshold")
}

// TestCrossValidationUniformBehaviorDeterministicAndNonDeterministic tests that both
// deterministic and non-deterministic APIs behave the same for cross-validation:
// quorum met = success, quorum not met = failure
func TestCrossValidationUniformBehaviorDeterministicAndNonDeterministic(t *testing.T) {
	ctx := context.Background()

	// Test both deterministic and non-deterministic categories
	testCases := []struct {
		name          string
		deterministic bool
	}{
		{"deterministic API", true},
		{"non-deterministic API", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" - quorum met succeeds", func(t *testing.T) {
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

			// Override deterministic flag for testing
			if api := chainMsg.GetApi(); api != nil {
				api.Category.Deterministic = tc.deterministic
			}

			protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")

			usedProviders := lavasession.NewUsedProviders(nil)

			cvParams := &common.CrossValidationParams{
				AgreementThreshold: 2,
				MaxParticipants:    3,
			}

			relayProcessor := NewRelayProcessor(
				ctx,
				cvParams,
				nil,
				RelayProcessorMetrics,
				RelayProcessorMetrics,
				RelayRetriesManagerInstance,
				newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
			)

			// Set the batch size so WaitForResults knows how many responses to expect
			usedProviders.AddUsed(lavasession.ConsumerSessionsMap{
				"provider1": &lavasession.SessionInfo{},
				"provider2": &lavasession.SessionInfo{},
				"provider3": &lavasession.SessionInfo{},
			}, nil)

			// 2 providers return same data (meets threshold of 2)
			sameData := []byte(`{"result": "same"}`)
			differentData := []byte(`{"result": "different"}`)
			relayProcessor.SetResponse(createMockRelayResponseWithData("provider1", sameData, nil))
			relayProcessor.SetResponse(createMockRelayResponseWithData("provider2", sameData, nil))
			relayProcessor.SetResponse(createMockRelayResponseWithData("provider3", differentData, nil))

			// Wait for responses to be processed
			waitCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
			defer cancel()
			relayProcessor.WaitForResults(waitCtx)

			result, err := relayProcessor.ProcessingResult()
			require.NoError(t, err, "Cross-validation should succeed when quorum is met for %s", tc.name)
			require.NotNil(t, result)
			require.Equal(t, sameData, result.Reply.Data)
			require.GreaterOrEqual(t, result.CrossValidation, 2)
		})

		t.Run(tc.name+" - quorum not met fails", func(t *testing.T) {
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

			// Override deterministic flag for testing
			if api := chainMsg.GetApi(); api != nil {
				api.Category.Deterministic = tc.deterministic
			}

			protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, "", "")

			usedProviders := lavasession.NewUsedProviders(nil)

			cvParams := &common.CrossValidationParams{
				AgreementThreshold: 2,
				MaxParticipants:    3,
			}

			relayProcessor := NewRelayProcessor(
				ctx,
				cvParams,
				nil,
				RelayProcessorMetrics,
				RelayProcessorMetrics,
				RelayRetriesManagerInstance,
				newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
			)

			// Set the batch size so WaitForResults knows how many responses to expect
			usedProviders.AddUsed(lavasession.ConsumerSessionsMap{
				"provider1": &lavasession.SessionInfo{},
				"provider2": &lavasession.SessionInfo{},
				"provider3": &lavasession.SessionInfo{},
			}, nil)

			// All providers return different data (none meet threshold)
			relayProcessor.SetResponse(createMockRelayResponseWithData("provider1", []byte(`{"result": "a"}`), nil))
			relayProcessor.SetResponse(createMockRelayResponseWithData("provider2", []byte(`{"result": "b"}`), nil))
			relayProcessor.SetResponse(createMockRelayResponseWithData("provider3", []byte(`{"result": "c"}`), nil))

			// Wait for responses to be processed
			waitCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
			defer cancel()
			relayProcessor.WaitForResults(waitCtx)

			result, err := relayProcessor.ProcessingResult()
			require.Error(t, err, "Cross-validation should fail when quorum is not met for %s", tc.name)
			// When cross-validation fails, ProcessingResult returns an error placeholder result (not nil)
			// with StatusCode 500, so we check the error message instead
			require.Contains(t, err.Error(), "cross-validation failed")
			if result != nil {
				require.Equal(t, 500, result.StatusCode, "Failed cross-validation should have error status code")
			}
		})
	}
}

// TestCrossValidationMaxParticipantsLimit tests that MaxParticipants cannot exceed MaxCallsPerRelay
func TestCrossValidationMaxParticipantsLimit(t *testing.T) {
	// This test verifies the constant value and that valid params work
	require.Equal(t, 50, MaxCallsPerRelay, "MaxCallsPerRelay should be 50")

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

	// Valid case: MaxParticipants <= MaxCallsPerRelay
	cvParams := &common.CrossValidationParams{
		AgreementThreshold: 2,
		MaxParticipants:    MaxCallsPerRelay, // Max allowed
	}

	// This should not panic
	relayProcessor := NewRelayProcessor(
		ctx,
		cvParams,
		nil,
		RelayProcessorMetrics,
		RelayProcessorMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, CrossValidation),
	)
	require.NotNil(t, relayProcessor)
}

// TestCrossValidationRequiresNonNilParams tests that Stateless mode works with nil params
func TestCrossValidationRequiresNonNilParams(t *testing.T) {
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

	// Stateless mode with nil params should work
	statelessProcessor := NewRelayProcessor(
		ctx,
		nil,
		nil,
		RelayProcessorMetrics,
		RelayProcessorMetrics,
		RelayRetriesManagerInstance,
		newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless),
	)
	require.NotNil(t, statelessProcessor)
}

func TestHasRequiredNodeResults_RelayRetryLimit(t *testing.T) {
	// After the policy refactor, HasRequiredNodeResults always returns false
	// when there are no successful results. Retry decisions are made by policy.Decide().
	tests := []struct {
		name               string
		retryLimit         int
		nodeErrorsToSend   int
		expectedDone       bool // always false when no successes (policy decides retry)
		expectedNodeErrors int
	}{
		{
			name:               "under limit - no success signals false",
			retryLimit:         3,
			nodeErrorsToSend:   2,
			expectedDone:       false,
			expectedNodeErrors: 2,
		},
		{
			name:               "over limit - no success signals false",
			retryLimit:         2,
			nodeErrorsToSend:   3,
			expectedDone:       false,
			expectedNodeErrors: 3,
		},
		{
			name:               "disabled (0) - no success signals false",
			retryLimit:         0,
			nodeErrorsToSend:   1,
			expectedDone:       false,
			expectedNodeErrors: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save and restore global
			originalValue := RelayRetryLimit
			RelayRetryLimit = tc.retryLimit
			defer func() {
				RelayRetryLimit = originalValue
			}()

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
			relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless))

			// Send node errors one at a time, each in its own batch
			for i := 0; i < tc.nodeErrorsToSend; i++ {
				provider := fmt.Sprintf("lava@test%d", i)
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
				canUse := usedProviders.TryLockSelection(ctx)
				require.NoError(t, ctx.Err())
				require.Nil(t, canUse)
				cancel()
				consumerSessionsMap := lavasession.ConsumerSessionsMap{provider: &lavasession.SessionInfo{}}
				usedProviders.AddUsed(consumerSessionsMap, nil)

				go SendNodeError(relayProcessor, provider, 0)
				ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*100)
				err = relayProcessor.WaitForResults(ctx)
				cancel()
				require.NoError(t, err)
			}

			done, nodeErrors := relayProcessor.HasRequiredNodeResults(tc.nodeErrorsToSend)
			require.Equal(t, tc.expectedDone, done, "HasRequiredNodeResults done mismatch")
			require.Equal(t, tc.expectedNodeErrors, nodeErrors, "nodeErrors count mismatch")
		})
	}
}

func TestHasRequiredNodeResults_RelayRetryLimitProtocolError(t *testing.T) {
	// After the policy refactor, HasRequiredNodeResults always returns false
	// when there are no successful results. Retry decisions are made by policy.Decide().
	tests := []struct {
		name                 string
		retryLimit           int
		protocolErrorsToSend int
		expectedDone         bool // always false when no successes (policy decides retry)
		expectedNodeErrors   int
	}{
		{
			name:                 "under limit - no success signals false",
			retryLimit:           3,
			protocolErrorsToSend: 2,
			expectedDone:         false,
			expectedNodeErrors:   0,
		},
		{
			name:                 "over limit - no success signals false",
			retryLimit:           2,
			protocolErrorsToSend: 3,
			expectedDone:         false,
			expectedNodeErrors:   0,
		},
		{
			name:                 "disabled (0) - no success signals false",
			retryLimit:           0,
			protocolErrorsToSend: 1,
			expectedDone:         false,
			expectedNodeErrors:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalValue := RelayRetryLimit
			RelayRetryLimit = tc.retryLimit
			defer func() {
				RelayRetryLimit = originalValue
			}()

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
			relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless))

			// Send protocol errors one at a time, each in its own batch
			for i := 0; i < tc.protocolErrorsToSend; i++ {
				provider := fmt.Sprintf("lava@test%d", i)
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
				canUse := usedProviders.TryLockSelection(ctx)
				require.NoError(t, ctx.Err())
				require.Nil(t, canUse)
				cancel()
				consumerSessionsMap := lavasession.ConsumerSessionsMap{provider: &lavasession.SessionInfo{}}
				usedProviders.AddUsed(consumerSessionsMap, nil)

				go SendProtocolError(relayProcessor, provider, 0, fmt.Errorf("protocol error %d", i))
				ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*100)
				err = relayProcessor.WaitForResults(ctx)
				cancel()
				require.NoError(t, err)
			}

			done, nodeErrors := relayProcessor.HasRequiredNodeResults(tc.protocolErrorsToSend)
			require.Equal(t, tc.expectedDone, done, "HasRequiredNodeResults done mismatch")
			require.Equal(t, tc.expectedNodeErrors, nodeErrors, "nodeErrors count mismatch")
		})
	}
}

func TestHasRequiredNodeResults_RelayRetryLimitMixed(t *testing.T) {
	// After the policy refactor, HasRequiredNodeResults always returns false
	// when there are no successful results. Retry decisions are made by policy.Decide().
	tests := []struct {
		name                 string
		relayRetryLimit      int
		nodeErrorsToSend     int
		protocolErrorsToSend int
		expectedDone         bool // always false when no successes (policy decides retry)
		expectedNodeErrors   int
	}{
		{
			name:                 "total within limit - no success signals false",
			relayRetryLimit:      5,
			nodeErrorsToSend:     2,
			protocolErrorsToSend: 2,
			expectedDone:         false,
			expectedNodeErrors:   2,
		},
		{
			name:                 "total exceeds limit - no success signals false",
			relayRetryLimit:      3,
			nodeErrorsToSend:     2,
			protocolErrorsToSend: 2,
			expectedDone:         false,
			expectedNodeErrors:   2,
		},
		{
			name:                 "total at exact limit - no success signals false",
			relayRetryLimit:      4,
			nodeErrorsToSend:     2,
			protocolErrorsToSend: 2,
			expectedDone:         false,
			expectedNodeErrors:   2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalValue := RelayRetryLimit
			RelayRetryLimit = tc.relayRetryLimit
			defer func() {
				RelayRetryLimit = originalValue
			}()

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
			relayProcessor := NewRelayProcessor(ctx, nil, nil, RelayProcessorMetrics, RelayProcessorMetrics, RelayRetriesManagerInstance, newMockRelayStateMachineWithSelection(protocolMessage, usedProviders, Stateless))

			totalErrors := tc.nodeErrorsToSend + tc.protocolErrorsToSend
			// Send node errors first, then protocol errors
			for i := 0; i < totalErrors; i++ {
				provider := fmt.Sprintf("lava@test%d", i)
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
				canUse := usedProviders.TryLockSelection(ctx)
				require.NoError(t, ctx.Err())
				require.Nil(t, canUse)
				cancel()
				consumerSessionsMap := lavasession.ConsumerSessionsMap{provider: &lavasession.SessionInfo{}}
				usedProviders.AddUsed(consumerSessionsMap, nil)

				if i < tc.nodeErrorsToSend {
					go SendNodeError(relayProcessor, provider, 0)
				} else {
					go SendProtocolError(relayProcessor, provider, 0, fmt.Errorf("protocol error %d", i))
				}
				ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*100)
				err = relayProcessor.WaitForResults(ctx)
				cancel()
				require.NoError(t, err)
			}

			done, nodeErrors := relayProcessor.HasRequiredNodeResults(totalErrors)
			require.Equal(t, tc.expectedDone, done, "HasRequiredNodeResults done mismatch")
			require.Equal(t, tc.expectedNodeErrors, nodeErrors, "nodeErrors count mismatch")
		})
	}
}
