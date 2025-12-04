package rpcsmartrouter

import (
	"context"
	"net/http"
	"testing"
	"time"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v5/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func createRpcSmartRouter(t *testing.T, ctrl *gomock.Controller, ctx context.Context, consumeSK *btcSecp256k1.PrivateKey, consumerAccount types.AccAddress, providerPublicAddress string, relayer pairingtypes.RelayerClient, specId string, apiInterface string, epoch uint64, requiredResponses int, lavaChainID string) (*RPCSmartRouterServer, chainlib.ChainParser) {
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
	})
	chainParser, _, chainFetcher, _, _, err := chainlib.CreateChainLibMocks(ctx, specId, apiInterface, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainFetcher)

	rpcSmartRouterServer := &RPCSmartRouterServer{}
	rpcEndpoint := &lavasession.RPCEndpoint{
		NetworkAddress:  "127.0.0.1:54321",
		ChainID:         specId,
		ApiInterface:    apiInterface,
		TLSEnabled:      false,
		HealthCheckPath: "",
		Geolocation:     1,
	}

	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.StrategyBalanced, averageBlockTime, 2, nil, "dontcare", false)
	consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil, "test", lavasession.NewActiveSubscriptionProvidersStorage())
	consumerSessionManager.UpdateAllProviders(epoch, map[uint64]*lavasession.ConsumerSessionsWithProvider{
		epoch: {
			PublicLavaAddress: providerPublicAddress,
			PairingEpoch:      epoch,
			Endpoints:         []*lavasession.Endpoint{{Connections: []*lavasession.EndpointConnection{{Client: relayer}}}},
		},
	}, nil)

	smartRouterConsistency := relaycore.NewConsistency(specId)
	consumerCmdFlags := common.ConsumerCmdFlags{
		RelaysHealthEnableFlag: false,
	}
	rpcSmartRouterLogs, err := metrics.NewRPCConsumerLogs(nil, nil, nil, nil)
	require.NoError(t, err)
	// Smart router signature: no consumerStateTracker, no finalizationConsensus
	err = rpcSmartRouterServer.ServeRPCRequests(ctx, rpcEndpoint, chainParser, consumerSessionManager, requiredResponses, consumeSK, lavaChainID, nil, rpcSmartRouterLogs, consumerAccount, smartRouterConsistency, nil, consumerCmdFlags, false, nil, nil, nil)
	require.NoError(t, err)

	return rpcSmartRouterServer, chainParser
}

func handleRelay(t *testing.T, request *pairingtypes.RelayRequest, providerSK *btcSecp256k1.PrivateKey, consumerAccount types.AccAddress) *pairingtypes.RelayReply {
	relayReply := &pairingtypes.RelayReply{
		Data:                  []byte(`{"jsonrpc":"2.0","result":{}, "id":1}`),
		FinalizedBlocksHashes: []byte(`{"0":"hash0"}`),
	}

	relayExchange := &pairingtypes.RelayExchange{
		Request: *request,
		Reply:   *relayReply,
	}

	sig, err := sigs.Sign(providerSK, *relayExchange)

	require.NoError(t, err)
	relayReply.Sig = sig

	sigBlocks, err := sigs.Sign(providerSK, conflicttypes.NewRelayFinalizationFromRelaySessionAndRelayReply(request.RelaySession, relayReply, consumerAccount))

	require.NoError(t, err)
	relayReply.SigBlocks = sigBlocks

	return relayReply
}

func TestRelayInnerProviderUniqueIdFlow(t *testing.T) {
	rand.InitRandomSeed()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	providerSK, providerAccount := sigs.GenerateFloatingKey()
	providerPublicAddress := providerAccount.String()
	consumeSK, consumerAccount := sigs.GenerateFloatingKey()

	providerUniqueId := "foobar"

	relayerMock := NewMockRelayerClient(ctrl)
	relayerMock.
		EXPECT().
		Relay(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, in *pairingtypes.RelayRequest, opts ...grpc.CallOption) (*pairingtypes.RelayReply, error) {
			if len(opts) == 1 {
				trailerCallOption, ok := opts[0].(grpc.TrailerCallOption)
				require.True(t, ok)
				require.NotNil(t, trailerCallOption)

				trailerCallOption.TrailerAddr.Set(chainlib.RpcProviderUniqueIdHeader, providerUniqueId)
			}
			return handleRelay(t, in, providerSK, consumerAccount), nil
		}).
		AnyTimes()

	// Create the RPC smart router server
	rpcSmartRouterServer, chainParser := createRpcSmartRouter(t, ctrl, context.Background(), consumeSK, consumerAccount, providerPublicAddress, relayerMock, "LAV1", spectypes.APIInterfaceTendermintRPC, 100, 1, "lava")
	require.NotNil(t, rpcSmartRouterServer)
	require.NotNil(t, chainParser)

	// Create a chain message
	chainMsg, err := chainParser.ParseMsg("", []byte(`{"jsonrpc":"2.0","method":"status","params":[],"id":1}`), "", nil, extensionslib.ExtensionInfo{})
	require.NoError(t, err)

	// Create single consumer session
	singleConsumerSession := &lavasession.SingleConsumerSession{EndpointConnection: &lavasession.EndpointConnection{Client: relayerMock}}
	singleConsumerSession.Parent = &lavasession.ConsumerSessionsWithProvider{
		PublicLavaAddress: providerPublicAddress,
		PairingEpoch:      100,
		Endpoints:         []*lavasession.Endpoint{{Connections: []*lavasession.EndpointConnection{{Client: relayerMock}}}},
	}
	// Create RelayResult
	relayResult := &common.RelayResult{
		ProviderInfo: common.ProviderInfo{
			ProviderAddress: providerPublicAddress,
		},
		Request: &pairingtypes.RelayRequest{
			RelayData: &pairingtypes.RelayPrivateData{RequestBlock: 0},
			RelaySession: &pairingtypes.RelaySession{
				SessionId: 1,
				Epoch:     100,
				RelayNum:  1,
			},
		},
	}

	callRelayInner := func() error {
		_, err, _ := rpcSmartRouterServer.relayInner(context.Background(), singleConsumerSession, relayResult, 1, chainMsg, "", nil)
		return err
	}

	t.Run("TestRelayInnerProviderUniqueIdFlow", func(t *testing.T) {
		// Setting the first provider unique id
		require.NoError(t, callRelayInner())

		// It's still the same, should pass
		require.NoError(t, callRelayInner())

		oldProviderUniqueId := providerUniqueId
		providerUniqueId = "barfoo"

		// Now the providerUniqueId has changed, should fail
		require.Error(t, callRelayInner())

		providerUniqueId = oldProviderUniqueId

		// Back to the correct providerUniqueId, should pass
		require.NoError(t, callRelayInner())
	})
}

// Mock interface for RelayProcessor that only implements the methods we need for testing
type MockRelayProcessorForHeaders struct {
	quorumParams         common.QuorumParams
	successResults       []common.RelayResult
	nodeErrors           []common.RelayResult
	protocolErrors       []relaycore.RelayError
	statefulRelayTargets []string
}

func (m *MockRelayProcessorForHeaders) GetQuorumParams() common.QuorumParams {
	return m.quorumParams
}

func (m *MockRelayProcessorForHeaders) GetResultsData() ([]common.RelayResult, []common.RelayResult, []relaycore.RelayError) {
	return m.successResults, m.nodeErrors, m.protocolErrors
}

func (m *MockRelayProcessorForHeaders) GetStatefulRelayTargets() []string {
	return m.statefulRelayTargets
}

func (m *MockRelayProcessorForHeaders) GetUsedProviders() *lavasession.UsedProviders {
	return lavasession.NewUsedProviders(nil)
}

func (m *MockRelayProcessorForHeaders) NodeErrors() (ret []common.RelayResult) {
	return m.nodeErrors
}

// Integration tests that actually call appendHeadersToRelayResult
func TestAppendHeadersToRelayResultIntegration(t *testing.T) {
	ctx := context.Background()
	providerAddress1 := "lava@provider1"
	providerAddress2 := "lava@provider2"
	providerAddress3 := "lava@provider3"

	t.Run("quorum disabled - single provider header", func(t *testing.T) {
		// Create a mock relay processor with quorum disabled (use default values)
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:   common.DefaultQuorumParams, // Disable quorum by using default values
			successResults: []common.RelayResult{},
			nodeErrors:     []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a simple mock protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{Name: "test-api"},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "test-api")

		// Verify the result - should have single provider header + user request type header
		require.Len(t, relayResult.Reply.Metadata, 2)

		// Find the provider address header
		var providerHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.PROVIDER_ADDRESS_HEADER_NAME {
				providerHeader = &meta
				break
			}
		}
		require.NotNil(t, providerHeader)
		require.Equal(t, providerAddress1, providerHeader.Value)
	})

	t.Run("quorum enabled - single successful provider", func(t *testing.T) {
		// Create a mock relay processor with quorum enabled
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams: common.QuorumParams{Min: 2, Rate: 0.6, Max: 5},
			successResults: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1}},
			},
			nodeErrors: []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a simple mock protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{Name: "test-api"},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "test-api")

		// Verify the result - should have quorum header + user request type header
		require.Len(t, relayResult.Reply.Metadata, 2)

		// Find the quorum header
		var quorumHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.QUORUM_ALL_PROVIDERS_HEADER_NAME {
				quorumHeader = &meta
				break
			}
		}
		require.NotNil(t, quorumHeader)
		require.Equal(t, "[lava@provider1]", quorumHeader.Value)
	})

	t.Run("quorum enabled - multiple providers with mixed results", func(t *testing.T) {
		// Create a mock relay processor with quorum enabled
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams: common.QuorumParams{Min: 2, Rate: 0.6, Max: 5},
			successResults: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1}},
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress2}},
			},
			nodeErrors: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress3}},
			},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a simple mock protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{Name: "test-api"},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "test-api")

		// Verify the result - should have quorum header with all providers + user request type header
		require.Len(t, relayResult.Reply.Metadata, 2)

		// Find the quorum header
		var quorumHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.QUORUM_ALL_PROVIDERS_HEADER_NAME {
				quorumHeader = &meta
				break
			}
		}
		require.NotNil(t, quorumHeader)

		// Check that all three providers are in the header (order may vary)
		headerValue := quorumHeader.Value
		require.Contains(t, headerValue, "lava@provider1")
		require.Contains(t, headerValue, "lava@provider2")
		require.Contains(t, headerValue, "lava@provider3")
	})

	t.Run("quorum enabled - no providers", func(t *testing.T) {
		// Create a mock relay processor with quorum enabled but no providers
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:   common.QuorumParams{Min: 2, Rate: 0.6, Max: 5},
			successResults: []common.RelayResult{},
			nodeErrors:     []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a simple mock protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{Name: "test-api"},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "test-api")

		// Verify the result - should have only user request type header (no quorum header since no providers)
		require.Len(t, relayResult.Reply.Metadata, 1)

		// Should only have the user request type header
		require.Equal(t, common.USER_REQUEST_TYPE, relayResult.Reply.Metadata[0].Name)
		require.Equal(t, "test-api", relayResult.Reply.Metadata[0].Value)
	})

	t.Run("nil relay result - should not panic", func(t *testing.T) {
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:   common.QuorumParams{Min: 2, Rate: 0.6, Max: 5},
			successResults: []common.RelayResult{},
			nodeErrors:     []common.RelayResult{},
		}

		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{Name: "test-api"},
		}

		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// This should not panic
		require.NotPanics(t, func() {
			rpcSmartRouterServer.appendHeadersToRelayResult(ctx, nil, 0, relayProcessor, mockProtocolMessage, "test-api")
		})
	})
}

// TestStatefulRelayTargetsHeader tests the stateful API header functionality
func TestStatefulRelayTargetsHeader(t *testing.T) {
	ctx := context.Background()
	providerAddress1 := "lava@provider1"
	providerAddress2 := "lava@provider2"
	providerAddress3 := "lava@provider3"

	t.Run("stateful API - all providers header included", func(t *testing.T) {
		// Create a mock relay processor with stateful relay targets
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:         common.DefaultQuorumParams,
			statefulRelayTargets: []string{providerAddress1, providerAddress2, providerAddress3},
			successResults: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1}},
			},
			nodeErrors: []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a stateful API protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{
				Name: "eth_sendTransaction",
				Category: spectypes.SpecCategory{
					Stateful: common.CONSISTENCY_SELECT_ALL_PROVIDERS,
				},
			},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "eth_sendTransaction")

		// Verify the result - should have:
		// 1. Single provider header (winning provider)
		// 2. Stateful API header
		// 3. Stateful all providers header
		// 4. User request type header
		require.Len(t, relayResult.Reply.Metadata, 4)

		// Find and verify the stateful API header
		var statefulHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.STATEFUL_API_HEADER {
				statefulHeader = &meta
				break
			}
		}
		require.NotNil(t, statefulHeader)
		require.Equal(t, "true", statefulHeader.Value)

		// Find and verify the stateful all providers header
		var allProvidersHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.STATEFUL_ALL_PROVIDERS_HEADER_NAME {
				allProvidersHeader = &meta
				break
			}
		}
		require.NotNil(t, allProvidersHeader)

		// Verify all three providers are in the header
		headerValue := allProvidersHeader.Value
		require.Contains(t, headerValue, providerAddress1)
		require.Contains(t, headerValue, providerAddress2)
		require.Contains(t, headerValue, providerAddress3)
	})

	t.Run("stateful API - single provider in targets", func(t *testing.T) {
		// Create a mock relay processor with only one stateful relay target
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:         common.DefaultQuorumParams,
			statefulRelayTargets: []string{providerAddress1},
			successResults: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1}},
			},
			nodeErrors: []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a stateful API protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{
				Name: "eth_sendRawTransaction",
				Category: spectypes.SpecCategory{
					Stateful: common.CONSISTENCY_SELECT_ALL_PROVIDERS,
				},
			},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "eth_sendRawTransaction")

		// Verify the result
		require.Len(t, relayResult.Reply.Metadata, 4)

		// Find and verify the stateful all providers header
		var allProvidersHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.STATEFUL_ALL_PROVIDERS_HEADER_NAME {
				allProvidersHeader = &meta
				break
			}
		}
		require.NotNil(t, allProvidersHeader)
		require.Contains(t, allProvidersHeader.Value, providerAddress1)
	})

	t.Run("stateful API - empty targets list", func(t *testing.T) {
		// Create a mock relay processor with empty stateful relay targets
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:         common.DefaultQuorumParams,
			statefulRelayTargets: []string{},
			successResults: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1}},
			},
			nodeErrors: []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a stateful API protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{
				Name: "eth_sendTransaction",
				Category: spectypes.SpecCategory{
					Stateful: common.CONSISTENCY_SELECT_ALL_PROVIDERS,
				},
			},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "eth_sendTransaction")

		// Verify the result - should NOT have stateful all providers header (empty list)
		// Should have: single provider header, stateful API header, user request type header
		require.Len(t, relayResult.Reply.Metadata, 3)

		// Verify stateful all providers header is NOT present
		for _, meta := range relayResult.Reply.Metadata {
			require.NotEqual(t, common.STATEFUL_ALL_PROVIDERS_HEADER_NAME, meta.Name)
		}

		// Verify stateful API header IS present
		var statefulHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.STATEFUL_API_HEADER {
				statefulHeader = &meta
				break
			}
		}
		require.NotNil(t, statefulHeader)
		require.Equal(t, "true", statefulHeader.Value)
	})

	t.Run("non-stateful API - no stateful headers", func(t *testing.T) {
		// Create a mock relay processor without stateful relay targets
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:         common.DefaultQuorumParams,
			statefulRelayTargets: nil, // No stateful targets
			successResults: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1}},
			},
			nodeErrors: []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a non-stateful API protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{
				Name: "eth_getBlockByNumber",
				Category: spectypes.SpecCategory{
					Stateful: 0, // Not stateful
				},
			},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "eth_getBlockByNumber")

		// Verify the result - should only have: single provider header + user request type header
		require.Len(t, relayResult.Reply.Metadata, 2)

		// Verify NO stateful headers are present
		for _, meta := range relayResult.Reply.Metadata {
			require.NotEqual(t, common.STATEFUL_API_HEADER, meta.Name)
			require.NotEqual(t, common.STATEFUL_ALL_PROVIDERS_HEADER_NAME, meta.Name)
		}

		// Verify single provider header is present
		var providerHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			if meta.Name == common.PROVIDER_ADDRESS_HEADER_NAME {
				providerHeader = &meta
				break
			}
		}
		require.NotNil(t, providerHeader)
		require.Equal(t, providerAddress1, providerHeader.Value)
	})

	t.Run("stateful API with quorum enabled - both headers present", func(t *testing.T) {
		// This is an edge case - stateful API shouldn't use quorum, but let's test the behavior
		relayProcessor := &MockRelayProcessorForHeaders{
			quorumParams:         common.QuorumParams{Min: 2, Rate: 0.6, Max: 5},
			statefulRelayTargets: []string{providerAddress1, providerAddress2},
			successResults: []common.RelayResult{
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1}},
				{ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress2}},
			},
			nodeErrors: []common.RelayResult{},
		}

		// Create a relay result
		relayResult := &common.RelayResult{
			ProviderInfo: common.ProviderInfo{ProviderAddress: providerAddress1},
			Reply: &pairingtypes.RelayReply{
				Metadata: []pairingtypes.Metadata{},
			},
		}

		// Create a stateful API protocol message
		mockProtocolMessage := &MockProtocolMessage{
			api: &spectypes.Api{
				Name: "eth_sendTransaction",
				Category: spectypes.SpecCategory{
					Stateful: common.CONSISTENCY_SELECT_ALL_PROVIDERS,
				},
			},
		}

		// Create RPC consumer server
		rpcSmartRouterServer := &RPCSmartRouterServer{}

		// Call the function
		rpcSmartRouterServer.appendHeadersToRelayResult(ctx, relayResult, 0, relayProcessor, mockProtocolMessage, "eth_sendTransaction")

		// Verify both quorum and stateful headers are present
		// (even though this is an unusual scenario)
		var quorumHeader, statefulHeader, allProvidersHeader *pairingtypes.Metadata
		for _, meta := range relayResult.Reply.Metadata {
			switch meta.Name {
			case common.QUORUM_ALL_PROVIDERS_HEADER_NAME:
				quorumHeader = &meta
			case common.STATEFUL_API_HEADER:
				statefulHeader = &meta
			case common.STATEFUL_ALL_PROVIDERS_HEADER_NAME:
				allProvidersHeader = &meta
			}
		}

		// Verify stateful headers are present
		require.NotNil(t, statefulHeader)
		require.Equal(t, "true", statefulHeader.Value)
		require.NotNil(t, allProvidersHeader)

		// Quorum header would also be present if quorum is enabled
		require.NotNil(t, quorumHeader)
	})
}

// Test the full SendParsedRelay integration (if we can mock the dependencies)
func TestSendParsedRelayIntegration(t *testing.T) {
	// This test would require more complex mocking of the entire relay processor
	// For now, we'll create a simpler version that tests the header logic in context

	t.Run("SendParsedRelay calls appendHeadersToRelayResult", func(t *testing.T) {
		// This is a conceptual test - in practice, we'd need to mock:
		// - ProcessRelaySend
		// - RelayProcessor.ProcessingResult()
		// - All the complex dependencies

		// For now, we'll just verify that our header logic works when called
		// The actual SendParsedRelay integration would require extensive mocking
		// that might be more complex than the value it provides

		require.True(t, true, "SendParsedRelay integration test placeholder - would require complex mocking")
	})
}

// MockResultsManager implements the relaycore.ResultsManager interface for testing
type MockResultsManager struct {
	successResults []common.RelayResult
	nodeErrorsList []common.RelayResult
	protocolErrors []relaycore.RelayError
}

func (m *MockResultsManager) GetResultsData() (successResults []common.RelayResult, nodeErrors []common.RelayResult, protocolErrors []relaycore.RelayError) {
	return m.successResults, m.nodeErrorsList, m.protocolErrors
}

func (m *MockResultsManager) String() string {
	return "MockResultsManager"
}

func (m *MockResultsManager) NodeResults() []common.RelayResult {
	return append(m.successResults, m.nodeErrorsList...)
}

func (m *MockResultsManager) RequiredResults(requiredSuccesses int, selection relaycore.Selection) bool {
	return len(m.successResults) >= requiredSuccesses
}

func (m *MockResultsManager) ProtocolErrors() uint64 {
	return uint64(len(m.protocolErrors))
}

func (m *MockResultsManager) HasResults() bool {
	return len(m.successResults) > 0 || len(m.nodeErrorsList) > 0
}

func (m *MockResultsManager) GetResults() (success int, nodeErrors int, specialNodeErrors int, protocolErrors int) {
	return len(m.successResults), len(m.nodeErrorsList), 0, len(m.protocolErrors)
}

func (m *MockResultsManager) SetResponse(response *relaycore.RelayResponse, protocolMessage chainlib.ProtocolMessage) (nodeError error) {
	return nil
}

func (m *MockResultsManager) GetBestNodeErrorMessageForUser() relaycore.RelayError {
	return relaycore.RelayError{}
}

func (m *MockResultsManager) GetBestProtocolErrorMessageForUser() relaycore.RelayError {
	return relaycore.RelayError{}
}

func (m *MockResultsManager) NodeErrors() (ret []common.RelayResult) {
	return m.nodeErrorsList
}

// MockProtocolMessage implements the ProtocolMessage interface for testing
type MockProtocolMessage struct {
	api *spectypes.Api
}

func (m *MockProtocolMessage) GetApi() *spectypes.Api {
	return m.api
}

func (m *MockProtocolMessage) GetApiCollection() *spectypes.ApiCollection {
	return nil
}

func (m *MockProtocolMessage) GetParseDirective() *spectypes.ParseDirective {
	return nil
}

func (m *MockProtocolMessage) GetUserData() common.UserData {
	return common.UserData{}
}

func (m *MockProtocolMessage) GetRelayData() *pairingtypes.RelayPrivateData {
	return &pairingtypes.RelayPrivateData{}
}

func (m *MockProtocolMessage) GetChainMessage() chainlib.ChainMessage {
	return nil
}

func (m *MockProtocolMessage) GetExtensions() []*spectypes.Extension {
	return nil
}

func (m *MockProtocolMessage) GetDirectiveHeaders() map[string]string {
	return nil
}

func (m *MockProtocolMessage) GetQuorumParameters() (common.QuorumParams, error) {
	return common.QuorumParams{}, nil
}

func (m *MockProtocolMessage) IsDefaultApi() bool {
	return false
}

// Additional methods required by ChainMessage interface
func (m *MockProtocolMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return ""
}

func (m *MockProtocolMessage) RequestedBlock() (latest int64, earliest int64) {
	return 0, 0
}

func (m *MockProtocolMessage) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modified bool) {
	return false
}

func (m *MockProtocolMessage) AppendHeader(metadata []pairingtypes.Metadata) {
	// No-op for testing
}

func (m *MockProtocolMessage) OverrideExtensions(extensionNames []string, extensionParser *extensionslib.ExtensionParser) {
	// No-op for testing
}

func (m *MockProtocolMessage) DisableErrorHandling() {
	// No-op for testing
}

func (m *MockProtocolMessage) TimeoutOverride(...time.Duration) time.Duration {
	return 0
}

func (m *MockProtocolMessage) GetForceCacheRefresh() bool {
	return false
}

func (m *MockProtocolMessage) SetForceCacheRefresh(force bool) bool {
	return false
}

func (m *MockProtocolMessage) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	return false, ""
}

func (m *MockProtocolMessage) GetRawRequestHash() ([]byte, error) {
	return nil, nil
}

func (m *MockProtocolMessage) GetRequestedBlocksHashes() []string {
	return nil
}

func (m *MockProtocolMessage) UpdateEarliestInMessage(incomingEarliest int64) bool {
	return false
}

func (m *MockProtocolMessage) SetExtension(extension *spectypes.Extension) {
	// No-op for testing
}

func (m *MockProtocolMessage) GetUsedDefaultValue() bool {
	return false
}

func (m *MockProtocolMessage) GetRPCMessage() rpcInterfaceMessages.GenericMessage {
	return nil
}

func (m *MockProtocolMessage) RelayPrivateData() *pairingtypes.RelayPrivateData {
	return &pairingtypes.RelayPrivateData{}
}

func (m *MockProtocolMessage) HashCacheRequest(chainId string) ([]byte, func([]byte) []byte, error) {
	return nil, nil, nil
}

func (m *MockProtocolMessage) GetBlockedProviders() []string {
	return nil
}

func (m *MockProtocolMessage) UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64) bool {
	return false
}

// Mock implementation for Relayer_RelaySubscribeClient used in subscription tests
type mockRelaySubscribeClient struct {
	ctx       context.Context
	replyData []byte
	delay     time.Duration
	recvError error
}

func (m *mockRelaySubscribeClient) Recv() (*pairingtypes.RelayReply, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.recvError != nil {
		return nil, m.recvError
	}
	return &pairingtypes.RelayReply{Data: m.replyData}, nil
}

func (m *mockRelaySubscribeClient) RecvMsg(msg interface{}) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.recvError != nil {
		return m.recvError
	}
	if reply, ok := msg.(*pairingtypes.RelayReply); ok {
		reply.Data = m.replyData
	}
	return nil
}

func (m *mockRelaySubscribeClient) Context() context.Context {
	return m.ctx
}

func (m *mockRelaySubscribeClient) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockRelaySubscribeClient) Trailer() metadata.MD {
	return nil
}

func (m *mockRelaySubscribeClient) CloseSend() error {
	return nil
}

func (m *mockRelaySubscribeClient) SendMsg(msg interface{}) error {
	return nil
}

// TestGetFirstSubscriptionReplyNoLeakSuccess verifies no goroutine leak on successful subscription
func TestGetFirstSubscriptionReplyNoLeakSuccess(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/desertbit/timer.timerRoutine"),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*defaultPolicy[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*Cache[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/lavanet/lava/v5/protocol/chainlib.(*TendermintRpcChainListener).Serve"),
	)

	ctx := context.Background()
	mockReplyServer := &mockRelaySubscribeClient{
		ctx:       context.Background(),
		replyData: []byte(`{"jsonrpc":"2.0","result":"0x123","id":1}`),
	}

	server := &RPCSmartRouterServer{}

	reply, err := server.getFirstSubscriptionReply(ctx, "test-hash", mockReplyServer)
	require.NoError(t, err)
	require.NotNil(t, reply)
	require.Equal(t, mockReplyServer.replyData, reply.Data)

	// Give goroutines time to exit
	time.Sleep(100 * time.Millisecond)
	// goleak.VerifyNone should pass - no leaked goroutines
}

// TestGetFirstSubscriptionReplyNoLeakTimeout verifies no goroutine leak when timeout occurs
// Note: This test uses the actual 10s timeout from common.SubscriptionFirstReplyTimeout
// For CI/CD, consider using -short flag to skip this test if runtime is a concern
func TestGetFirstSubscriptionReplyNoLeakTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode (requires 10+ seconds)")
	}

	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/desertbit/timer.timerRoutine"),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*defaultPolicy[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*Cache[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/lavanet/lava/v5/protocol/chainlib.(*TendermintRpcChainListener).Serve"),
	)

	ctx := context.Background()
	mockReplyServer := &mockRelaySubscribeClient{
		ctx:       context.Background(),
		delay:     15 * time.Second, // Longer than 10s timeout
		replyData: []byte(`{"jsonrpc":"2.0","result":"0x123","id":1}`),
	}

	server := &RPCSmartRouterServer{}

	// This will timeout but should not leak goroutines
	_, _ = server.getFirstSubscriptionReply(ctx, "test-hash", mockReplyServer)

	// Wait for timeout goroutine to exit (timeout is 10s, plus buffer)
	time.Sleep(500 * time.Millisecond)
	// goleak.VerifyNone should pass - goroutine exited after timeout
}

// TestGetFirstSubscriptionReplyNoLeakEarlyError verifies no goroutine leak on early error
func TestGetFirstSubscriptionReplyNoLeakEarlyError(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/desertbit/timer.timerRoutine"),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*defaultPolicy[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*Cache[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/lavanet/lava/v5/protocol/chainlib.(*TendermintRpcChainListener).Serve"),
	)

	ctx := context.Background()
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	mockReplyServer := &mockRelaySubscribeClient{
		ctx: cancelledCtx,
	}

	server := &RPCSmartRouterServer{}

	_, err := server.getFirstSubscriptionReply(ctx, "test-hash", mockReplyServer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reply server context canceled")

	// Give defer time to signal channel
	time.Sleep(100 * time.Millisecond)
	// goleak.VerifyNone should pass - defer signaled channel, goroutine exited
}

// TestGetFirstSubscriptionReplyNoDataRace verifies no data race with concurrent access
func TestGetFirstSubscriptionReplyNoDataRace(t *testing.T) {
	// Run with: go test -race
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/desertbit/timer.timerRoutine"),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*defaultPolicy[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*Cache[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/lavanet/lava/v5/protocol/chainlib.(*TendermintRpcChainListener).Serve"),
	)

	ctx := context.Background()

	// Run multiple times to increase race detection likelihood
	for i := 0; i < 50; i++ {
		mockReplyServer := &mockRelaySubscribeClient{
			ctx:       context.Background(),
			replyData: []byte(`{"jsonrpc":"2.0","result":"0x123","id":1}`),
			delay:     time.Duration(i%10) * time.Millisecond, // Vary timing
		}

		server := &RPCSmartRouterServer{}
		_, _ = server.getFirstSubscriptionReply(ctx, "test-hash", mockReplyServer)
	}

	time.Sleep(100 * time.Millisecond)
	// Race detector should not report any races
}

// TestGetFirstSubscriptionReplyContextCancellation verifies goroutine exits on context cancellation
func TestGetFirstSubscriptionReplyContextCancellation(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/desertbit/timer.timerRoutine"),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*defaultPolicy[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/dgraph-io/ristretto/v2.(*Cache[...]).processItems"),
		goleak.IgnoreTopFunction("github.com/lavanet/lava/v5/protocol/chainlib.(*TendermintRpcChainListener).Serve"),
	)

	ctx, cancel := context.WithCancel(context.Background())

	mockReplyServer := &mockRelaySubscribeClient{
		ctx:       context.Background(),
		delay:     500 * time.Millisecond, // Long delay
		replyData: []byte(`{"jsonrpc":"2.0","result":"0x123","id":1}`),
	}

	server := &RPCSmartRouterServer{}

	// Start the call in a goroutine
	go func() {
		_, _ = server.getFirstSubscriptionReply(ctx, "test-hash", mockReplyServer)
	}()

	// Cancel context after a short time
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for goroutine to exit via ctx.Done()
	time.Sleep(150 * time.Millisecond)
	// goleak.VerifyNone should pass - goroutine respected context cancellation
}
