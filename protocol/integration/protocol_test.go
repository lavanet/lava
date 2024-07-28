package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/provideroptimizer"
	"github.com/lavanet/lava/protocol/rpcconsumer"
	"github.com/lavanet/lava/protocol/rpcprovider"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/connectivity"

	spectypes "github.com/lavanet/lava/x/spec/types"
)

var (
	seed       int64
	randomizer *sigs.ZeroReader
	addressGen uniqueAddressGenerator
)

func TestMain(m *testing.M) {
	// This code will run once before any test cases are executed.
	seed = time.Now().Unix()
	rand.SetSpecificSeed(seed)
	addressGen = uniqueAddressGenerator{}
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

func createRpcConsumer(t *testing.T, ctx context.Context, specId string, apiInterface string, account sigs.Account, consumerListenAddress string, epoch uint64, pairingList map[uint64]*lavasession.ConsumerSessionsWithProvider, requiredResponses int, lavaChainID string) *rpcconsumer.RPCConsumerServer {
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
	})
	chainParser, _, chainFetcher, _, _, err := chainlib.CreateChainLibMocks(ctx, specId, apiInterface, serverHandler, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainFetcher)

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
	baseLatency := common.AverageWorldLatency / 2
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, averageBlockTime, baseLatency, 2)
	consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil, "test")
	consumerSessionManager.UpdateAllProviders(epoch, pairingList)

	consumerConsistency := rpcconsumer.NewConsumerConsistency(specId)
	consumerCmdFlags := common.ConsumerCmdFlags{}
	rpcsonumerLogs, err := metrics.NewRPCConsumerLogs(nil, nil)
	require.NoError(t, err)
	err = rpcConsumerServer.ServeRPCRequests(ctx, rpcEndpoint, consumerStateTracker, chainParser, finalizationConsensus, consumerSessionManager, requiredResponses, account.SK, lavaChainID, nil, rpcsonumerLogs, account.Addr, consumerConsistency, nil, consumerCmdFlags, false, nil, nil, &rpcconsumer.RelayRetriesManager{})
	require.NoError(t, err)
	// wait for consumer server to be up
	consumerUp := checkServerStatusWithTimeout("http://"+consumerListenAddress, time.Millisecond*61)
	require.True(t, consumerUp)

	return rpcConsumerServer
}

func createRpcProvider(t *testing.T, ctx context.Context, consumerAddress string, specId string, apiInterface string, listenAddress string, account sigs.Account, lavaChainID string, addons []string) (*rpcprovider.RPCProviderServer, *lavasession.RPCProviderEndpoint, *ReplySetter, *MockChainFetcher) {
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

	chainParser, chainRouter, chainFetcher, _, endpoint, err := chainlib.CreateChainLibMocks(ctx, specId, apiInterface, serverHandler, "../../", addons)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainFetcher)
	require.NotNil(t, chainRouter)
	endpoint.NetworkAddress.Address = listenAddress

	rpcProviderServer := &rpcprovider.RPCProviderServer{}
	rpcProviderEndpoint := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{
			Address:    endpoint.NetworkAddress.Address,
			KeyPem:     "",
			CertPem:    "",
			DisableTLS: false,
		},
		ChainID:      specId,
		ApiInterface: apiInterface,
		Geolocation:  1,
		NodeUrls: []common.NodeUrl{
			{
				Url:               endpoint.NodeUrls[0].Url,
				InternalPath:      "",
				AuthConfig:        common.AuthConfig{},
				IpForwarding:      false,
				Timeout:           0,
				Addons:            addons,
				SkipVerifications: []string{},
			},
		},
	}
	rewardDB, err := createInMemoryRewardDb([]string{specId})
	require.NoError(t, err)
	_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
	mockProviderStateTracker := mockProviderStateTracker{consumerAddressForPairing: consumerAddress, averageBlockTime: averageBlockTime}
	rws := rewardserver.NewRewardServer(&mockProviderStateTracker, nil, rewardDB, "badger_test", 1, 10, nil)

	blockMemorySize, err := mockProviderStateTracker.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx)
	require.NoError(t, err)
	providerSessionManager := lavasession.NewProviderSessionManager(rpcProviderEndpoint, blockMemorySize)
	providerPolicy := rpcprovider.GetAllAddonsAndExtensionsFromNodeUrlSlice(rpcProviderEndpoint.NodeUrls)
	chainParser.SetPolicy(providerPolicy, specId, apiInterface)

	blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		BlocksToSave:        blocksToSaveChainTracker,
		AverageBlockTime:    averageBlockTime,
		ServerBlockMemory:   rpcprovider.ChainTrackerDefaultMemory + blocksToSaveChainTracker,
		NewLatestCallback:   nil,
		ConsistencyCallback: nil,
		Pmetrics:            nil,
	}

	mockChainFetcher := NewMockChainFetcher(1000, int64(blocksToSaveChainTracker), nil)
	chainTracker, err := chaintracker.NewChainTracker(ctx, mockChainFetcher, chainTrackerConfig)
	require.NoError(t, err)
	reliabilityManager := reliabilitymanager.NewReliabilityManager(chainTracker, &mockProviderStateTracker, account.Addr.String(), chainRouter, chainParser)
	rpcProviderServer.ServeRPCRequests(ctx, rpcProviderEndpoint, chainParser, rws, providerSessionManager, reliabilityManager, account.SK, nil, chainRouter, &mockProviderStateTracker, account.Addr, lavaChainID, rpcprovider.DEFAULT_ALLOWED_MISSING_CU, nil, nil)
	listener := rpcprovider.NewProviderListener(ctx, rpcProviderEndpoint.NetworkAddress, "/health")
	err = listener.RegisterReceiver(rpcProviderServer, rpcProviderEndpoint)
	require.NoError(t, err)
	chainParser.Activate()
	chainTracker.RegisterForBlockTimeUpdates(chainParser)
	providerUp := checkGrpcServerStatusWithTimeout(rpcProviderEndpoint.NetworkAddress.Address, time.Millisecond*261)
	require.True(t, providerUp)
	return rpcProviderServer, endpoint, &replySetter, mockChainFetcher
}

func TestConsumerProviderBasic(t *testing.T) {
	ctx := context.Background()
	// can be any spec and api interface
	specId := "LAV1"
	apiInterface := spectypes.APIInterfaceTendermintRPC
	epoch := uint64(100)
	requiredResponses := 1
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
		providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher = createRpcProvider(t, ctx, consumerAccount.Addr.String(), specId, apiInterface, listenAddress, providerDataI.account, lavaChainID, []string(nil))
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
	rpcconsumerServer := createRpcConsumer(t, ctx, specId, apiInterface, consumerAccount, consumerListenAddress, epoch, pairingList, requiredResponses, lavaChainID)
	require.NotNil(t, rpcconsumerServer)
	client := http.Client{}
	resp, err := client.Get("http://" + consumerListenAddress + "/status")
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
			requiredResponses := 1
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
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher = createRpcProvider(t, ctx, consumerAccount.Addr.String(), specId, apiInterface, listenAddress, providerDataI.account, lavaChainID, []string(nil))
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
			rpcconsumerServer := createRpcConsumer(t, ctx, specId, apiInterface, consumerAccount, consumerListenAddress, epoch, pairingList, requiredResponses, lavaChainID)
			require.NotNil(t, rpcconsumerServer)
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
			requiredResponses := 1
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
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher = createRpcProvider(t, ctx, consumerAccount.Addr.String(), specId, apiInterface, listenAddress, providerDataI.account, lavaChainID, []string(nil))
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
			rpcconsumerServer := createRpcConsumer(t, ctx, specId, apiInterface, consumerAccount, consumerListenAddress, epoch, pairingList, requiredResponses, lavaChainID)
			require.NotNil(t, rpcconsumerServer)

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
			requiredResponses := 1
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
				providers[i].server, providers[i].endpoint, providers[i].replySetter, providers[i].mockChainFetcher = createRpcProvider(t, ctx, consumerAccount.Addr.String(), specId, apiInterface, listenAddress, providerDataI.account, lavaChainID, []string(nil))
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
			rpcconsumerServer := createRpcConsumer(t, ctx, specId, apiInterface, consumerAccount, consumerListenAddress, epoch, pairingList, requiredResponses, lavaChainID)
			require.NotNil(t, rpcconsumerServer)

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
