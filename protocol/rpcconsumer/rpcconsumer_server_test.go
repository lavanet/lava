package rpcconsumer

import (
	"context"
	"net/http"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavaprotocol"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/protocol/metrics"
	"github.com/lavanet/lava/v2/protocol/provideroptimizer"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/lavanet/lava/v2/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"
)

func createRpcConsumer(t *testing.T, ctrl *gomock.Controller, ctx context.Context, consumeSK *btcSecp256k1.PrivateKey, consumerAccount types.AccAddress, providerPublicAddress string, relayer pairingtypes.RelayerClient, specId string, apiInterface string, epoch uint64, requiredResponses int, lavaChainID string) (*RPCConsumerServer, chainlib.ChainParser) {
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
	})
	chainParser, _, chainFetcher, _, _, err := chainlib.CreateChainLibMocks(ctx, specId, apiInterface, serverHandler, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainFetcher)

	rpcConsumerServer := &RPCConsumerServer{}
	rpcEndpoint := &lavasession.RPCEndpoint{
		NetworkAddress:  "127.0.0.1:54321",
		ChainID:         specId,
		ApiInterface:    apiInterface,
		TLSEnabled:      false,
		HealthCheckPath: "",
		Geolocation:     1,
	}

	consumerStateTracker := NewMockConsumerStateTrackerInf(ctrl)
	consumerStateTracker.
		EXPECT().
		GetLatestVirtualEpoch().
		Return(uint64(0)).
		AnyTimes()

	finalizationConsensus := lavaprotocol.NewFinalizationConsensus(rpcEndpoint.ChainID)
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	baseLatency := common.AverageWorldLatency / 2
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, averageBlockTime, baseLatency, 2)
	consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil, "test")
	consumerSessionManager.UpdateAllProviders(epoch, map[uint64]*lavasession.ConsumerSessionsWithProvider{
		epoch: {
			PublicLavaAddress: providerPublicAddress,
			PairingEpoch:      epoch,
			Endpoints:         []*lavasession.Endpoint{{Connections: []*lavasession.EndpointConnection{{Client: relayer}}}},
		},
	})

	consumerConsistency := NewConsumerConsistency(specId)
	consumerCmdFlags := common.ConsumerCmdFlags{
		RelaysHealthEnableFlag: false,
	}
	rpcsonumerLogs, err := metrics.NewRPCConsumerLogs(nil, nil)
	require.NoError(t, err)
	err = rpcConsumerServer.ServeRPCRequests(ctx, rpcEndpoint, consumerStateTracker, chainParser, finalizationConsensus, consumerSessionManager, requiredResponses, consumeSK, lavaChainID, nil, rpcsonumerLogs, consumerAccount, consumerConsistency, nil, consumerCmdFlags, false, nil, nil)
	require.NoError(t, err)

	return rpcConsumerServer, chainParser
}

func handleRelay(t *testing.T, request *pairingtypes.RelayRequest, providerSK *btcSecp256k1.PrivateKey, consumerAccount types.AccAddress) *pairingtypes.RelayReply {
	relayReply := &pairingtypes.RelayReply{
		Data:                  []byte(`{"jsonrpc":"2.0","result":{}, "id":1}`),
		FinalizedBlocksHashes: []byte(`{}`),
	}

	relayExchange := &pairingtypes.RelayExchange{
		Request: *request,
		Reply:   *relayReply,
	}

	sig, err := sigs.Sign(providerSK, *relayExchange)

	require.NoError(t, err)
	relayReply.Sig = sig

	sigBlocks, err := sigs.Sign(providerSK, pairingtypes.RelayFinalization{
		Exchange: *relayExchange,
		Addr:     consumerAccount,
	})

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

	// Create the RPC consumer server
	rpcConsumerServer, chainParser := createRpcConsumer(t, ctrl, context.Background(), consumeSK, consumerAccount, providerPublicAddress, relayerMock, "LAV1", spectypes.APIInterfaceTendermintRPC, 100, 1, "lava")
	require.NotNil(t, rpcConsumerServer)
	require.NotNil(t, chainParser)

	// Create a chain message
	chainMsg, err := chainParser.ParseMsg("", []byte(`{"jsonrpc":"2.0","method":"status","params":[],"id":1}`), "", nil, extensionslib.ExtensionInfo{})
	require.NoError(t, err)

	// Create single consumer session
	singleConsumerSession := &lavasession.SingleConsumerSession{EndpointConnection: &lavasession.EndpointConnection{Client: relayerMock}}

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
		_, err, _ := rpcConsumerServer.relayInner(context.Background(), singleConsumerSession, relayResult, 1, chainMsg, "", nil)
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
