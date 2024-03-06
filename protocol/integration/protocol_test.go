package integration_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/provideroptimizer"
	"github.com/lavanet/lava/protocol/rpcconsumer"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/stretchr/testify/require"

	// pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

var seed int64

func TestMain(m *testing.M) {
	// This code will run once before any test cases are executed.
	seed = time.Now().Unix()

	rand.SetSpecificSeed(seed)
	// Run the actual tests
	exitCode := m.Run()
	if exitCode != 0 {
		utils.LavaFormatDebug("failed tests seed", utils.Attribute{Key: "seed", Value: seed})
	}
	os.Exit(exitCode)
}

func isServerUp(url string) bool {
	client := http.Client{
		Timeout: 20 * time.Millisecond,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.ContentLength > 0
}

func checkServerStatusWithTimeout(url string, totalTimeout time.Duration) bool {
	startTime := time.Now()

	for time.Since(startTime) < totalTimeout {
		if isServerUp(url) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}

	return false
}
func TestConsumerProviderBasic(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
	})
	// can be any spec and api interface
	specId := "LAV1"
	apiInterface := spectypes.APIInterfaceTendermintRPC
	epoch := uint64(100)
	requiredResponses := 1
	lavaChainID := "lava"
	providerListenAddress := "localhost:0"
	consumerListenAddress := "localhost:11111"
	pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{
		1: {
			PublicLavaAddress: "",
			Endpoints: []*lavasession.Endpoint{
				{
					NetworkAddress: providerListenAddress,
					Enabled:        true,
					Geolocation:    0,
				},
			},
			Sessions:         map[int64]*lavasession.SingleConsumerSession{},
			MaxComputeUnits:  10000,
			UsedComputeUnits: 0,
			PairingEpoch:     epoch,
		},
	}

	chainParser, _, chainFetcher, _, err := chainlib.CreateChainLibMocks(ctx, specId, apiInterface, serverHandler, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainFetcher)
	// create basic consumer stuff
	specIdLava := "LAV1"
	chainParserLava, _, chainFetcherLava, _, err := chainlib.CreateChainLibMocks(ctx, specIdLava, spectypes.APIInterfaceRest, serverHandler, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParserLava)
	require.NotNil(t, chainFetcherLava)
	rpcConsumerServer := &rpcconsumer.RPCConsumerServer{}
	rpcEndpoint := &lavasession.RPCEndpoint{
		NetworkAddress:  consumerListenAddress,
		ChainID:         specId,
		ApiInterface:    apiInterface,
		TLSEnabled:      false,
		HealthCheckPath: "",
		Geolocation:     1,
	}
	consumerStateTracker := &mockConsumerStateTracker{}
	finalizationConsensus := lavaprotocol.NewFinalizationConsensus(rpcEndpoint.ChainID)
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	baseLatency := common.AverageWorldLatency / 2 // we want performance to be half our timeout or better
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, averageBlockTime, baseLatency, 2)
	consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil)

	consumerSessionManager.UpdateAllProviders(epoch, pairingList)
	randomizer := sigs.NewZeroReader(seed)
	account := sigs.GenerateDeterministicFloatingKey(randomizer)
	consumerConsistency := rpcconsumer.NewConsumerConsistency(specId)
	consumerCmdFlags := common.ConsumerCmdFlags{}
	rpcsonumerLogs, err := metrics.NewRPCConsumerLogs(nil, nil)
	require.NoError(t, err)
	err = rpcConsumerServer.ServeRPCRequests(ctx, rpcEndpoint, consumerStateTracker, chainParser, finalizationConsensus, consumerSessionManager, requiredResponses, account.SK, lavaChainID, nil, rpcsonumerLogs, account.Addr, consumerConsistency, nil, consumerCmdFlags, false, nil, nil)
	require.NoError(t, err)

	// wait for consumer server to be up
	consumerUp := checkServerStatusWithTimeout("http://"+consumerListenAddress, time.Millisecond*50)
	require.True(t, consumerUp)

	client := http.Client{}
	resp, err := client.Get("http://" + consumerListenAddress + "/status")
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	resp.Body.Close()
}
