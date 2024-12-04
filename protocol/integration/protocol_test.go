package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	slices "github.com/lavanet/lava/v4/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v4/ecosystem/cache"
	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol/finalizationconsensus"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/protocol/performance"
	"github.com/lavanet/lava/v4/protocol/provideroptimizer"
	"github.com/lavanet/lava/v4/protocol/rpcconsumer"
	"github.com/lavanet/lava/v4/protocol/rpcprovider"
	"github.com/lavanet/lava/v4/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/v4/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	"github.com/lavanet/lava/v4/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/connectivity"

	conflicttypes "github.com/lavanet/lava/v4/x/conflict/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

var (
	seed                                    int64
	randomizer                              *sigs.ZeroReader
	addressGen                              uniqueAddressGenerator
	numberOfRetriesOnNodeErrorsProviderSide int = 2
)

func TestMain(m *testing.M) {
	// This code will run once before any test cases are executed.
	seed = time.Now().Unix()
	rand.SetSpecificSeed(seed)
	addressGen = NewUniqueAddressGenerator()
	randomizer = sigs.NewZeroReader(seed)
	lavasession.AllowInsecureConnectionToProviders = true
	// Run the actual tests
	exitCode := m.Run()
	if exitCode != 0 {
		utils.LavaFormatDebug("failed tests seed", utils.Attribute{Key: "seed", Value: seed})
	}
	os.Exit(exitCode)
}

func isGrpcServerUp(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()
	conn, err := lavasession.ConnectGRPCClient(context.Background(), url, true, false, false)
	if err != nil {
		return false
	}
	defer conn.Close()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return true
		} else if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			return false
		}

		select {
		case <-time.After(10 * time.Millisecond):
			// Check the connection state again after a short delay
		case <-ctx.Done():
			// The context has timed out
			return false
		}
	}
}

func checkGrpcServerStatusWithTimeout(url string, totalTimeout time.Duration) bool {
	startTime := time.Now()

	for time.Since(startTime) < totalTimeout {
		if isGrpcServerUp(url) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}

	return false
}

func isServerUp(urlPath string) bool {
	u, err := url.Parse(urlPath)
	if err != nil {
		panic(err)
	}

	switch {
	case u.Scheme == "http":
		return isHttpServerUp(urlPath)
	case u.Scheme == "ws":
		return isWebsocketServerUp(urlPath)
	default:
		panic("unsupported scheme")
	}
}

func isHttpServerUp(urlPath string) bool {
	client := http.Client{
		Timeout: 20 * time.Millisecond,
	}

	resp, err := client.Get(urlPath)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.ContentLength > 0
}

func isWebsocketServerUp(urlPath string) bool {
	client, _, err := websocket.DefaultDialer.Dial(urlPath, nil)
	if err != nil {
		return false
	}
	client.Close()
	return true
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

func createInMemoryRewardDb(specs []string) (*rewardserver.RewardDB, error) {
	rewardDB := rewardserver.NewRewardDB()
	for _, spec := range specs {
		db := rewardserver.NewMemoryDB(spec)
		err := rewardDB.AddDB(db)
		if err != nil {
			return nil, err
		}
	}
	return rewardDB, nil
}

type PolicySt struct {
	addons       []string
	extensions   []string
	apiInterface string
}

func (a PolicySt) GetSupportedAddons(string) ([]string, error) {
	return a.addons, nil
}

func (a PolicySt) GetSupportedExtensions(string) ([]epochstoragetypes.EndpointService, error) {
	ret := []epochstoragetypes.EndpointService{}
	for _, ext := range a.extensions {
		ret = append(ret, epochstoragetypes.EndpointService{Extension: ext, ApiInterface: a.apiInterface})
	}
	return ret, nil
}

type rpcConsumerOptions struct {
	specId                string
	apiInterface          string
	account               sigs.Account
	consumerListenAddress string
	epoch                 uint64
	pairingList           map[uint64]*lavasession.ConsumerSessionsWithProvider
	requiredResponses     int
	lavaChainID           string
	cacheListenAddress    string
}

type rpcConsumerOut struct {
	rpcConsumerServer        *rpcconsumer.RPCConsumerServer
	mockConsumerStateTracker *mockConsumerStateTracker
	cache                    *performance.Cache
}

func createRpcConsumer(t *testing.T, ctx context.Context, rpcConsumerOptions rpcConsumerOptions) rpcConsumerOut {
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
	})
	chainParser, _, chainFetcher, _, _, err := chainlib.CreateChainLibMocks(ctx, rpcConsumerOptions.specId, rpcConsumerOptions.apiInterface, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainFetcher)

	rpcConsumerServer := &rpcconsumer.RPCConsumerServer{}
	rpcEndpoint := &lavasession.RPCEndpoint{
		NetworkAddress:  rpcConsumerOptions.consumerListenAddress,
		ChainID:         rpcConsumerOptions.specId,
		ApiInterface:    rpcConsumerOptions.apiInterface,
		TLSEnabled:      false,
		HealthCheckPath: "",
		Geolocation:     1,
	}
	consumerStateTracker := &mockConsumerStateTracker{}
	finalizationConsensus := finalizationconsensus.NewFinalizationConsensus(rpcEndpoint.ChainID)
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	baseLatency := common.AverageWorldLatency / 2
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, averageBlockTime, baseLatency, 2, nil, "dontcare")
	consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil, "test", lavasession.NewActiveSubscriptionProvidersStorage())
	consumerSessionManager.UpdateAllProviders(rpcConsumerOptions.epoch, rpcConsumerOptions.pairingList)

	// Just setting the providers available extensions and policy so the consumer is aware of them
	addons := []string{}
	extensions := []string{}
	for _, provider := range rpcConsumerOptions.pairingList {
		for _, endpoint := range provider.Endpoints {
			for addon := range endpoint.Addons {
				if !slices.Contains(addons, addon) {
					addons = append(addons, addon)
				}
				if !slices.Contains(extensions, addon) {
					extensions = append(extensions, addon)
				}
			}
			for extension := range endpoint.Extensions {
				if !slices.Contains(extensions, extension) {
					extensions = append(extensions, extension)
				}
				if !slices.Contains(addons, extension) {
					addons = append(addons, extension)
				}
			}
		}
	}
	policy := PolicySt{
		addons:       addons,
		extensions:   extensions,
		apiInterface: rpcConsumerOptions.apiInterface,
	}
	chainParser.SetPolicy(policy, rpcConsumerOptions.specId, rpcConsumerOptions.apiInterface)

	var cache *performance.Cache = nil
	if rpcConsumerOptions.cacheListenAddress != "" {
		cache, err = performance.InitCache(ctx, rpcConsumerOptions.cacheListenAddress)
		if err != nil {
			t.Fatalf("Failed To Connect to cache at address %s: %v", rpcConsumerOptions.cacheListenAddress, err)
		}
	}

	consumerConsistency := rpcconsumer.NewConsumerConsistency(rpcConsumerOptions.specId)
	consumerCmdFlags := common.ConsumerCmdFlags{}
	rpcconsumerLogs, err := metrics.NewRPCConsumerLogs(nil, nil, nil)
	require.NoError(t, err)
	err = rpcConsumerServer.ServeRPCRequests(ctx, rpcEndpoint, consumerStateTracker, chainParser, finalizationConsensus, consumerSessionManager, rpcConsumerOptions.requiredResponses, rpcConsumerOptions.account.SK, rpcConsumerOptions.lavaChainID, cache, rpcconsumerLogs, rpcConsumerOptions.account.Addr, consumerConsistency, nil, consumerCmdFlags, false, nil, nil, nil)
	require.NoError(t, err)

	// wait for consumer to finish initialization
	listeningAddr := rpcConsumerServer.GetListeningAddress()
	for listeningAddr == "" {
		time.Sleep(10 * time.Millisecond)
		listeningAddr = rpcConsumerServer.GetListeningAddress()
	}

	// wait for consumer server to be up
	consumerUp := checkServerStatusWithTimeout("http://"+listeningAddr, time.Millisecond*61)
	require.True(t, consumerUp)
	if rpcEndpoint.ApiInterface == "tendermintrpc" || rpcEndpoint.ApiInterface == "jsonrpc" {
		consumerUp = checkServerStatusWithTimeout("ws://"+listeningAddr+"/ws", time.Millisecond*61)
		require.True(t, consumerUp)
	}

	return rpcConsumerOut{rpcConsumerServer, consumerStateTracker, cache}
}

type rpcProviderOptions struct {
	consumerAddress    string
	specId             string
	apiInterface       string
	listenAddress      string
	account            sigs.Account
	lavaChainID        string
	addons             []string
	providerUniqueId   string
	cacheListenAddress string
}

func createRpcProvider(t *testing.T, ctx context.Context, rpcProviderOptions rpcProviderOptions) (*rpcprovider.RPCProviderServer, *lavasession.RPCProviderEndpoint, *ReplySetter, *MockChainFetcher, *MockReliabilityManager) {
	replySetter := ReplySetter{
		status:       http.StatusOK,
		replyDataBuf: []byte(`{"reply": "REPLY-STUB"}`),
		handler:      nil,
	}
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response

		status := replySetter.status
		data := replySetter.replyDataBuf
		if replySetter.handler != nil {
			data = make([]byte, r.ContentLength)
			r.Body.Read(data)
			data, status = replySetter.handler(data, r.Header)
		}
		w.WriteHeader(status)
		fmt.Fprint(w, string(data))
	})

	chainParser, chainRouter, chainFetcher, _, endpoint, err := chainlib.CreateChainLibMocks(ctx, rpcProviderOptions.specId, rpcProviderOptions.apiInterface, serverHandler, nil, "../../", rpcProviderOptions.addons)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainFetcher)
	require.NotNil(t, chainRouter)
	endpoint.NetworkAddress.Address = rpcProviderOptions.listenAddress

	rpcProviderServer := &rpcprovider.RPCProviderServer{}
	if rpcProviderOptions.providerUniqueId != "" {
		rpcProviderServer.SetProviderUniqueId(rpcProviderOptions.providerUniqueId)
	}

	rpcProviderEndpoint := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{
			Address:    endpoint.NetworkAddress.Address,
			KeyPem:     "",
			CertPem:    "",
			DisableTLS: false,
		},
		ChainID:      rpcProviderOptions.specId,
		ApiInterface: rpcProviderOptions.apiInterface,
		Geolocation:  1,
		NodeUrls: []common.NodeUrl{
			{
				Url:               endpoint.NodeUrls[0].Url,
				InternalPath:      "",
				AuthConfig:        common.AuthConfig{},
				IpForwarding:      false,
				Timeout:           0,
				Addons:            rpcProviderOptions.addons,
				SkipVerifications: []string{},
			},
		},
	}
	rewardDB, err := createInMemoryRewardDb([]string{rpcProviderOptions.specId})
	require.NoError(t, err)
	_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
	mockProviderStateTracker := mockProviderStateTracker{consumerAddressForPairing: rpcProviderOptions.consumerAddress, averageBlockTime: averageBlockTime}
	rws := rewardserver.NewRewardServer(&mockProviderStateTracker, nil, rewardDB, "badger_test", 1, 10, nil)

	blockMemorySize, err := mockProviderStateTracker.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx)
	require.NoError(t, err)
	providerSessionManager := lavasession.NewProviderSessionManager(rpcProviderEndpoint, blockMemorySize)
	providerPolicy := rpcprovider.GetAllAddonsAndExtensionsFromNodeUrlSlice(rpcProviderEndpoint.NodeUrls)
	chainParser.SetPolicy(providerPolicy, rpcProviderOptions.specId, rpcProviderOptions.apiInterface)

	blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		BlocksToSave:        blocksToSaveChainTracker,
		AverageBlockTime:    averageBlockTime,
		ServerBlockMemory:   rpcprovider.ChainTrackerDefaultMemory + blocksToSaveChainTracker,
		NewLatestCallback:   nil,
		ConsistencyCallback: nil,
		Pmetrics:            nil,
	}

	var cache *performance.Cache = nil
	if rpcProviderOptions.cacheListenAddress != "" {
		cache, err = performance.InitCache(ctx, rpcProviderOptions.cacheListenAddress)
		if err != nil {
			t.Fatalf("Failed To Connect to cache at address %s: %v", rpcProviderOptions.cacheListenAddress, err)
		}
	}

	mockChainFetcher := NewMockChainFetcher(1000, int64(blocksToSaveChainTracker), nil)
	chainTracker, err := chaintracker.NewChainTracker(ctx, mockChainFetcher, chainTrackerConfig)
	require.NoError(t, err)
	chainTracker.StartAndServe(ctx)
	reliabilityManager := reliabilitymanager.NewReliabilityManager(chainTracker, &mockProviderStateTracker, rpcProviderOptions.account.Addr.String(), chainRouter, chainParser)
	mockReliabilityManager := NewMockReliabilityManager(reliabilityManager)
	rpcProviderServer.ServeRPCRequests(ctx, rpcProviderEndpoint, chainParser, rws, providerSessionManager, mockReliabilityManager, rpcProviderOptions.account.SK, cache, chainRouter, &mockProviderStateTracker, rpcProviderOptions.account.Addr, rpcProviderOptions.lavaChainID, rpcprovider.DEFAULT_ALLOWED_MISSING_CU, nil, nil, nil, false, nil, numberOfRetriesOnNodeErrorsProviderSide)
	listener := rpcprovider.NewProviderListener(ctx, rpcProviderEndpoint.NetworkAddress, "/health")
	err = listener.RegisterReceiver(rpcProviderServer, rpcProviderEndpoint)
	require.NoError(t, err)
	chainParser.Activate()
	chainTracker.RegisterForBlockTimeUpdates(chainParser)
	providerUp := checkGrpcServerStatusWithTimeout(rpcProviderEndpoint.NetworkAddress.Address, time.Millisecond*261)
	require.True(t, providerUp)
	return rpcProviderServer, endpoint, &replySetter, mockChainFetcher, mockReliabilityManager
}

func createCacheServer(t *testing.T, ctx context.Context, listenAddress string) {
	go func() {
		cs := cache.CacheServer{CacheMaxCost: 2 * 1024 * 1024 * 1024} // taken from max-items default value
		cs.InitCache(
			ctx,
			cache.DefaultExpirationTimeFinalized,
			cache.DefaultExpirationForNonFinalized,
			cache.DefaultExpirationNodeErrors,
			cache.DefaultExpirationBlocksHashesToHeights,
			"disabled",
			cache.DefaultExpirationTimeFinalizedMultiplier,
			cache.DefaultExpirationTimeNonFinalizedMultiplier,
		)
		cs.Serve(ctx, listenAddress)
	}()
	cacheServerUp := checkServerStatusWithTimeout("http://"+listenAddress, time.Second*7)
	require.True(t, cacheServerUp)
}

func TestConsumerProviderBasic(t *testing.T) {
	ctx := context.Background()
	// can be any spec and api interface
	specId := "LAV1"
	apiInterface := spectypes.APIInterfaceRest
	epoch := uint64(100)
	lavaChainID := "lava"

	numProviders := 1

	consumerListenAddress := addressGen.GetAddress()
	pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	type providerData struct {
		account          sigs.Account
		endpoint         *lavasession.RPCProviderEndpoint
		server           *rpcprovider.RPCProviderServer
		replySetter      *ReplySetter
		mockChainFetcher *MockChainFetcher
	}
	providers := []providerData{}

	for i := 0; i < numProviders; i++ {
		account := sigs.GenerateDeterministicFloatingKey(randomizer)
		providerDataI := providerData{account: account}
		providers = append(providers, providerDataI)
	}
	consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
	for i := 0; i < numProviders; i++ {
		ctx := context.Background()
		providerDataI := providers[i]
		listenAddress := addressGen.GetAddress()
		rpcProviderOptions := rpcProviderOptions{
			consumerAddress:  consumerAccount.Addr.String(),
			specId:           specId,
			apiInterface:     apiInterface,
			listenAddress:    listenAddress,
			account:          providerDataI.account,
			lavaChainID:      lavaChainID,
			addons:           []string(nil),
			providerUniqueId: fmt.Sprintf("provider%d", i),
		}
		providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
	}
	for i := 0; i < numProviders; i++ {
		pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
			PublicLavaAddress: providers[i].account.Addr.String(),
			Endpoints: []*lavasession.Endpoint{
				{
					NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
					Enabled:        true,
					Geolocation:    1,
				},
			},
			Sessions:         map[int64]*lavasession.SingleConsumerSession{},
			MaxComputeUnits:  10000,
			UsedComputeUnits: 0,
			PairingEpoch:     epoch,
		}
	}

	rpcConsumerOptions := rpcConsumerOptions{
		specId:                specId,
		apiInterface:          apiInterface,
		account:               consumerAccount,
		consumerListenAddress: consumerListenAddress,
		epoch:                 epoch,
		pairingList:           pairingList,
		requiredResponses:     1,
		lavaChainID:           lavaChainID,
	}
	rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
	require.NotNil(t, rpcConsumerOut.rpcConsumerServer)
	client := http.Client{}
	resp, err := client.Get("http://" + consumerListenAddress + "/cosmos/base/tendermint/v1beta1/blocks/latest")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, providers[0].replySetter.replyDataBuf, bodyBytes)
	resp.Body.Close()
}

func TestConsumerProviderWithProviders(t *testing.T) {
	playbook := []struct {
		name     string
		scenario int
	}{
		{
			name:     "basic-success",
			scenario: 0,
		},
		{
			name:     "with errors",
			scenario: 1,
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ctx := context.Background()
			// can be any spec and api interface
			specId := "LAV1"
			apiInterface := spectypes.APIInterfaceTendermintRPC
			epoch := uint64(100)
			lavaChainID := "lava"
			numProviders := 5

			consumerListenAddress := addressGen.GetAddress()
			pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
			type providerData struct {
				account          sigs.Account
				endpoint         *lavasession.RPCProviderEndpoint
				server           *rpcprovider.RPCProviderServer
				replySetter      *ReplySetter
				mockChainFetcher *MockChainFetcher
			}
			providers := []providerData{}

			for i := 0; i < numProviders; i++ {
				account := sigs.GenerateDeterministicFloatingKey(randomizer)
				providerDataI := providerData{account: account}
				providers = append(providers, providerDataI)
			}
			consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
			for i := 0; i < numProviders; i++ {
				ctx := context.Background()
				providerDataI := providers[i]
				listenAddress := addressGen.GetAddress()
				rpcProviderOptions := rpcProviderOptions{
					consumerAddress:  consumerAccount.Addr.String(),
					specId:           specId,
					apiInterface:     apiInterface,
					listenAddress:    listenAddress,
					account:          providerDataI.account,
					lavaChainID:      lavaChainID,
					addons:           []string(nil),
					providerUniqueId: fmt.Sprintf("provider%d", i),
				}
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
				providers[i].replySetter.replyDataBuf = []byte(fmt.Sprintf(`{"reply": %d}`, i+1))
			}
			for i := 0; i < numProviders; i++ {
				pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
					PublicLavaAddress: providers[i].account.Addr.String(),
					Endpoints: []*lavasession.Endpoint{
						{
							NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
							Enabled:        true,
							Geolocation:    1,
						},
					},
					Sessions:         map[int64]*lavasession.SingleConsumerSession{},
					MaxComputeUnits:  10000,
					UsedComputeUnits: 0,
					PairingEpoch:     epoch,
				}
			}

			rpcConsumerOptions := rpcConsumerOptions{
				specId:                specId,
				apiInterface:          apiInterface,
				account:               consumerAccount,
				consumerListenAddress: consumerListenAddress,
				epoch:                 epoch,
				pairingList:           pairingList,
				requiredResponses:     1,
				lavaChainID:           lavaChainID,
			}
			rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
			require.NotNil(t, rpcConsumerOut.rpcConsumerServer)
			if play.scenario != 1 {
				counter := map[int]int{}
				for i := 0; i <= 1000; i++ {
					client := http.Client{}
					resp, err := client.Get("http://" + consumerListenAddress + "/status")
					require.NoError(t, err)
					require.Equal(t, http.StatusOK, resp.StatusCode)
					bodyBytes, err := io.ReadAll(resp.Body)
					require.NoError(t, err)
					resp.Body.Close()
					mapi := map[string]int{}
					err = json.Unmarshal(bodyBytes, &mapi)
					require.NoError(t, err)
					id, ok := mapi["reply"]
					require.True(t, ok)
					counter[id]++
					handler := func(req []byte, header http.Header) (data []byte, status int) {
						time.Sleep(3 * time.Millisecond) // cause timeout for providers we got a reply for so others get chosen with a bigger likelihood
						return providers[id-1].replySetter.replyDataBuf, http.StatusOK
					}
					providers[id-1].replySetter.handler = handler
				}

				require.Len(t, counter, numProviders) // make sure to talk with all of them
			}
			if play.scenario != 0 {
				// add a chance for node errors and timeouts
				for i := 0; i < numProviders; i++ {
					replySetter := providers[i].replySetter
					index := i
					handler := func(req []byte, header http.Header) (data []byte, status int) {
						randVal := rand.Intn(10)
						switch randVal {
						case 1:
							if index < (numProviders+1)/2 {
								time.Sleep(2 * time.Second) // cause timeout, but only possible on half the providers so there's always a provider that answers
							}
						case 2, 3, 4:
							return []byte(`{"message":"bad","code":123}`), http.StatusServiceUnavailable
						case 5:
							return []byte(`{"message":"bad","code":777}`), http.StatusTooManyRequests // cause protocol error
						}
						return replySetter.replyDataBuf, http.StatusOK
					}
					providers[i].replySetter.handler = handler
				}

				seenError := false
				statuses := map[int]struct{}{}
				for i := 0; i <= 100; i++ {
					client := http.Client{}
					req, err := http.NewRequest("GET", "http://"+consumerListenAddress+"/status", nil)
					require.NoError(t, err)

					// Add custom headers to the request
					req.Header.Add(common.RELAY_TIMEOUT_HEADER_NAME, "90ms")

					// Perform the request
					resp, err := client.Do(req)
					require.NoError(t, err, i)
					if resp.StatusCode == http.StatusServiceUnavailable {
						seenError = true
					}
					statuses[resp.StatusCode] = struct{}{}
					require.NotEqual(t, resp.StatusCode, http.StatusTooManyRequests, i) // should never return too many requests, because it triggers a retry
					resp.Body.Close()
				}
				require.True(t, seenError, statuses)
			}
		})
	}
}

func TestConsumerProviderTx(t *testing.T) {
	playbook := []struct {
		name string
	}{
		{
			name: "basic-tx",
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ctx := context.Background()
			// can be any spec and api interface
			specId := "LAV1"
			apiInterface := spectypes.APIInterfaceRest
			epoch := uint64(100)
			lavaChainID := "lava"
			numProviders := 5

			consumerListenAddress := addressGen.GetAddress()
			pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
			type providerData struct {
				account          sigs.Account
				endpoint         *lavasession.RPCProviderEndpoint
				server           *rpcprovider.RPCProviderServer
				replySetter      *ReplySetter
				mockChainFetcher *MockChainFetcher
			}
			providers := []providerData{}

			for i := 0; i < numProviders; i++ {
				// providerListenAddress := "localhost:111" + strconv.Itoa(i)
				account := sigs.GenerateDeterministicFloatingKey(randomizer)
				providerDataI := providerData{account: account}
				providers = append(providers, providerDataI)
			}
			consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
			for i := 0; i < numProviders; i++ {
				ctx := context.Background()
				providerDataI := providers[i]
				listenAddress := addressGen.GetAddress()
				rpcProviderOptions := rpcProviderOptions{
					consumerAddress:  consumerAccount.Addr.String(),
					specId:           specId,
					apiInterface:     apiInterface,
					listenAddress:    listenAddress,
					account:          providerDataI.account,
					lavaChainID:      lavaChainID,
					addons:           []string(nil),
					providerUniqueId: fmt.Sprintf("provider%d", i),
				}
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
				providers[i].replySetter.replyDataBuf = []byte(fmt.Sprintf(`{"result": %d}`, i+1))
			}
			for i := 0; i < numProviders; i++ {
				pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
					PublicLavaAddress: providers[i].account.Addr.String(),
					Endpoints: []*lavasession.Endpoint{
						{
							NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
							Enabled:        true,
							Geolocation:    1,
						},
					},
					Sessions:         map[int64]*lavasession.SingleConsumerSession{},
					MaxComputeUnits:  10000,
					UsedComputeUnits: 0,
					PairingEpoch:     epoch,
				}
			}

			rpcConsumerOptions := rpcConsumerOptions{
				specId:                specId,
				apiInterface:          apiInterface,
				account:               consumerAccount,
				consumerListenAddress: consumerListenAddress,
				epoch:                 epoch,
				pairingList:           pairingList,
				requiredResponses:     1,
				lavaChainID:           lavaChainID,
			}
			rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
			require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

			for i := 0; i < numProviders; i++ {
				replySetter := providers[i].replySetter
				index := i
				handler := func(req []byte, header http.Header) (data []byte, status int) {
					if index == 1 {
						// only one provider responds correctly, but after a delay
						time.Sleep(20 * time.Millisecond)
						return replySetter.replyDataBuf, http.StatusOK
					} else {
						return []byte(`{"message":"bad","code":777}`), http.StatusInternalServerError
					}
				}
				providers[i].replySetter.handler = handler
			}

			client := http.Client{Timeout: 500 * time.Millisecond}
			req, err := http.NewRequest(http.MethodPost, "http://"+consumerListenAddress+"/cosmos/tx/v1beta1/txs", nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, `{"result": 2}`, string(bodyBytes))
		})
	}
}

func TestConsumerProviderJsonRpcWithNullID(t *testing.T) {
	playbook := []struct {
		name         string
		specId       string
		method       string
		expected     string
		apiInterface string
	}{
		{
			name:         "jsonrpc",
			specId:       "ETH1",
			method:       "eth_blockNumber",
			expected:     `{"jsonrpc":"2.0","id":null,"result":{}}`,
			apiInterface: spectypes.APIInterfaceJsonRPC,
		},
		{
			name:         "tendermintrpc",
			specId:       "LAV1",
			method:       "status",
			expected:     `{"jsonrpc":"2.0","result":{}}`,
			apiInterface: spectypes.APIInterfaceTendermintRPC,
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ctx := context.Background()
			// can be any spec and api interface
			specId := play.specId
			apiInterface := play.apiInterface
			epoch := uint64(100)
			lavaChainID := "lava"
			numProviders := 5

			consumerListenAddress := addressGen.GetAddress()
			pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
			type providerData struct {
				account          sigs.Account
				endpoint         *lavasession.RPCProviderEndpoint
				server           *rpcprovider.RPCProviderServer
				replySetter      *ReplySetter
				mockChainFetcher *MockChainFetcher
			}
			providers := []providerData{}

			for i := 0; i < numProviders; i++ {
				account := sigs.GenerateDeterministicFloatingKey(randomizer)
				providerDataI := providerData{account: account}
				providers = append(providers, providerDataI)
			}
			consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
			for i := 0; i < numProviders; i++ {
				ctx := context.Background()
				providerDataI := providers[i]
				listenAddress := addressGen.GetAddress()
				rpcProviderOptions := rpcProviderOptions{
					consumerAddress:  consumerAccount.Addr.String(),
					specId:           specId,
					apiInterface:     apiInterface,
					listenAddress:    listenAddress,
					account:          providerDataI.account,
					lavaChainID:      lavaChainID,
					addons:           []string(nil),
					providerUniqueId: fmt.Sprintf("provider%d", i),
				}
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
				providers[i].replySetter.replyDataBuf = []byte(fmt.Sprintf(`{"result": %d}`, i+1))
			}
			for i := 0; i < numProviders; i++ {
				pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
					PublicLavaAddress: providers[i].account.Addr.String(),
					Endpoints: []*lavasession.Endpoint{
						{
							NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
							Enabled:        true,
							Geolocation:    1,
						},
					},
					Sessions:         map[int64]*lavasession.SingleConsumerSession{},
					MaxComputeUnits:  10000,
					UsedComputeUnits: 0,
					PairingEpoch:     epoch,
				}
			}

			rpcConsumerOptions := rpcConsumerOptions{
				specId:                specId,
				apiInterface:          apiInterface,
				account:               consumerAccount,
				consumerListenAddress: consumerListenAddress,
				epoch:                 epoch,
				pairingList:           pairingList,
				requiredResponses:     1,
				lavaChainID:           lavaChainID,
			}
			rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
			require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

			for i := 0; i < numProviders; i++ {
				handler := func(req []byte, header http.Header) (data []byte, status int) {
					var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
					err := json.Unmarshal(req, &jsonRpcMessage)
					require.NoError(t, err)

					response := fmt.Sprintf(`{"jsonrpc":"2.0","result": {}, "id": %v}`, string(jsonRpcMessage.ID))
					return []byte(response), http.StatusOK
				}
				providers[i].replySetter.handler = handler
			}

			client := http.Client{Timeout: 500 * time.Millisecond}
			jsonMsg := fmt.Sprintf(`{"jsonrpc":"2.0","method":"%v","params": [], "id":null}`, play.method)
			msgBuffer := bytes.NewBuffer([]byte(jsonMsg))
			req, err := http.NewRequest(http.MethodPost, "http://"+consumerListenAddress, msgBuffer)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode, string(bodyBytes))

			resp.Body.Close()

			require.Equal(t, play.expected, string(bodyBytes))
		})
	}
}

func TestConsumerProviderSubscriptionsHappyFlow(t *testing.T) {
	playbook := []struct {
		name         string
		specId       string
		method       string
		expected     string
		apiInterface string
	}{
		{
			name:         "jsonrpc",
			specId:       "ETH1",
			method:       "eth_blockNumber",
			expected:     `{"jsonrpc":"2.0","id":null,"result":{}}`,
			apiInterface: spectypes.APIInterfaceJsonRPC,
		},
		{
			name:         "tendermintrpc",
			specId:       "LAV1",
			method:       "status",
			expected:     `{"jsonrpc":"2.0","result":{}}`,
			apiInterface: spectypes.APIInterfaceTendermintRPC,
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ctx := context.Background()
			// can be any spec and api interface
			specId := play.specId
			apiInterface := play.apiInterface
			epoch := uint64(100)
			lavaChainID := "lava"
			numProviders := 5

			consumerListenAddress := addressGen.GetAddress()
			pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
			type providerData struct {
				account          sigs.Account
				endpoint         *lavasession.RPCProviderEndpoint
				server           *rpcprovider.RPCProviderServer
				replySetter      *ReplySetter
				mockChainFetcher *MockChainFetcher
			}
			providers := []providerData{}

			for i := 0; i < numProviders; i++ {
				account := sigs.GenerateDeterministicFloatingKey(randomizer)
				providerDataI := providerData{account: account}
				providers = append(providers, providerDataI)
			}
			consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
			for i := 0; i < numProviders; i++ {
				ctx := context.Background()
				providerDataI := providers[i]
				listenAddress := addressGen.GetAddress()
				rpcProviderOptions := rpcProviderOptions{
					consumerAddress:  consumerAccount.Addr.String(),
					specId:           specId,
					apiInterface:     apiInterface,
					listenAddress:    listenAddress,
					account:          providerDataI.account,
					lavaChainID:      lavaChainID,
					addons:           []string(nil),
					providerUniqueId: fmt.Sprintf("provider%d", i),
				}
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
				providers[i].replySetter.replyDataBuf = []byte(fmt.Sprintf(`{"result": %d}`, i+1))
			}
			for i := 0; i < numProviders; i++ {
				pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
					PublicLavaAddress: providers[i].account.Addr.String(),
					Endpoints: []*lavasession.Endpoint{
						{
							NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
							Enabled:        true,
							Geolocation:    1,
						},
					},
					Sessions:         map[int64]*lavasession.SingleConsumerSession{},
					MaxComputeUnits:  10000,
					UsedComputeUnits: 0,
					PairingEpoch:     epoch,
				}
			}

			rpcConsumerOptions := rpcConsumerOptions{
				specId:                specId,
				apiInterface:          apiInterface,
				account:               consumerAccount,
				consumerListenAddress: consumerListenAddress,
				epoch:                 epoch,
				pairingList:           pairingList,
				requiredResponses:     1,
				lavaChainID:           lavaChainID,
			}
			rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
			require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

			for i := 0; i < numProviders; i++ {
				handler := func(req []byte, header http.Header) (data []byte, status int) {
					var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
					err := json.Unmarshal(req, &jsonRpcMessage)
					require.NoError(t, err)

					response := fmt.Sprintf(`{"jsonrpc":"2.0","result": {}, "id": %v}`, string(jsonRpcMessage.ID))
					return []byte(response), http.StatusOK
				}
				providers[i].replySetter.handler = handler
			}

			client := http.Client{Timeout: 500 * time.Millisecond}
			jsonMsg := fmt.Sprintf(`{"jsonrpc":"2.0","method":"%v","params": [], "id":null}`, play.method)
			msgBuffer := bytes.NewBuffer([]byte(jsonMsg))
			req, err := http.NewRequest(http.MethodPost, "http://"+consumerListenAddress, msgBuffer)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode, string(bodyBytes))

			resp.Body.Close()

			require.Equal(t, play.expected, string(bodyBytes))
		})
	}
}

func TestSameProviderConflictBasicResponseCheck(t *testing.T) {
	playbook := []struct {
		name                string
		shouldGetError      bool
		numOfProviders      int
		numOfLyingProviders int
	}{
		{
			name:                "one provider - fake block hashes - return conflict error",
			numOfProviders:      1,
			numOfLyingProviders: 1,
			shouldGetError:      true,
		},
		{
			name:                "multiple providers - only one with fake block hashes - return ok",
			numOfProviders:      2,
			numOfLyingProviders: 1,
			shouldGetError:      false,
		},
		{
			name:                "multiple providers - everyone is returning fake block hashes - return conflict error",
			numOfProviders:      3,
			numOfLyingProviders: 3,
			shouldGetError:      true,
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			require.LessOrEqual(t, play.numOfLyingProviders, play.numOfProviders, "too many lying providers")
			ctx := context.Background()
			// can be any spec and api interface
			specId := "LAV1"
			apiInterface := spectypes.APIInterfaceRest
			epoch := uint64(100)
			lavaChainID := "lava"
			numProviders := play.numOfProviders

			consumerListenAddress := addressGen.GetAddress()
			pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

			type providerData struct {
				account                sigs.Account
				endpoint               *lavasession.RPCProviderEndpoint
				server                 *rpcprovider.RPCProviderServer
				replySetter            *ReplySetter
				mockChainFetcher       *MockChainFetcher
				mockReliabilityManager *MockReliabilityManager
			}
			providers := []providerData{}

			for i := 0; i < numProviders; i++ {
				account := sigs.GenerateDeterministicFloatingKey(randomizer)
				providerDataI := providerData{account: account}
				providers = append(providers, providerDataI)
			}
			consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)

			for i := 0; i < numProviders; i++ {
				ctx := context.Background()
				providerDataI := providers[i]
				listenAddress := addressGen.GetAddress()
				rpcProviderOptions := rpcProviderOptions{
					consumerAddress:  consumerAccount.Addr.String(),
					specId:           specId,
					apiInterface:     apiInterface,
					listenAddress:    listenAddress,
					account:          providerDataI.account,
					lavaChainID:      lavaChainID,
					addons:           []string(nil),
					providerUniqueId: fmt.Sprintf("provider%d", i),
				}
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, providers[i].mockReliabilityManager = createRpcProvider(t, ctx, rpcProviderOptions)
				providers[i].replySetter.replyDataBuf = []byte(fmt.Sprintf(`{"result": %d}`, i+1))
			}

			for i := 0; i < numProviders; i++ {
				pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
					PublicLavaAddress: providers[i].account.Addr.String(),
					Endpoints: []*lavasession.Endpoint{
						{
							NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
							Enabled:        true,
							Geolocation:    1,
						},
					},
					Sessions:         map[int64]*lavasession.SingleConsumerSession{},
					MaxComputeUnits:  10000,
					UsedComputeUnits: 0,
					PairingEpoch:     epoch,
				}
			}

			rpcConsumerOptions := rpcConsumerOptions{
				specId:                specId,
				apiInterface:          apiInterface,
				account:               consumerAccount,
				consumerListenAddress: consumerListenAddress,
				epoch:                 epoch,
				pairingList:           pairingList,
				requiredResponses:     1,
				lavaChainID:           lavaChainID,
			}
			rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
			require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

			// Set first provider as a "liar", to return wrong block hashes
			getLatestBlockDataWrapper := func(rmi rpcprovider.ReliabilityManagerInf, fromBlock, toBlock, specificBlock int64) (int64, []*chaintracker.BlockStore, time.Time, error) {
				latestBlock, requestedHashes, changeTime, err := rmi.GetLatestBlockData(fromBlock, toBlock, specificBlock)

				for _, block := range requestedHashes {
					block.Hash += strconv.Itoa(int(rand.Int63()))
				}

				return latestBlock, requestedHashes, changeTime, err
			}

			for i := 0; i < play.numOfLyingProviders; i++ {
				providers[i].mockReliabilityManager.SetGetLatestBlockDataWrapper(getLatestBlockDataWrapper)
			}

			client := http.Client{Timeout: 500 * time.Millisecond}
			req, err := http.NewRequest(http.MethodPost, "http://"+consumerListenAddress+"/cosmos/tx/v1beta1/txs", nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			resp.Body.Close()
			if play.shouldGetError {
				require.Contains(t, string(bodyBytes), "found same provider conflict")
			} else {
				require.Equal(t, `{"result": 2}`, string(bodyBytes))
			}
		})
	}
}

func TestArchiveProvidersRetry(t *testing.T) {
	playbook := []struct {
		name               string
		numOfProviders     int
		archiveProviders   int
		nodeErrorProviders int
		expectedResult     string
		statusCode         int
	}{
		{
			name:               "archive with 1 errored provider",
			numOfProviders:     3,
			archiveProviders:   3,
			nodeErrorProviders: 1,
			expectedResult:     `{"result": "success"}`,
			statusCode:         200,
		},
		{
			name:               "happy flow",
			numOfProviders:     3,
			archiveProviders:   3,
			nodeErrorProviders: 0,
			expectedResult:     `{"result": "success"}`,
			statusCode:         200,
		},
		{
			name:               "archive with 2 errored provider",
			numOfProviders:     3,
			archiveProviders:   3,
			nodeErrorProviders: 2,
			expectedResult:     `{"result": "success"}`,
			statusCode:         200,
		},
		{
			name:               "archive with 3 errored provider",
			numOfProviders:     3,
			archiveProviders:   3,
			nodeErrorProviders: 3,
			expectedResult:     `{"error": "failure", "message": "test", "code": "-32132"}`,
			statusCode:         555,
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ctx := context.Background()
			// can be any spec and api interface
			specId := "LAV1"
			apiInterface := spectypes.APIInterfaceRest
			epoch := uint64(100)
			lavaChainID := "lava"
			numProviders := play.numOfProviders

			consumerListenAddress := addressGen.GetAddress()

			type providerData struct {
				account                sigs.Account
				endpoint               *lavasession.RPCProviderEndpoint
				server                 *rpcprovider.RPCProviderServer
				replySetter            *ReplySetter
				mockChainFetcher       *MockChainFetcher
				mockReliabilityManager *MockReliabilityManager
			}
			providers := []providerData{}

			for i := 0; i < numProviders; i++ {
				account := sigs.GenerateDeterministicFloatingKey(randomizer)
				providerDataI := providerData{account: account}
				providers = append(providers, providerDataI)
			}
			consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)

			for i := 0; i < numProviders; i++ {
				ctx := context.Background()
				providerDataI := providers[i]
				listenAddress := addressGen.GetAddress()
				addons := []string(nil)
				if i+1 <= play.archiveProviders {
					addons = []string{"archive"}
				}

				rpcProviderOptions := rpcProviderOptions{
					consumerAddress:  consumerAccount.Addr.String(),
					specId:           specId,
					apiInterface:     apiInterface,
					listenAddress:    listenAddress,
					account:          providerDataI.account,
					lavaChainID:      lavaChainID,
					addons:           addons,
					providerUniqueId: fmt.Sprintf("provider%d", i),
				}
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, providers[i].mockReliabilityManager = createRpcProvider(t, ctx, rpcProviderOptions)
				providers[i].replySetter.replyDataBuf = []byte(`{"result": "success"}`)
				if i+1 <= play.nodeErrorProviders {
					providers[i].replySetter.replyDataBuf = []byte(`{"error": "failure", "message": "test", "code": "-32132"}`)
					providers[i].replySetter.status = 555
				}
			}

			pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
			for i := 0; i < numProviders; i++ {
				extensions := map[string]struct{}{}
				if i+1 <= play.archiveProviders {
					extensions = map[string]struct{}{"archive": {}}
				}
				pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
					PublicLavaAddress: providers[i].account.Addr.String(),

					Endpoints: []*lavasession.Endpoint{
						{
							NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
							Enabled:        true,
							Geolocation:    1,
							Extensions:     extensions,
						},
					},
					Sessions:         map[int64]*lavasession.SingleConsumerSession{},
					MaxComputeUnits:  10000,
					UsedComputeUnits: 0,
					PairingEpoch:     epoch,
				}
			}

			rpcConsumerOptions := rpcConsumerOptions{
				specId:                specId,
				apiInterface:          apiInterface,
				account:               consumerAccount,
				consumerListenAddress: consumerListenAddress,
				epoch:                 epoch,
				pairingList:           pairingList,
				requiredResponses:     1,
				lavaChainID:           lavaChainID,
			}
			rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
			require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

			for i := 0; i < 10; i++ {
				client := http.Client{Timeout: 10000 * time.Millisecond}
				req, err := http.NewRequest(http.MethodGet, "http://"+consumerListenAddress+"/lavanet/lava/conflict/params", nil)
				req.Header["lava-extension"] = []string{"archive"}
				require.NoError(t, err)

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, play.statusCode, resp.StatusCode)

				bodyBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				resp.Body.Close()
				require.Equal(t, string(bodyBytes), play.expectedResult)
			}
		})
	}
}

func TestSameProviderConflictReport(t *testing.T) {
	type providerData struct {
		account                sigs.Account
		endpoint               *lavasession.RPCProviderEndpoint
		server                 *rpcprovider.RPCProviderServer
		replySetter            *ReplySetter
		mockChainFetcher       *MockChainFetcher
		mockReliabilityManager *MockReliabilityManager
	}

	createProvidersData := func(numProviders int) (providers []*providerData) {
		providers = []*providerData{}

		for i := 0; i < numProviders; i++ {
			account := sigs.GenerateDeterministicFloatingKey(randomizer)
			providerDataI := providerData{account: account}
			providers = append(providers, &providerDataI)
		}

		return providers
	}

	initProvidersData := func(consumerAccount sigs.Account, providers []*providerData, specId, apiInterface, lavaChainID string) {
		for i := 0; i < len(providers); i++ {
			ctx := context.Background()
			providerDataI := providers[i]
			listenAddress := addressGen.GetAddress()
			rpcProviderOptions := rpcProviderOptions{
				consumerAddress:  consumerAccount.Addr.String(),
				specId:           specId,
				apiInterface:     apiInterface,
				listenAddress:    listenAddress,
				account:          providerDataI.account,
				lavaChainID:      lavaChainID,
				addons:           []string(nil),
				providerUniqueId: fmt.Sprintf("provider%d", i),
			}
			providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, providers[i].mockReliabilityManager = createRpcProvider(t, ctx, rpcProviderOptions)
			providers[i].replySetter.replyDataBuf = []byte(fmt.Sprintf(`{"result": %d}`, i+1))
		}
	}

	initPairingList := func(providers []*providerData, epoch uint64) (pairingList map[uint64]*lavasession.ConsumerSessionsWithProvider) {
		pairingList = map[uint64]*lavasession.ConsumerSessionsWithProvider{}

		for i := 0; i < len(providers); i++ {
			pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
				PublicLavaAddress: providers[i].account.Addr.String(),
				Endpoints: []*lavasession.Endpoint{
					{
						NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
						Enabled:        true,
						Geolocation:    1,
					},
				},
				Sessions:         map[int64]*lavasession.SingleConsumerSession{},
				MaxComputeUnits:  10000,
				UsedComputeUnits: 0,
				PairingEpoch:     epoch,
			}
		}

		return pairingList
	}

	t.Run("same provider conflict report", func(t *testing.T) {
		ctx := context.Background()
		// can be any spec and api interface
		specId := "LAV1"
		apiInterface := spectypes.APIInterfaceRest
		epoch := uint64(100)
		lavaChainID := "lava"
		numProviders := 1

		consumerListenAddress := addressGen.GetAddress()

		providers := createProvidersData(numProviders)
		consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)

		initProvidersData(consumerAccount, providers, specId, apiInterface, lavaChainID)

		rpcConsumerOptions := rpcConsumerOptions{
			specId:                specId,
			apiInterface:          apiInterface,
			account:               consumerAccount,
			consumerListenAddress: consumerListenAddress,
			epoch:                 epoch,
			pairingList:           initPairingList(providers, epoch),
			requiredResponses:     1,
			lavaChainID:           lavaChainID,
		}
		rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)

		conflictSent := false
		wg := sync.WaitGroup{}
		wg.Add(1)
		txConflictDetectionMock := func(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, conflictHandler common.ConflictHandlerInterface) error {
			if finalizationConflict == nil {
				wg.Done()
				require.FailNow(t, "Finalization conflict should not be nil")
				return nil
			}
			utils.LavaFormatDebug("@@@@@@@@@@@@@@@ Called conflict mock tx", utils.LogAttr("provider0", finalizationConflict.RelayFinalization_0.RelaySession.Provider), utils.LogAttr("provider0", finalizationConflict.RelayFinalization_1.RelaySession.Provider))

			if finalizationConflict.RelayFinalization_0.RelaySession.Provider != finalizationConflict.RelayFinalization_1.RelaySession.Provider {
				require.FailNow(t, "Finalization conflict should not have different provider addresses")
			}

			if finalizationConflict.RelayFinalization_0.RelaySession.Provider != providers[0].account.Addr.String() {
				require.FailNow(t, "Finalization conflict provider address is not the provider address")
			}
			wg.Done()
			conflictSent = true
			return nil
		}
		rpcConsumerOut.mockConsumerStateTracker.SetTxConflictDetectionWrapper(txConflictDetectionMock)
		require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

		// Set first provider as a "liar", to return wrong block hashes
		getLatestBlockDataWrapper := func(rmi rpcprovider.ReliabilityManagerInf, fromBlock, toBlock, specificBlock int64) (int64, []*chaintracker.BlockStore, time.Time, error) {
			latestBlock, requestedHashes, changeTime, err := rmi.GetLatestBlockData(fromBlock, toBlock, specificBlock)

			for _, block := range requestedHashes {
				block.Hash += strconv.Itoa(int(rand.Int63()))
			}

			return latestBlock, requestedHashes, changeTime, err
		}

		providers[0].mockReliabilityManager.SetGetLatestBlockDataWrapper(getLatestBlockDataWrapper)

		client := http.Client{Timeout: 1 * time.Minute}
		clientCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(clientCtx, http.MethodPost, "http://"+consumerListenAddress+"/cosmos/tx/v1beta1/txs", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		// conflict calls happen concurrently, therefore we need to wait the call.
		wg.Wait()
		require.True(t, conflictSent)
	})

	t.Run("two providers conflict report", func(t *testing.T) {
		ctx := context.Background()
		// can be any spec and api interface
		specId := "LAV1"
		apiInterface := spectypes.APIInterfaceRest
		epoch := uint64(100)
		lavaChainID := "lava"
		numProviders := 2

		consumerListenAddress := addressGen.GetAddress()
		providers := createProvidersData(numProviders)
		consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)

		initProvidersData(consumerAccount, providers, specId, apiInterface, lavaChainID)

		rpcConsumerOptions := rpcConsumerOptions{
			specId:                specId,
			apiInterface:          apiInterface,
			account:               consumerAccount,
			consumerListenAddress: consumerListenAddress,
			epoch:                 epoch,
			pairingList:           initPairingList(providers, epoch),
			requiredResponses:     1,
			lavaChainID:           lavaChainID,
		}
		rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)

		twoProvidersConflictSent := false
		sameProviderConflictSent := false
		numberOfRelays := 10
		reported := make(chan bool, numberOfRelays)
		txConflictDetectionMock := func(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, conflictHandler common.ConflictHandlerInterface) error {
			if finalizationConflict == nil {
				require.FailNow(t, "Finalization conflict should not be nil")
				reported <- true
				return nil
			}
			utils.LavaFormatDebug("@@@@@@@@@@@@@@@ Called conflict mock tx", utils.LogAttr("provider0", finalizationConflict.RelayFinalization_0.RelaySession.Provider), utils.LogAttr("provider0", finalizationConflict.RelayFinalization_1.RelaySession.Provider))

			if finalizationConflict.RelayFinalization_0.RelaySession.Provider == finalizationConflict.RelayFinalization_1.RelaySession.Provider {
				utils.LavaFormatDebug("@@@@@@@@@@@@@@@ Called SAME conflict tx")
				sameProviderConflictSent = true
			}

			if finalizationConflict.RelayFinalization_0.RelaySession.Provider != providers[0].account.Addr.String() {
				require.FailNow(t, "Finalization conflict provider 0 address is not the first provider")
			}

			if !sameProviderConflictSent && finalizationConflict.RelayFinalization_1.RelaySession.Provider != providers[1].account.Addr.String() {
				require.FailNow(t, "Finalization conflict provider 1 address is not the first provider")
			}

			twoProvidersConflictSent = true
			reported <- true
			return nil
		}
		rpcConsumerOut.mockConsumerStateTracker.SetTxConflictDetectionWrapper(txConflictDetectionMock)
		require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

		// Set first provider as a "liar", to return wrong block hashes
		getLatestBlockDataWrapper := func(rmi rpcprovider.ReliabilityManagerInf, fromBlock, toBlock, specificBlock int64) (int64, []*chaintracker.BlockStore, time.Time, error) {
			latestBlock, requestedHashes, changeTime, err := rmi.GetLatestBlockData(fromBlock, toBlock, specificBlock)

			for _, block := range requestedHashes {
				block.Hash += strconv.Itoa(int(rand.Int63()))
			}

			return latestBlock, requestedHashes, changeTime, err
		}

		providers[0].mockReliabilityManager.SetGetLatestBlockDataWrapper(getLatestBlockDataWrapper)

		client := http.Client{Timeout: 1000 * time.Millisecond}
		req, err := http.NewRequest(http.MethodPost, "http://"+consumerListenAddress+"/cosmos/tx/v1beta1/txs", nil)
		require.NoError(t, err)

		for i := 0; i < numberOfRelays; i++ {
			// Two relays to trigger both same provider conflict and
			go func() {
				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
			}()
		}
		// conflict calls happen concurrently, therefore we need to wait the call.
		<-reported
		require.True(t, sameProviderConflictSent)
		require.True(t, twoProvidersConflictSent)
	})
}

func TestConsumerProviderStatic(t *testing.T) {
	ctx := context.Background()
	// can be any spec and api interface
	specId := "LAV1"
	apiInterface := spectypes.APIInterfaceTendermintRPC
	epoch := uint64(100)
	lavaChainID := "lava"

	numProviders := 1

	consumerListenAddress := addressGen.GetAddress()
	pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	type providerData struct {
		account          sigs.Account
		endpoint         *lavasession.RPCProviderEndpoint
		server           *rpcprovider.RPCProviderServer
		replySetter      *ReplySetter
		mockChainFetcher *MockChainFetcher
	}
	providers := []providerData{}

	for i := 0; i < numProviders; i++ {
		account := sigs.GenerateDeterministicFloatingKey(randomizer)
		providerDataI := providerData{account: account}
		providers = append(providers, providerDataI)
	}
	consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
	for i := 0; i < numProviders; i++ {
		ctx := context.Background()
		providerDataI := providers[i]
		listenAddress := addressGen.GetAddress()
		rpcProviderOptions := rpcProviderOptions{
			consumerAddress:  consumerAccount.Addr.String(),
			specId:           specId,
			apiInterface:     apiInterface,
			listenAddress:    listenAddress,
			account:          providerDataI.account,
			lavaChainID:      lavaChainID,
			addons:           []string(nil),
			providerUniqueId: fmt.Sprintf("provider%d", i),
		}
		providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
	}
	// provider is static
	for i := 0; i < numProviders; i++ {
		pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
			PublicLavaAddress: "BANANA" + strconv.Itoa(i),
			Endpoints: []*lavasession.Endpoint{
				{
					NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
					Enabled:        true,
					Geolocation:    1,
				},
			},
			Sessions:         map[int64]*lavasession.SingleConsumerSession{},
			MaxComputeUnits:  10000,
			UsedComputeUnits: 0,
			PairingEpoch:     epoch,
			StaticProvider:   true,
		}
	}

	rpcConsumerOptions := rpcConsumerOptions{
		specId:                specId,
		apiInterface:          apiInterface,
		account:               consumerAccount,
		consumerListenAddress: consumerListenAddress,
		epoch:                 epoch,
		pairingList:           pairingList,
		requiredResponses:     1,
		lavaChainID:           lavaChainID,
	}
	rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
	require.NotNil(t, rpcConsumerOut.rpcConsumerServer)
	client := http.Client{}
	// consumer sends the relay to a provider with an address BANANA+%d so the provider needs to skip validations for this to work
	resp, err := client.Get("http://" + consumerListenAddress + "/status")
	require.NoError(t, err)
	// we expect provider to fail the request on a verification
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	for i := 0; i < numProviders; i++ {
		providers[i].server.StaticProvider = true
	}
	resp, err = client.Get("http://" + consumerListenAddress + "/status")
	require.NoError(t, err)
	// we expect provider to fail the request on a verification
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, providers[0].replySetter.replyDataBuf, bodyBytes)
	resp.Body.Close()
}

func jsonRpcIdToInt(t *testing.T, rawID json.RawMessage) int {
	var idInterface interface{}
	err := json.Unmarshal(rawID, &idInterface)
	require.NoError(t, err)

	id, ok := idInterface.(float64)
	require.True(t, ok, idInterface)
	return int(id)
}

func waitForCondition(timeout, interval time.Duration, condition func() bool) bool {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return true
			}
		case <-timeoutTimer.C:
			return false
		}
	}
}

func TestConsumerProviderWithConsumerSideCache(t *testing.T) {
	ctx := context.Background()
	// can be any spec and api interface
	specId := "LAV1"
	apiInterface := spectypes.APIInterfaceTendermintRPC
	epoch := uint64(100)
	lavaChainID := "lava"

	consumerListenAddress := addressGen.GetAddress()
	cacheListenAddress := addressGen.GetAddress()
	pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	type providerData struct {
		account     sigs.Account
		endpoint    *lavasession.RPCProviderEndpoint
		replySetter *ReplySetter
	}

	consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
	providerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
	provider := providerData{account: providerAccount}
	listenAddress := addressGen.GetAddress()
	rpcProviderOptions := rpcProviderOptions{
		consumerAddress:  consumerAccount.Addr.String(),
		specId:           specId,
		apiInterface:     apiInterface,
		listenAddress:    listenAddress,
		account:          provider.account,
		lavaChainID:      lavaChainID,
		addons:           []string(nil),
		providerUniqueId: "provider",
	}

	var mockChainFetcher *MockChainFetcher
	_, provider.endpoint, provider.replySetter, mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
	provider.replySetter.handler = func(req []byte, header http.Header) (data []byte, status int) {
		var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
		err := json.Unmarshal(req, &jsonRpcMessage)
		require.NoError(t, err, req)

		response := fmt.Sprintf(`{"jsonrpc":"2.0","result": {}, "id": %v}`, string(jsonRpcMessage.ID))
		return []byte(response), http.StatusOK
	}

	pairingList[0] = &lavasession.ConsumerSessionsWithProvider{
		PublicLavaAddress: provider.account.Addr.String(),
		Endpoints: []*lavasession.Endpoint{
			{
				NetworkAddress: provider.endpoint.NetworkAddress.Address,
				Enabled:        true,
				Geolocation:    1,
			},
		},
		Sessions:         map[int64]*lavasession.SingleConsumerSession{},
		MaxComputeUnits:  10000,
		UsedComputeUnits: 0,
		PairingEpoch:     epoch,
	}

	createCacheServer(t, ctx, cacheListenAddress)

	rpcConsumerOptions := rpcConsumerOptions{
		specId:                specId,
		apiInterface:          apiInterface,
		account:               consumerAccount,
		consumerListenAddress: consumerListenAddress,
		epoch:                 epoch,
		pairingList:           pairingList,
		requiredResponses:     1,
		lavaChainID:           lavaChainID,
		cacheListenAddress:    cacheListenAddress,
	}
	rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
	require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

	client := http.Client{}
	id := 0
	sendMessage := func(method string, params []string) http.Header {
		// Get latest block
		body := fmt.Sprintf(`{"jsonrpc":"2.0","method":"%v","params": [%v], "id":%v}`, method, strings.Join(params, ","), id)
		resp, err := client.Post("http://"+consumerListenAddress, "application/json", bytes.NewBuffer([]byte(body)))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
		err = json.Unmarshal(bodyBytes, &jsonRpcMessage)
		require.NoError(t, err)

		respId := jsonRpcIdToInt(t, jsonRpcMessage.ID)
		require.Equal(t, id, respId)
		resp.Body.Close()
		id++

		return resp.Header
	}

	mockChainFetcher.SetBlock(1000)

	// Get latest for sanity check
	providerAddr := provider.account.Addr.String()
	headers := sendMessage("status", []string{})
	require.Equal(t, providerAddr, headers.Get(common.PROVIDER_ADDRESS_HEADER_NAME))

	// Get block, this should be cached for next time
	headers = sendMessage("block", []string{"1000"})
	require.Equal(t, providerAddr, headers.Get(common.PROVIDER_ADDRESS_HEADER_NAME))

	repliedWithCache := waitForCondition(2*time.Second, 100*time.Millisecond, func() bool {
		headers = sendMessage("block", []string{"1000"})
		return headers.Get(common.PROVIDER_ADDRESS_HEADER_NAME) == "Cached"
	})
	require.True(t, repliedWithCache)

	// This should be from cache
	headers = sendMessage("block", []string{"1000"})
	require.Equal(t, "Cached", headers.Get(common.PROVIDER_ADDRESS_HEADER_NAME))
}

func TestConsumerProviderWithProviderSideCache(t *testing.T) {
	ctx := context.Background()
	// can be any spec and api interface
	specId := "LAV1"
	apiInterface := spectypes.APIInterfaceTendermintRPC
	epoch := uint64(100)
	lavaChainID := "lava"

	consumerListenAddress := addressGen.GetAddress()
	cacheListenAddress := addressGen.GetAddress()
	pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	type providerData struct {
		account     sigs.Account
		endpoint    *lavasession.RPCProviderEndpoint
		replySetter *ReplySetter
	}

	consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
	providerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
	provider := providerData{account: providerAccount}
	providerListenAddress := addressGen.GetAddress()

	createCacheServer(t, ctx, cacheListenAddress)
	testJsonRpcId := 42
	nodeRequestsCounter := 0
	rpcProviderOptions := rpcProviderOptions{
		consumerAddress:    consumerAccount.Addr.String(),
		specId:             specId,
		apiInterface:       apiInterface,
		listenAddress:      providerListenAddress,
		account:            provider.account,
		lavaChainID:        lavaChainID,
		addons:             []string(nil),
		providerUniqueId:   "provider",
		cacheListenAddress: cacheListenAddress,
	}

	var mockChainFetcher *MockChainFetcher
	_, provider.endpoint, provider.replySetter, mockChainFetcher, _ = createRpcProvider(t, ctx, rpcProviderOptions)
	provider.replySetter.handler = func(req []byte, header http.Header) (data []byte, status int) {
		var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
		err := json.Unmarshal(req, &jsonRpcMessage)
		require.NoError(t, err, req)

		reqId := jsonRpcIdToInt(t, jsonRpcMessage.ID)
		if reqId == testJsonRpcId {
			nodeRequestsCounter++
		}

		response := fmt.Sprintf(`{"jsonrpc":"2.0","result": {}, "id": %v}`, string(jsonRpcMessage.ID))
		return []byte(response), http.StatusOK
	}

	pairingList[0] = &lavasession.ConsumerSessionsWithProvider{
		PublicLavaAddress: provider.account.Addr.String(),
		Endpoints: []*lavasession.Endpoint{
			{
				NetworkAddress: provider.endpoint.NetworkAddress.Address,
				Enabled:        true,
				Geolocation:    1,
			},
		},
		Sessions:         map[int64]*lavasession.SingleConsumerSession{},
		MaxComputeUnits:  10000,
		UsedComputeUnits: 0,
		PairingEpoch:     epoch,
	}

	rpcConsumerOptions := rpcConsumerOptions{
		specId:                specId,
		apiInterface:          apiInterface,
		account:               consumerAccount,
		consumerListenAddress: consumerListenAddress,
		epoch:                 epoch,
		pairingList:           pairingList,
		requiredResponses:     1,
		lavaChainID:           lavaChainID,
	}
	rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
	require.NotNil(t, rpcConsumerOut.rpcConsumerServer)

	client := http.Client{}
	sendMessage := func(method string, params []string) http.Header {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","method":"%v","params": [%v], "id":%v}`, method, strings.Join(params, ","), testJsonRpcId)
		resp, err := client.Post("http://"+consumerListenAddress, "application/json", bytes.NewBuffer([]byte(body)))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
		err = json.Unmarshal(bodyBytes, &jsonRpcMessage)
		require.NoError(t, err)

		respId := jsonRpcIdToInt(t, jsonRpcMessage.ID)
		require.Equal(t, testJsonRpcId, respId)
		resp.Body.Close()

		return resp.Header
	}

	mockChainFetcher.SetBlock(1000)

	// Get latest for sanity check
	sendMessage("status", []string{})

	// Get block, this should be cached for next time
	sendMessage("block", []string{"1000"})

	timesSentMessage := 2

	hitCache := waitForCondition(2*time.Second, 100*time.Millisecond, func() bool {
		// Get block, at some point it should be from cache
		sendMessage("block", []string{"1000"})
		timesSentMessage++
		return timesSentMessage > nodeRequestsCounter
	})
	require.True(t, hitCache)

	// Get block again, this time it should be from cache
	sendMessage("block", []string{"1000"})
	timesSentMessage++

	cacheHits := timesSentMessage - nodeRequestsCounter

	// Verify that overall we have 2 cache hits
	require.Equal(t, 2, cacheHits)
}

func TestArchiveProvidersRetryOnParsedHash(t *testing.T) {
	playbook := []struct {
		name             string
		numOfProviders   int
		archiveProviders int
		expectedResult   string
		statusCode       int
	}{
		{
			name:             "happy flow",
			numOfProviders:   2,
			archiveProviders: 1,
			expectedResult:   `{"jsonrpc":"2.0","id":1,"result":{"result":"success"}}`,
			statusCode:       200,
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ctx := context.Background()
			// can be any spec and api interface
			specId := "NEAR"
			apiInterface := spectypes.APIInterfaceJsonRPC
			numberOfRetriesOnNodeErrorsProviderSide = 0
			epoch := uint64(100)
			lavaChainID := "lava"
			numProviders := play.numOfProviders
			cacheListenAddress := addressGen.GetAddress()
			createCacheServer(t, ctx, cacheListenAddress)
			blockHash := "5NFtBbExnjk4TFXpfXhJidcCm5KYPk7QCY51nWiwyQNU"
			consumerListenAddress := addressGen.GetAddress()

			type providerData struct {
				account                sigs.Account
				endpoint               *lavasession.RPCProviderEndpoint
				server                 *rpcprovider.RPCProviderServer
				replySetter            *ReplySetter
				mockChainFetcher       *MockChainFetcher
				mockReliabilityManager *MockReliabilityManager
			}
			providers := []providerData{}
			for i := 0; i < numProviders; i++ {
				account := sigs.GenerateDeterministicFloatingKey(randomizer)
				providerDataI := providerData{account: account}
				providers = append(providers, providerDataI)
			}
			consumerAccount := sigs.GenerateDeterministicFloatingKey(randomizer)
			pairingList := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

			allowArchiveRet := atomic.Bool{}
			allowArchiveRet.Store(false)
			stageTwoCheckFirstTimeArchive := false
			timesCalledProvidersOnSecondStage := 0

			for i := 0; i < numProviders; i++ {
				ctx := context.Background()
				providerDataI := providers[i]
				listenAddress := addressGen.GetAddress()
				handler := func(req []byte, header http.Header) (data []byte, status int) {
					var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
					fmt.Println("regular handler got request", string(req))
					err := json.Unmarshal(req, &jsonRpcMessage)
					require.NoError(t, err)
					if strings.Contains(string(req), blockHash) {
						fmt.Println("hash request, allowing valid archive retry", string(req))
						allowArchiveRet.Store(true)
					}
					if stageTwoCheckFirstTimeArchive {
						timesCalledProvidersOnSecondStage++
					}
					id, _ := json.Marshal(1)
					res := rpcclient.JsonrpcMessage{
						Version: "2.0",
						ID:      id,
						Error:   &rpcclient.JsonError{Code: 1, Message: "test"},
					}
					resBytes, _ := json.Marshal(res)
					return resBytes, 299
				}
				addons := []string(nil)
				extensions := map[string]struct{}{}
				if i >= numProviders-play.archiveProviders {
					extensions = map[string]struct{}{"archive": {}}
					addons = []string{"archive", ""}
					handler = func(req []byte, header http.Header) (data []byte, status int) {
						fmt.Println("archive handler got request", string(req))
						var jsonRpcMessage rpcInterfaceMessages.JsonrpcMessage
						err := json.Unmarshal(req, &jsonRpcMessage)
						require.NoError(t, err)
						if stageTwoCheckFirstTimeArchive {
							timesCalledProvidersOnSecondStage++
						}
						if allowArchiveRet.Load() == true {
							id, _ := json.Marshal(1)
							resultBody, _ := json.Marshal(map[string]string{"result": "success"})
							res := rpcclient.JsonrpcMessage{
								Version: "2.0",
								ID:      id,
								Result:  resultBody,
							}
							resBytes, _ := json.Marshal(res)
							fmt.Println("returning success", string(resBytes))
							return resBytes, http.StatusOK
						}
						id, _ := json.Marshal(1)
						res := rpcclient.JsonrpcMessage{
							Version: "2.0",
							ID:      id,
							Error:   &rpcclient.JsonError{Code: 1, Message: "test"},
						}
						resBytes, _ := json.Marshal(res)
						fmt.Println("returning 299", string(resBytes))
						if strings.Contains(string(req), blockHash) {
							allowArchiveRet.Store(true)
							fmt.Println(allowArchiveRet.Load(), "hash request", string(req))
						}
						return resBytes, 299
					}
				}

				rpcProviderOptions := rpcProviderOptions{
					consumerAddress:  consumerAccount.Addr.String(),
					specId:           specId,
					apiInterface:     apiInterface,
					listenAddress:    listenAddress,
					account:          providerDataI.account,
					lavaChainID:      lavaChainID,
					addons:           addons,
					providerUniqueId: fmt.Sprintf("provider%d", i),
				}
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher, providers[i].mockReliabilityManager = createRpcProvider(t, ctx, rpcProviderOptions)
				providers[i].replySetter.handler = handler

				pairingList[uint64(i)] = &lavasession.ConsumerSessionsWithProvider{
					PublicLavaAddress: providers[i].account.Addr.String(),
					Endpoints: []*lavasession.Endpoint{
						{
							NetworkAddress: providers[i].endpoint.NetworkAddress.Address,
							Enabled:        true,
							Geolocation:    1,
							Extensions:     extensions,
						},
					},
					Sessions:         map[int64]*lavasession.SingleConsumerSession{},
					MaxComputeUnits:  10000,
					UsedComputeUnits: 0,
					PairingEpoch:     epoch,
				}
			}

			rpcConsumerOptions := rpcConsumerOptions{
				specId:                specId,
				apiInterface:          apiInterface,
				account:               consumerAccount,
				consumerListenAddress: consumerListenAddress,
				epoch:                 epoch,
				pairingList:           pairingList,
				requiredResponses:     1,
				lavaChainID:           lavaChainID,
				cacheListenAddress:    cacheListenAddress,
			}
			rpcConsumerOut := createRpcConsumer(t, ctx, rpcConsumerOptions)
			require.NotNil(t, rpcConsumerOut.rpcConsumerServer)
			// wait for consumer to bootstrap
			params, _ := json.Marshal([]string{blockHash})
			id, _ := json.Marshal(1)
			reqBody := rpcclient.JsonrpcMessage{
				Version: "2.0",
				Method:  "block", // Query latest block
				Params:  params,  // Use "final" to get the latest final block
				ID:      id,
			}

			// Convert request to JSON
			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				log.Fatalf("Error marshalling request: %v", err)
			}

			client := http.Client{Timeout: 100000000 * time.Millisecond}
			req, err := http.NewRequest(http.MethodPost, "http://"+consumerListenAddress, bytes.NewBuffer(jsonData))
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, play.statusCode, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			resp.Body.Close()
			require.Equal(t, string(bodyBytes), play.expectedResult)
			fmt.Println("timesCalledProviders", timesCalledProvidersOnSecondStage)

			// Allow relay to hit cache.
			for {
				time.Sleep(time.Millisecond)
				cacheCtx, cancel := context.WithTimeout(ctx, time.Second)
				cacheReply, err := rpcConsumerOut.cache.GetEntry(cacheCtx, &pairingtypes.RelayCacheGet{
					RequestHash:           []byte("test"),
					RequestedBlock:        1005,
					ChainId:               specId,
					SeenBlock:             1005,
					BlocksHashesToHeights: []*pairingtypes.BlockHashToHeight{{Hash: blockHash, Height: spectypes.NOT_APPLICABLE}},
				}) // caching in the consumer doesn't care about hashes, and we don't have data on finalization yet
				cancel()
				if err != nil {
					continue
				}
				if len(cacheReply.BlocksHashesToHeights) > 0 && cacheReply.BlocksHashesToHeights[0].Height != spectypes.NOT_APPLICABLE {
					fmt.Println("cache has this entry", cacheReply.BlocksHashesToHeights)
					break
				} else {
					fmt.Println("cache does not have this entry", cacheReply.BlocksHashesToHeights)
				}
			}
			// attempt 2nd time, this time we should have only one retry
			// set stageTwoCheckFirstTimeArchive to true
			stageTwoCheckFirstTimeArchive = true
			// create new client
			client = http.Client{Timeout: 10000 * time.Millisecond}
			req, err = http.NewRequest(http.MethodPost, "http://"+consumerListenAddress, bytes.NewBuffer(jsonData))
			require.NoError(t, err)

			resp, err = client.Do(req)
			require.NoError(t, err)
			require.Equal(t, play.statusCode, resp.StatusCode)

			bodyBytes, err = io.ReadAll(resp.Body)
			require.NoError(t, err)

			resp.Body.Close()
			require.Equal(t, string(bodyBytes), play.expectedResult)
			require.Equal(t, 1, timesCalledProvidersOnSecondStage) // must go directly to archive as we have it in cache.
			fmt.Println("timesCalledProviders", timesCalledProvidersOnSecondStage)
		})
	}
}
