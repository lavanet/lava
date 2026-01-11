package rpcprovider

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/app"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/performance"

	"github.com/lavanet/lava/v5/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/v5/protocol/statetracker"
	"github.com/lavanet/lava/v5/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v5/protocol/upgrade"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	"github.com/lavanet/lava/v5/utils/memoryutils"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/sigs"
	epochstorage "github.com/lavanet/lava/v5/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ChainTrackerDefaultMemory  = 100
	DEFAULT_ALLOWED_MISSING_CU = 0.2

	ShardIDFlagName           = "shard-id"
	StickinessHeaderName      = "sticky-header"
	DefaultShardID       uint = 0

	CacheMaxCost          = 10 * 1024 // each item cost would be 1
	CacheNumCounters      = 1000      // expect 2000 items
	VerificationsTTL      = 30 * time.Second
	VerificationsCacheKey = "verifications"
)

var (
	Yaml_config_properties     = []string{"network-address.address", "chain-id", "api-interface", "node-urls.url"}
	DefaultRPCProviderFileName = "rpcprovider.yml"

	RelaysHealthEnableFlagDefault  = true
	RelayHealthIntervalFlagDefault = 5 * time.Minute
)

// used to call SetPolicy in base chain parser so we are allowed to run verifications on the addons and extensions
type ProviderPolicy struct {
	addons     []string
	extensions []epochstorage.EndpointService
}

func (pp *ProviderPolicy) GetSupportedAddons(specID string) (addons []string, err error) {
	return pp.addons, nil
}

func (pp *ProviderPolicy) GetSupportedExtensions(specID string) (extensions []epochstorage.EndpointService, err error) {
	return pp.extensions, nil
}

type ProviderStateTrackerInf interface {
	RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf)
	RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error
	RegisterForSpecVerifications(ctx context.Context, specVerifier updaters.SpecVerifier, chainId string) error
	RegisterForEpochUpdates(ctx context.Context, epochUpdatable updaters.EpochUpdatable)
	RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error
	LatestBlock() int64
	GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epocu uint64) (maxCu uint64, err error)
	VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error)
	GetEpochSize(ctx context.Context) (uint64, error)
	EarliestBlockInMemory(ctx context.Context) (uint64, error)
	RegisterPaymentUpdatableForPayments(ctx context.Context, paymentUpdatable updaters.PaymentUpdatable)
	GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
	GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
	GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error)
	GetVirtualEpoch(epoch uint64) uint64
	GetAverageBlockTime() time.Duration
}

type rpcProviderStartOptions struct {
	ctx                       context.Context
	txFactory                 tx.Factory
	clientCtx                 client.Context
	rpcProviderEndpoints      []*lavasession.RPCProviderEndpoint
	cache                     *performance.Cache
	parallelConnections       uint
	metricsListenAddress      string
	rewardStoragePath         string
	rewardTTL                 time.Duration
	shardID                   uint
	rewardsSnapshotThreshold  uint
	rewardsSnapshotTimeoutSec uint
	healthCheckMetricsOptions *rpcProviderHealthCheckMetricsOptions
	staticProvider            bool
	staticSpecPath            string
	githubToken               string
	relayLoadLimit            uint64
	testMode                  bool
	testResponsesFile         string
	epochDuration             time.Duration
	resourceLimiterOptions    *resourceLimiterOptions
	memoryGCThresholdGB       float64
}

type resourceLimiterOptions struct {
	enabled             bool
	cuThreshold         uint64
	heavyMaxConcurrent  int64
	heavyQueueSize      int
	normalMaxConcurrent int64
}

type rpcProviderHealthCheckMetricsOptions struct {
	relaysHealthEnableFlag   bool          // enables relay health check
	relaysHealthIntervalFlag time.Duration // interval for relay health check
	grpcHealthCheckEndpoint  string
}

type RPCProvider struct {
	providerStateTracker ProviderStateTrackerInf
	rpcProviderListeners map[string]*ProviderListener
	lock                 sync.RWMutex
	// all of the following members need to be concurrency proof
	providerMetricsManager       *metrics.ProviderMetricsManager
	rewardServer                 *rewardserver.RewardServer
	privKey                      *btcSecp256k1.PrivateKey
	lavaChainID                  string
	addr                         sdk.AccAddress
	blockMemorySize              uint64
	chainMutexes                 map[string]*sync.Mutex
	parallelConnections          uint
	cache                        *performance.Cache
	cacheLatestBlockEnabled      bool
	shardID                      uint // shardID is a flag that allows setting up multiple provider databases of the same chain
	chainTrackers                *common.SafeSyncMap[string, chaintracker.IChainTracker]
	relaysMonitorAggregator      *metrics.RelaysMonitorAggregator
	relaysHealthCheckEnabled     bool
	relaysHealthCheckInterval    time.Duration
	grpcHealthCheckEndpoint      string
	providerUniqueId             string
	epochTimer                   *common.EpochTimer
	sessionManagers              map[string]*lavasession.ProviderSessionManager // key: chainID-apiInterface
	sessionManagersLock          sync.RWMutex                                   // protects sessionManagers map
	staticProvider               bool
	staticSpecPath               string
	githubToken                  string
	relayLoadLimit               uint64
	providerLoadManagersPerChain *common.SafeSyncMap[string, *ProviderLoadManager]

	verificationsResponseCache                          *ristretto.Cache[string, []*pairingtypes.Verification]
	allChainsAndAPIInterfacesVerificationStatusFetchers []IVerificationsStatus
	testMode                                            bool
	testResponsesFile                                   string
	resourceLimiterOptions                              resourceLimiterOptions
}

func (rpcp *RPCProvider) AddVerificationStatusFetcher(fetcher IVerificationsStatus) {
	rpcp.lock.Lock()
	defer rpcp.lock.Unlock()
	if rpcp.allChainsAndAPIInterfacesVerificationStatusFetchers == nil {
		rpcp.allChainsAndAPIInterfacesVerificationStatusFetchers = []IVerificationsStatus{}
	}
	rpcp.allChainsAndAPIInterfacesVerificationStatusFetchers = append(rpcp.allChainsAndAPIInterfacesVerificationStatusFetchers, fetcher)
	rpcp.verificationsResponseCache.Clear()
}

func (rpcp *RPCProvider) Start(options *rpcProviderStartOptions) (err error) {
	ctx, cancel := context.WithCancel(options.ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	rpcp.providerUniqueId = strconv.FormatUint(utils.GenerateUniqueIdentifier(), 10)
	rpcp.chainTrackers = &common.SafeSyncMap[string, chaintracker.IChainTracker]{}
	rpcp.parallelConnections = options.parallelConnections
	rpcp.cache = options.cache
	rpcp.providerMetricsManager = metrics.NewProviderMetricsManager(options.metricsListenAddress) // start up prometheus metrics
	rpcp.providerMetricsManager.SetVersion(upgrade.GetCurrentVersion().ProviderVersion)
	rpcp.rpcProviderListeners = make(map[string]*ProviderListener)
	rpcp.shardID = options.shardID
	rpcp.relaysHealthCheckEnabled = options.healthCheckMetricsOptions.relaysHealthEnableFlag
	rpcp.relaysHealthCheckInterval = options.healthCheckMetricsOptions.relaysHealthIntervalFlag
	rpcp.relaysMonitorAggregator = metrics.NewRelaysMonitorAggregator(rpcp.relaysHealthCheckInterval, rpcp.providerMetricsManager)
	rpcp.grpcHealthCheckEndpoint = options.healthCheckMetricsOptions.grpcHealthCheckEndpoint
	rpcp.staticProvider = options.staticProvider
	rpcp.staticSpecPath = options.staticSpecPath
	rpcp.githubToken = options.githubToken
	rpcp.relayLoadLimit = options.relayLoadLimit
	rpcp.providerLoadManagersPerChain = &common.SafeSyncMap[string, *ProviderLoadManager]{}
	if options.resourceLimiterOptions != nil {
		rpcp.resourceLimiterOptions = *options.resourceLimiterOptions
	}

	// Initialize session managers map for epoch timer callbacks
	rpcp.sessionManagers = make(map[string]*lavasession.ProviderSessionManager)

	// Create time-based epoch timer for static providers (standalone mode)
	if options.staticProvider {
		// Validate that static providers have a spec path configured
		// Standalone mode cannot fetch specs from the blockchain, so specs must be provided
		if options.staticSpecPath == "" {
			return utils.LavaFormatError(
				"--static-spec-path is required when using --static-providers",
				nil,
				utils.LogAttr("static-providers", true),
				utils.LogAttr("static-spec-path", "empty"),
			)
		}

		// Static providers ALWAYS run with epoch timer
		epochDuration := options.epochDuration
		if epochDuration == 0 {
			epochDuration = common.StandaloneEpochDuration // 15 minutes default for standalone
		}

		rpcp.epochTimer = common.NewEpochTimer(epochDuration)
		currentEpoch := rpcp.epochTimer.GetCurrentEpoch()
		timeUntilNext := rpcp.epochTimer.GetTimeUntilNextEpoch()

		utils.LavaFormatInfo("Static provider: using time-based epochs (standalone mode)",
			utils.LogAttr("epochDuration", epochDuration),
			utils.LogAttr("currentEpoch", currentEpoch),
			utils.LogAttr("timeUntilNextEpoch", timeUntilNext),
			utils.LogAttr("nextEpochTime", time.Now().Add(timeUntilNext).Format("15:04:05 MST")),
		)
	}

	// single state tracker
	var providerStateTracker ProviderStateTrackerInf
	if !options.staticProvider {
		// Regular provider: connect to Lava blockchain
		lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, options.clientCtx)
		pst, err := statetracker.NewProviderStateTracker(ctx, options.txFactory, options.clientCtx, lavaChainFetcher, rpcp.providerMetricsManager)
		if err != nil {
			return err
		}
		providerStateTracker = pst
	} else {
		// Static provider: use standalone state tracker (no blockchain connection)
		utils.LavaFormatInfo("Static provider mode: using standalone state tracker (no Lava blockchain connection)")
		// Pass 0 to use default LAV1 block time (15s)
		// This is for the Lava blockchain itself, not for serviced chains
		providerStateTracker = statetracker.NewStandaloneStateTracker(rpcp.epochTimer, 0)
	}

	rpcp.providerStateTracker = providerStateTracker

	// Register metrics updater (only for non-standalone, as RegisterForUpdates is not in the interface)
	if !options.staticProvider {
		// Type assert to access RegisterForUpdates method
		if pst, ok := providerStateTracker.(*statetracker.ProviderStateTracker); ok {
			pst.RegisterForUpdates(ctx, updaters.NewMetricsUpdater(rpcp.providerMetricsManager))
		}
	}

	// check version
	version, err := rpcp.providerStateTracker.GetProtocolVersion(ctx)
	if err != nil {
		utils.LavaFormatFatal("failed fetching protocol version from node", err)
	}
	rpcp.providerStateTracker.RegisterForVersionUpdates(ctx, version.Version, &upgrade.ProtocolVersion{})

	// Start memory GC monitoring goroutine
	memoryutils.StartMemoryGC(ctx, options.memoryGCThresholdGB)

	// single reward server
	if !options.staticProvider {
		rewardDB := rewardserver.NewRewardDBWithTTL(options.rewardTTL)
		rpcp.rewardServer = rewardserver.NewRewardServer(providerStateTracker, rpcp.providerMetricsManager, rewardDB, options.rewardStoragePath, options.rewardsSnapshotThreshold, options.rewardsSnapshotTimeoutSec, rpcp)
		rpcp.providerStateTracker.RegisterForEpochUpdates(ctx, rpcp.rewardServer)
		rpcp.providerStateTracker.RegisterPaymentUpdatableForPayments(ctx, rpcp.rewardServer)
	}

	// Get private key and address - only needed for regular providers
	if !options.staticProvider {
		// Regular provider: Load key from keyring
		keyName, err := sigs.GetKeyName(options.clientCtx)
		if err != nil {
			utils.LavaFormatFatal("failed getting key name from clientCtx", err)
		}
		privKey, err := sigs.GetPrivKey(options.clientCtx, keyName)
		if err != nil {
			utils.LavaFormatFatal("failed getting private key from key name", err, utils.Attribute{Key: "keyName", Value: keyName})
		}
		rpcp.privKey = privKey
		clientKey, _ := options.clientCtx.Keyring.Key(keyName)
		rpcp.lavaChainID = options.clientCtx.ChainID

		pubKey, err := clientKey.GetPubKey()
		if err != nil {
			return err
		}

		err = rpcp.addr.Unmarshal(pubKey.Address())
		if err != nil {
			utils.LavaFormatFatal("failed unmarshaling public address", err, utils.Attribute{Key: "keyName", Value: keyName}, utils.Attribute{Key: "pubkey", Value: pubKey.Address()})
		}

		utils.LavaFormatInfo("RPCProvider pubkey: " + rpcp.addr.String())
	} else {
		// Static provider: Generate ephemeral key (not used for signing transactions)
		utils.LavaFormatInfo("Static provider mode: generating ephemeral key for address identification")

		// Generate a new secp256k1 private key
		cosmosPrivKey := secp256k1.GenPrivKey()

		// Convert cosmos SDK key to btcec format
		privKey, _ := btcSecp256k1.PrivKeyFromBytes(cosmosPrivKey.Bytes())
		rpcp.privKey = privKey

		// Derive address from the ephemeral key
		pubKey := cosmosPrivKey.PubKey()
		err := rpcp.addr.Unmarshal(pubKey.Address())
		if err != nil {
			utils.LavaFormatFatal("failed unmarshaling ephemeral public address", err)
		}

		// Use a generic chain ID for static providers
		rpcp.lavaChainID = "standalone"

		utils.LavaFormatInfo("RPCProvider using ephemeral address: " + rpcp.addr.String())
	}

	rpcp.createAndRegisterFreezeUpdatersByOptions(ctx, providerStateTracker, rpcp.addr.String(), options.staticProvider)

	utils.LavaFormatInfo("RPCProvider setting up endpoints", utils.Attribute{Key: "count", Value: strconv.Itoa(len(options.rpcProviderEndpoints))})

	blockMemorySize, err := rpcp.providerStateTracker.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx) // get the number of blocks to keep in PSM.
	if err != nil {
		utils.LavaFormatFatal("Failed fetching GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment in RPCProvider Start", err)
	}
	rpcp.blockMemorySize = blockMemorySize
	// pre loop to handle synchronous actions
	rpcp.chainMutexes = map[string]*sync.Mutex{}
	for idx, endpoint := range options.rpcProviderEndpoints {
		rpcp.chainMutexes[endpoint.ChainID] = &sync.Mutex{}   // create a mutex per chain for shared resources
		if idx > 0 && endpoint.NetworkAddress.Address == "" { // handle undefined addresses as the previous endpoint for shared listeners
			endpoint.NetworkAddress = options.rpcProviderEndpoints[idx-1].NetworkAddress
		}
	}

	specValidator := NewSpecValidator()
	utils.LavaFormatInfo("Running setup for RPCProvider endpoints", utils.LogAttr("endpoints", options.rpcProviderEndpoints))
	disabledEndpointsList := rpcp.SetupProviderEndpoints(options.rpcProviderEndpoints, specValidator, true)
	rpcp.relaysMonitorAggregator.StartMonitoring(ctx)
	specValidator.Start(ctx)

	// Start epoch timer after all endpoints are set up (only for static providers)
	if rpcp.staticProvider && rpcp.epochTimer != nil {
		// Register all session managers for epoch updates
		rpcp.sessionManagersLock.RLock()
		sessionManagersCopy := make(map[string]*lavasession.ProviderSessionManager, len(rpcp.sessionManagers))
		for k, v := range rpcp.sessionManagers {
			sessionManagersCopy[k] = v
		}
		rpcp.sessionManagersLock.RUnlock()

		for chainKey, sm := range sessionManagersCopy {
			sessionManager := sm // Capture for closure
			chainKeyLog := chainKey

			rpcp.epochTimer.RegisterCallback(func(epoch uint64) {
				utils.LavaFormatInfo("ProviderSessionManager: Epoch update triggered",
					utils.LogAttr("epoch", epoch),
					utils.LogAttr("chainKey", chainKeyLog),
					utils.LogAttr("time", time.Now().Format("15:04:05 MST")),
				)

				// Update session manager to trigger cleanup
				sessionManager.UpdateEpoch(epoch)
			})

			utils.LavaFormatInfo("RPCProvider: Registered session manager for epoch updates.",
				utils.LogAttr("chainKey", chainKey),
			)
		}

		// Start the epoch timer
		rpcp.epochTimer.Start(ctx)
	}

	utils.LavaFormatInfo("RPCProvider done setting up endpoints, ready for service")
	if len(disabledEndpointsList) > 0 {
		utils.LavaFormatError(utils.FormatStringerList("[-] RPCProvider running with disabled endpoints:", disabledEndpointsList, "[-]"), nil)
		if len(disabledEndpointsList) == len(options.rpcProviderEndpoints) {
			utils.LavaFormatFatal("all endpoints are disabled", nil)
		}
		activeEndpointsList := getActiveEndpoints(options.rpcProviderEndpoints, disabledEndpointsList)
		utils.LavaFormatInfo(utils.FormatStringerList("[+] active endpoints:", activeEndpointsList, "[+]"))
		// try to save disabled endpoints
		go rpcp.RetryDisabledEndpoints(disabledEndpointsList, specValidator, 1)
	} else {
		utils.LavaFormatInfo("[+] all endpoints up and running")
	}
	// tearing down
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("Provider Server ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("Provider Server signalChan")
	}

	for _, listener := range rpcp.rpcProviderListeners {
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		listener.Shutdown(shutdownCtx)
		defer shutdownRelease()
	}

	// close all reward dbs
	err = rpcp.rewardServer.CloseAllDataBases()
	if err != nil {
		utils.LavaFormatError("failed to close reward db", err)
	}

	return nil
}

func (rpcp *RPCProvider) createAndRegisterFreezeUpdatersByOptions(ctx context.Context, providerStateTracker ProviderStateTrackerInf, publicAddress string, staticProvider bool) {
	if staticProvider {
		// Static providers don't need freeze/jail monitoring
		utils.LavaFormatDebug("Static provider mode: skipping freeze/jail updater registration")
		return
	}

	// Regular providers: monitor freeze/jail status on blockchain
	// Access StateQuery from ProviderStateTracker (only available for non-standalone tracker)
	type stateQueryAccessor interface {
		GetStateQuery() *updaters.StateQuery
	}

	if pst, ok := providerStateTracker.(stateQueryAccessor); ok {
		stateQuery := pst.GetStateQuery()
		if stateQuery != nil {
			freezeJailUpdater := updaters.NewProviderFreezeJailUpdater(stateQuery, publicAddress, rpcp.providerMetricsManager)
			rpcp.providerStateTracker.RegisterForEpochUpdates(ctx, freezeJailUpdater)
		}
	}
}

func getActiveEndpoints(rpcProviderEndpoints []*lavasession.RPCProviderEndpoint, disabledEndpointsList []*lavasession.RPCProviderEndpoint) []*lavasession.RPCProviderEndpoint {
	activeEndpoints := map[*lavasession.RPCProviderEndpoint]struct{}{}
	for _, endpoint := range rpcProviderEndpoints {
		activeEndpoints[endpoint] = struct{}{}
	}
	for _, disabled := range disabledEndpointsList {
		delete(activeEndpoints, disabled)
	}
	activeEndpointsList := []*lavasession.RPCProviderEndpoint{}
	for endpoint := range activeEndpoints {
		activeEndpointsList = append(activeEndpointsList, endpoint)
	}
	return activeEndpointsList
}

func (rpcp *RPCProvider) RetryDisabledEndpoints(disabledEndpoints []*lavasession.RPCProviderEndpoint, specValidator *SpecValidator, retryCount int) {
	time.Sleep(time.Duration(retryCount) * time.Second)
	parallel := retryCount > 2
	utils.LavaFormatInfo("Retrying disabled endpoints", utils.Attribute{Key: "disabled endpoints list", Value: disabledEndpoints}, utils.Attribute{Key: "parallel", Value: parallel})
	disabledEndpointsAfterRetry := rpcp.SetupProviderEndpoints(disabledEndpoints, specValidator, parallel)
	if len(disabledEndpointsAfterRetry) > 0 {
		utils.LavaFormatError(utils.FormatStringerList("RPCProvider running with disabled endpoints:", disabledEndpointsAfterRetry, "[-]"), nil)
		rpcp.RetryDisabledEndpoints(disabledEndpointsAfterRetry, specValidator, retryCount+1)
		activeEndpointsList := getActiveEndpoints(disabledEndpoints, disabledEndpointsAfterRetry)
		utils.LavaFormatInfo(utils.FormatStringerList("[+] active endpoints:", activeEndpointsList, "[+]"))
	} else {
		utils.LavaFormatInfo("[+] all endpoints up and running")
	}
}

func (rpcp *RPCProvider) SetupProviderEndpoints(rpcProviderEndpoints []*lavasession.RPCProviderEndpoint, specValidator *SpecValidator, parallel bool) (disabledEndpointsRet []*lavasession.RPCProviderEndpoint) {
	var wg sync.WaitGroup
	parallelJobs := len(rpcProviderEndpoints)
	wg.Add(parallelJobs)
	disabledEndpoints := make(chan *lavasession.RPCProviderEndpoint, parallelJobs)
	// validate static spec configuration is used only on a single chain setup.
	for _, rpcProviderEndpoint := range rpcProviderEndpoints {
		setupEndpoint := func(rpcProviderEndpoint *lavasession.RPCProviderEndpoint, specValidator *SpecValidator) {
			defer wg.Done()
			err := rpcp.SetupEndpoint(context.Background(), rpcProviderEndpoint, specValidator)
			if err != nil {
				rpcp.providerMetricsManager.SetDisabledChain(rpcProviderEndpoint.ChainID, rpcProviderEndpoint.ApiInterface)
				disabledEndpoints <- rpcProviderEndpoint
			}
		}
		if parallel {
			go setupEndpoint(rpcProviderEndpoint, specValidator)
		} else {
			setupEndpoint(rpcProviderEndpoint, specValidator)
		}
	}
	wg.Wait()
	close(disabledEndpoints)
	disabledEndpointsList := []*lavasession.RPCProviderEndpoint{}
	for disabledEndpoint := range disabledEndpoints {
		disabledEndpointsList = append(disabledEndpointsList, disabledEndpoint)
	}
	return disabledEndpointsList
}

func GetAllAddonsAndExtensionsFromNodeUrlSlice(nodeUrls []common.NodeUrl) *ProviderPolicy {
	policy := &ProviderPolicy{}
	for _, nodeUrl := range nodeUrls {
		policy.addons = append(policy.addons, nodeUrl.Addons...) // addons are added without validation while extensions are. so we add to the addons all.
	}
	return policy
}

func GetAllNodeUrlsInternalPaths(nodeUrls []common.NodeUrl) []string {
	paths := []string{}
	for _, nodeUrl := range nodeUrls {
		paths = append(paths, nodeUrl.InternalPath)
	}
	return paths
}

func (rpcp *RPCProvider) GetVerificationsStatus() []*pairingtypes.Verification {
	verifications, ok := rpcp.verificationsResponseCache.Get(VerificationsCacheKey)
	if ok {
		return verifications
	}
	rpcp.lock.RLock()
	defer rpcp.lock.RUnlock()
	verifications = make([]*pairingtypes.Verification, 0)
	for _, fetcher := range rpcp.allChainsAndAPIInterfacesVerificationStatusFetchers {
		verifications = append(verifications, fetcher.GetVerificationsStatus()...)
	}
	rpcp.verificationsResponseCache.SetWithTTL(VerificationsCacheKey, verifications, 1, VerificationsTTL)
	return verifications
}

func (rpcp *RPCProvider) SetupEndpoint(ctx context.Context, rpcProviderEndpoint *lavasession.RPCProviderEndpoint, specValidator *SpecValidator) error {
	err := rpcProviderEndpoint.Validate()
	if err != nil {
		return utils.LavaFormatError("[PANIC] panic severity critical error, aborting support for chain api due to invalid node url definition, continuing with others", err, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()})
	}

	chainID := rpcProviderEndpoint.ChainID
	apiInterface := rpcProviderEndpoint.ApiInterface
	providerSessionManager := lavasession.NewProviderSessionManager(rpcProviderEndpoint, rpcp.blockMemorySize)

	// Store session manager for epoch timer callbacks
	sessionManagerKey := chainID + "-" + apiInterface
	rpcp.sessionManagersLock.Lock()
	rpcp.sessionManagers[sessionManagerKey] = providerSessionManager
	rpcp.sessionManagersLock.Unlock()

	rpcp.providerStateTracker.RegisterForEpochUpdates(ctx, providerSessionManager)
	chainParser, err := chainlib.NewChainParser(apiInterface)
	if err != nil {
		return utils.LavaFormatError("[PANIC] panic severity critical error, aborting support for chain api due to invalid chain parser, continuing with others", err, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()})
	}

	rpcEndpoint := lavasession.RPCEndpoint{ChainID: chainID, ApiInterface: apiInterface}
	err = statetracker.RegisterForSpecUpdatesOrSetStaticSpecWithToken(ctx, chainParser, rpcp.staticSpecPath, rpcEndpoint, rpcp.providerStateTracker, rpcp.githubToken)
	if err != nil {
		return utils.LavaFormatError("[PANIC] failed to RegisterForSpecUpdates, panic severity critical error, aborting support for chain api due to invalid chain parser, continuing with others", err, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()})
	}

	// warn if not all internal paths are configured
	configuredInternalPaths := GetAllNodeUrlsInternalPaths(rpcProviderEndpoint.NodeUrls)
	chainInternalPaths := chainParser.GetAllInternalPaths()
	overConfiguredInternalPaths := lavaslices.MissingElements(configuredInternalPaths, chainInternalPaths)
	if len(overConfiguredInternalPaths) > 0 {
		utils.LavaFormatWarning("Some configured internal paths are not in the chain's spec", nil,
			utils.LogAttr("chainID", chainID),
			utils.LogAttr("apiInterface", apiInterface),
			utils.LogAttr("internalPaths", strings.Join(overConfiguredInternalPaths, ",")),
		)
	}

	underConfiguredInternalPaths := lavaslices.MissingElements(chainInternalPaths, configuredInternalPaths)
	if len(underConfiguredInternalPaths) > 0 {
		utils.LavaFormatWarning("Some internal paths from the spec are not configured for this provider", nil,
			utils.LogAttr("chainID", chainID),
			utils.LogAttr("apiInterface", apiInterface),
			utils.LogAttr("internalPaths", strings.Join(underConfiguredInternalPaths, ",")),
		)

		addMissingInternalPaths(rpcProviderEndpoint, chainParser, chainID, apiInterface, underConfiguredInternalPaths)
	}

	// after registering for spec updates our chain parser contains the spec and we can add our addons and extensions to allow our provider to function properly
	providerPolicy := GetAllAddonsAndExtensionsFromNodeUrlSlice(rpcProviderEndpoint.NodeUrls)
	utils.LavaFormatInfo("supported services for provider",
		utils.LogAttr("specId", rpcProviderEndpoint.ChainID),
		utils.LogAttr("apiInterface", apiInterface),
		utils.LogAttr("supportedServices", providerPolicy.addons),
		utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
	chainParser.SetPolicy(providerPolicy, rpcProviderEndpoint.ChainID, apiInterface)
	chainRouter, err := chainlib.GetChainRouter(ctx, rpcp.parallelConnections, rpcProviderEndpoint, chainParser)
	if err != nil {
		return utils.LavaFormatError("[PANIC] panic severity critical error, failed creating chain proxy, continuing with others endpoints", err, utils.Attribute{Key: "parallelConnections", Value: uint64(rpcp.parallelConnections)}, utils.Attribute{Key: "rpcProviderEndpoint", Value: rpcProviderEndpoint})
	}

	_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
	var chainTracker chaintracker.IChainTracker
	// chainTracker accepts a callback to be called on new blocks, we use this to call metrics update on a new block
	recordMetricsOnNewBlock := func(blockFrom int64, blockTo int64, hash string) {
		utils.LavaFormatDebug("ChainTracker callback triggered - new blocks detected",
			utils.LogAttr("chainID", chainID),
			utils.LogAttr("blockFrom", blockFrom),
			utils.LogAttr("blockTo", blockTo),
			utils.LogAttr("hash", hash),
			utils.LogAttr("provider_endpoint", rpcProviderEndpoint.NetworkAddress.Address),
			utils.LogAttr("num_blocks", blockTo-blockFrom),
		)
		for block := blockFrom + 1; block <= blockTo; block++ {
			utils.LavaFormatDebug("ChainTracker calling SetLatestBlock for block",
				utils.LogAttr("chainID", chainID),
				utils.LogAttr("block", block),
				utils.LogAttr("provider_endpoint", rpcProviderEndpoint.NetworkAddress.Address),
			)
			rpcp.providerMetricsManager.SetLatestBlock(chainID, rpcProviderEndpoint.NetworkAddress.Address, uint64(block))
		}
		utils.LavaFormatDebug("ChainTracker callback completed",
			utils.LogAttr("chainID", chainID),
			utils.LogAttr("blockFrom", blockFrom),
			utils.LogAttr("blockTo", blockTo),
			utils.LogAttr("total_blocks_processed", blockTo-blockFrom),
		)
	}
	utils.LavaFormatInfo("Setting up ChainFetcher for spec", utils.LogAttr("chainId", rpcEndpoint.ChainID))
	var chainFetcher chainlib.IChainFetcher = chainlib.NewChainFetcher(
		ctx,
		&chainlib.ChainFetcherOptions{
			ChainRouter: chainRouter,
			ChainParser: chainParser,
			Endpoint:    rpcProviderEndpoint,
			Cache:       rpcp.cache,
		},
	)
	// so we can fetch failed verifications we need to add the chainFetcher before returning
	rpcp.AddVerificationStatusFetcher(chainFetcher)

	// check the chain fetcher verification works, if it doesn't we disable the chain+apiInterface and this triggers a boot retry
	utils.LavaFormatInfo("validating ChainFetcher for spec",
		utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name},
		utils.Attribute{Key: "Chain", Value: rpcProviderEndpoint.ChainID})
	err = chainFetcher.Validate(ctx)
	if err != nil {
		return utils.LavaFormatError("[PANIC] Failed starting due to chain fetcher validation failure", err,
			utils.Attribute{Key: "Chain", Value: rpcProviderEndpoint.ChainID},
			utils.Attribute{Key: "apiInterface", Value: apiInterface},
			utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name},
		)
	}

	// in order to utilize shared resources between chains we need go routines with the same chain to wait for one another here
	var loadManager *ProviderLoadManager
	chainCommonSetup := func() error {
		rpcp.chainMutexes[chainID].Lock()
		defer rpcp.chainMutexes[chainID].Unlock()

		consistencyErrorCallback := func(oldBlock, newBlock int64) {
			utils.LavaFormatError("Consistency issue detected", nil,
				utils.Attribute{Key: "oldBlock", Value: oldBlock},
				utils.Attribute{Key: "newBlock", Value: newBlock},
				utils.Attribute{Key: "Chain", Value: rpcProviderEndpoint.ChainID},
				utils.Attribute{Key: "apiInterface", Value: apiInterface},
				utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name},
			)
		}
		blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)
		chainTrackerConfig := chaintracker.ChainTrackerConfig{
			BlocksToSave:          blocksToSaveChainTracker,
			AverageBlockTime:      averageBlockTime,
			ServerBlockMemory:     ChainTrackerDefaultMemory + blocksToSaveChainTracker,
			NewLatestCallback:     recordMetricsOnNewBlock,
			ConsistencyCallback:   consistencyErrorCallback,
			Pmetrics:              rpcp.providerMetricsManager,
			ChainId:               chainID,
			ParseDirectiveEnabled: chainParser.ParseDirectiveEnabled(),
		}

		chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
		if err != nil {
			return utils.LavaFormatError("panic severity critical error, aborting support for chain api due to node access, continuing with other endpoints", err, utils.Attribute{Key: "chainTrackerConfig", Value: chainTrackerConfig}, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint}, utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
		}

		if !chainTracker.IsDummy() {
			chainTrackerLoaded, loaded, err := rpcp.chainTrackers.LoadOrStore(chainID, chainTracker)
			if err != nil {
				utils.LavaFormatFatal("failed to load or store chain tracker", err, utils.LogAttr("chainID", chainID), utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
			}

			if !loaded { // this is the first time we are setting up the chain tracker, we need to register for spec verifications
				chainTracker.StartAndServe(ctx)
				utils.LavaFormatInfo("Registering for spec verifications for endpoint", utils.LogAttr("rpcEndpoint", rpcEndpoint), utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
				// we register for spec verifications only once, and this triggers all chainFetchers of that specId when it triggers
				err = rpcp.providerStateTracker.RegisterForSpecVerifications(ctx, specValidator, rpcEndpoint.ChainID)
				if err != nil {
					return utils.LavaFormatError("failed to RegisterForSpecUpdates, panic severity critical error, aborting support for chain api due to invalid chain parser, continuing with others", err, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()}, utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
				}
			} else { // loaded an existing chain tracker. use the same one instead
				chainTracker = chainTrackerLoaded
				utils.LavaFormatInfo("reusing chain tracker", utils.Attribute{Key: "chain", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
			}
		}

		// create provider load manager per chain ID
		loadManager, _, err = rpcp.providerLoadManagersPerChain.LoadOrStore(rpcProviderEndpoint.ChainID, NewProviderLoadManager(rpcp.relayLoadLimit))
		if err != nil {
			utils.LavaFormatError("Failed LoadOrStore providerLoadManagersPerChain", err, utils.LogAttr("chainId", rpcProviderEndpoint.ChainID), utils.LogAttr("rpcp.relayLoadLimit", rpcp.relayLoadLimit))
		}
		return nil
	}
	err = chainCommonSetup()
	if err != nil {
		utils.LavaFormatError("failed to run chain common setup", err, utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
		return err
	}

	// Add the chain fetcher to the spec validator
	// check the chain fetcher verification works, if it doesn't we disable the chain+apiInterface and this triggers a boot retry
	if chainParser.ParseDirectiveEnabled() {
		err = specValidator.AddChainFetcher(ctx, &chainFetcher, chainID)
		if err != nil {
			return utils.LavaFormatError("panic severity critical error, failed validating chain", err, utils.Attribute{Key: "rpcProviderEndpoint", Value: rpcProviderEndpoint})
		}
	}
	providerMetrics := rpcp.providerMetricsManager.AddProviderMetrics(chainID, apiInterface, rpcProviderEndpoint.NetworkAddress.Address)

	// add a database for this chainID if does not exist.
	rpcp.rewardServer.AddDataBase(rpcProviderEndpoint.ChainID, rpcp.addr.String(), rpcp.shardID)

	var relaysMonitor *metrics.RelaysMonitor = nil
	if rpcp.relaysHealthCheckEnabled {
		relaysMonitor = metrics.NewRelaysMonitor(rpcp.relaysHealthCheckInterval, chainID, apiInterface)
		rpcp.relaysMonitorAggregator.RegisterRelaysMonitor(rpcEndpoint.Key(), relaysMonitor)
		rpcp.providerMetricsManager.RegisterRelaysMonitor(chainID, apiInterface, relaysMonitor)
	}

	rpcProviderServer := &RPCProviderServer{providerUniqueId: rpcp.providerUniqueId}

	var providerNodeSubscriptionManager *chainlib.ProviderNodeSubscriptionManager
	if rpcProviderEndpoint.ApiInterface == spectypes.APIInterfaceTendermintRPC || rpcProviderEndpoint.ApiInterface == spectypes.APIInterfaceJsonRPC {
		utils.LavaFormatInfo("Creating provider node subscription manager", utils.LogAttr("rpcProviderEndpoint", rpcProviderEndpoint))
		providerNodeSubscriptionManager = chainlib.NewProviderNodeSubscriptionManager(chainRouter, chainParser, rpcp.privKey)
	}

	// Create test mode config if enabled
	var testModeConfig *TestModeConfig
	if rpcp.testMode {
		// Load test responses from file
		err = rpcProviderServer.loadTestModeConfig(rpcp.testMode, rpcp.testResponsesFile)
		if err != nil {
			return utils.LavaFormatError("failed to load test mode configuration", err)
		}
		testModeConfig = rpcProviderServer.testModeConfig
	}

	// Create resource limiter if enabled
	rlOptions := rpcp.resourceLimiterOptions
	enableResourceLimiter := rlOptions.enabled
	cuThreshold := rlOptions.cuThreshold
	heavyMaxConcurrent := rlOptions.heavyMaxConcurrent
	heavyQueueSize := rlOptions.heavyQueueSize
	normalMaxConcurrent := rlOptions.normalMaxConcurrent

	// Validate and apply fallback for CU threshold
	const DefaultCUThreshold = 100
	const MinCUThreshold = 50
	if cuThreshold == 0 {
		cuThreshold = DefaultCUThreshold
		if enableResourceLimiter {
			utils.LavaFormatWarning("resource-limiter-cu-threshold not set or is 0, using default",
				nil,
				utils.LogAttr("default_cu_threshold", DefaultCUThreshold),
			)
		}
	} else if cuThreshold < MinCUThreshold {
		utils.LavaFormatWarning("resource-limiter-cu-threshold is too low, using minimum",
			nil,
			utils.LogAttr("provided", cuThreshold),
			utils.LogAttr("minimum", MinCUThreshold),
		)
		cuThreshold = MinCUThreshold
	}

	// Use provider name if set, otherwise use chainID+apiInterface key
	endpointLabel := rpcProviderEndpoint.Name
	if endpointLabel == "" {
		endpointLabel = rpcProviderEndpoint.Key()
	}
	resourceLimiter := NewResourceLimiter(enableResourceLimiter, endpointLabel, cuThreshold, heavyMaxConcurrent, heavyQueueSize, normalMaxConcurrent)

	if enableResourceLimiter {
		utils.LavaFormatInfo("Resource limiter enabled",
			utils.LogAttr("cu_threshold", cuThreshold),
			utils.LogAttr("heavy_max_concurrent", heavyMaxConcurrent),
			utils.LogAttr("heavy_queue_size", heavyQueueSize),
			utils.LogAttr("normal_max_concurrent", normalMaxConcurrent),
		)
	}

	rpcProviderServer.ServeRPCRequests(ctx, rpcProviderEndpoint, chainParser, rpcp.rewardServer, providerSessionManager, chainTracker, rpcp.privKey, rpcp.cache, rpcp.cacheLatestBlockEnabled, chainRouter, rpcp.providerStateTracker, rpcp.addr, rpcp.lavaChainID, DEFAULT_ALLOWED_MISSING_CU, providerMetrics, relaysMonitor, providerNodeSubscriptionManager, rpcp.staticProvider, loadManager, rpcp, numberOfRetriesAllowedOnNodeErrors, testModeConfig, resourceLimiter)
	// set up grpc listener
	var listener *ProviderListener
	func() {
		rpcp.lock.Lock()
		defer rpcp.lock.Unlock()
		var ok bool
		listener, ok = rpcp.rpcProviderListeners[rpcProviderEndpoint.NetworkAddress.Address]
		if !ok {
			utils.LavaFormatInfo("creating new gRPC listener", utils.Attribute{Key: "NetworkAddress", Value: rpcProviderEndpoint.NetworkAddress})
			listener = NewProviderListener(ctx, rpcProviderEndpoint.NetworkAddress, rpcp.grpcHealthCheckEndpoint)
			specValidator.AddRPCProviderListener(rpcProviderEndpoint.NetworkAddress.Address, listener)
			rpcp.rpcProviderListeners[rpcProviderEndpoint.NetworkAddress.Address] = listener
		}
	}()

	if listener == nil {
		utils.LavaFormatFatal("listener not defined, cant register RPCProviderServer", nil, utils.Attribute{Key: "RPCProviderEndpoint", Value: rpcProviderEndpoint.String()})
	}

	err = listener.RegisterReceiver(rpcProviderServer, rpcProviderEndpoint)
	if err != nil {
		utils.LavaFormatError("error in register receiver", err)
	}

	utils.LavaFormatInfo("provider finished setting up endpoint", utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.Key()}, utils.Attribute{Key: "Name", Value: rpcProviderEndpoint.Name})
	// prevents these objects form being overrun later
	chainParser.Activate()
	chainTracker.RegisterForBlockTimeUpdates(chainParser)
	rpcp.providerMetricsManager.SetEnabledChain(rpcProviderEndpoint.ChainID, apiInterface)
	return nil
}

func (rpcp *RPCProvider) GetLatestBlockNumForSpec(specID string) int64 {
	chainTracker, found, err := rpcp.chainTrackers.Load(specID)
	if err != nil {
		utils.LavaFormatFatal("failed to load chain tracker", err, utils.LogAttr("specID", specID))
	}
	if !found {
		return 0
	}

	block, _ := chainTracker.GetLatestBlockNum()
	return block
}

func ParseEndpointsCustomName(viper_endpoints *viper.Viper, endpointsConfigName string, geolocation uint64) (endpoints []*lavasession.RPCProviderEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(endpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal endpoints", err, utils.Attribute{Key: "viper_endpoints", Value: viper_endpoints.AllSettings()})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation
	}
	return endpoints, err
}

func ParseEndpoints(viper_endpoints *viper.Viper, geolocation uint64) (endpoints []*lavasession.RPCProviderEndpoint, err error) {
	return ParseEndpointsCustomName(viper_endpoints, common.EndpointsConfigName, geolocation)
}

// ParseStaticProviderEndpoints parses static provider configuration into extended endpoint types
func ParseStaticProviderEndpoints(viper_endpoints *viper.Viper, endpointsConfigName string, geolocation uint64) (endpoints []*lavasession.RPCStaticProviderEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(endpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal extended endpoints", err, utils.Attribute{Key: "viper_endpoints", Value: viper_endpoints.AllSettings()})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation

		// Validate that the provider name is not empty
		if err := endpoint.Validate(); err != nil {
			return nil, utils.LavaFormatError("invalid provider configuration", err,
				utils.Attribute{Key: "chainID", Value: endpoint.ChainID},
				utils.Attribute{Key: "apiInterface", Value: endpoint.ApiInterface})
		}
	}
	return endpoints, err
}

func CreateRPCProviderCobraCommand() *cobra.Command {
	cmdRPCProvider := &cobra.Command{
		Use:   `rpcprovider [config-file] | { {listen-ip:listen-port spec-chain-id api-interface "comma-separated-node-urls"} ... } --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,
		Short: `rpcprovider sets up a server to listen for rpc-consumers requests from the lava protocol send them to a configured node and respond with the reply`,
		Long: `rpcprovider sets up a server to listen for rpc-consumers requests from the lava protocol send them to a configured node and respond with the reply
		all configs should be located in` + app.DefaultNodeHome + "/config or the local running directory" + ` 
		if no arguments are passed, assumes default config file: ` + DefaultRPCProviderFileName + `
		if one argument is passed, its assumed the config file name
		--gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE are necessary to send reward transactions, according to the current lava gas price set in validators
		`,
		Example: `required flags: --geolocation 1 --from alice
optional: --save-conf
rpcprovider <flags>
rpcprovider rpcprovider_conf.yml <flags>
rpcprovider 127.0.0.1:3333 ETH1 jsonrpc wss://www.eth-node.com:80 <flags>
rpcprovider 127.0.0.1:3333 OSMOSIS tendermintrpc "wss://www.node-path.com:80,https://www.node-path.com:80" 127.0.0.1:3333 OSMOSIS rest https://www.node-path.com:1317 <flags>`,
		Args: func(cmd *cobra.Command, args []string) error {
			// Optionally run one of the validators provided by cobra
			if err := cobra.RangeArgs(0, 1)(cmd, args); err == nil {
				// zero or one argument is allowed
				return nil
			}
			if len(args)%len(Yaml_config_properties) != 0 {
				return fmt.Errorf("invalid number of arguments, either its a single config file or repeated groups of 4 HOST:PORT chain-id api-interface [node_url,node_url_2], arg count: %d", len(args))
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate --from flag: required ONLY when NOT using --static-providers
			staticProvider, _ := cmd.Flags().GetBool(common.StaticProvidersConfigName)
			fromFlag := cmd.Flags().Lookup(flags.FlagFrom)

			if !staticProvider && (fromFlag == nil || !fromFlag.Changed) {
				return fmt.Errorf("required flag \"%s\" not set (not required with --static-providers)", flags.FlagFrom)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo(common.ProcessStartLogText)
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			config_name := DefaultRPCProviderFileName
			if len(args) == 1 {
				config_name = args[0] // name of config file (without extension)
			}
			viper.SetConfigName(config_name)
			viper.SetConfigType("yml")
			viper.AddConfigPath(".")
			viper.AddConfigPath("./config")

			// set log format
			logFormat := viper.GetString(flags.FlagLogFormat)
			utils.JsonFormat = logFormat == "json"
			// set rolling log.
			closeLoggerOnFinish := common.SetupRollingLogger()
			defer closeLoggerOnFinish()

			utils.LavaFormatInfo("RPCProvider started")
			var rpcProviderEndpoints []*lavasession.RPCProviderEndpoint
			var endpoints_strings []string
			var viper_endpoints *viper.Viper
			if len(args) > 1 {
				viper_endpoints, err = common.ParseEndpointArgs(args, Yaml_config_properties, common.EndpointsConfigName)
				if err != nil {
					return utils.LavaFormatError("invalid endpoints arguments", err, utils.Attribute{Key: "endpoint_strings", Value: strings.Join(args, "")})
				}
				save_config, err := cmd.Flags().GetBool(common.SaveConfigFlagName)
				if err != nil {
					return utils.LavaFormatError("failed reading flag", err, utils.Attribute{Key: "flag_name", Value: common.SaveConfigFlagName})
				}
				viper.MergeConfigMap(viper_endpoints.AllSettings())
				if save_config {
					err := viper.SafeWriteConfigAs(DefaultRPCProviderFileName)
					if err != nil {
						utils.LavaFormatInfo("did not create new config file, if it's desired remove the config file", utils.Attribute{Key: "file_name", Value: DefaultRPCProviderFileName}, utils.Attribute{Key: "error", Value: err})
					} else {
						utils.LavaFormatInfo("created new config file", utils.Attribute{Key: "file_name", Value: DefaultRPCProviderFileName})
					}
				}
			} else if err = viper.ReadInConfig(); err != nil {
				utils.LavaFormatFatal("could not load config file", err, utils.Attribute{Key: "expected_config_name", Value: viper.ConfigFileUsed()})
			} else {
				utils.LavaFormatInfo("read config file successfully", utils.Attribute{Key: "expected_config_name", Value: viper.ConfigFileUsed()})
			}
			geolocation, err := cmd.Flags().GetUint64(lavasession.GeolocationFlag)
			if err != nil {
				utils.LavaFormatFatal("failed to read geolocation flag, required flag", err)
			}

			// Check if using static providers mode (requires provider names)
			staticProvider := viper.GetBool(common.StaticProvidersConfigName)
			if staticProvider {
				// In static provider mode, read from "endpoints" key but validate names
				staticEndpoints, err := ParseStaticProviderEndpoints(viper.GetViper(), common.EndpointsConfigName, geolocation)
				if err != nil || len(staticEndpoints) == 0 {
					return utils.LavaFormatError("invalid endpoints definition in static provider mode, must include 'name' field for each provider", err, utils.Attribute{Key: "endpoint_strings", Value: strings.Join(endpoints_strings, "")})
				}
				// Convert static endpoints to base endpoints for compatibility
				rpcProviderEndpoints = make([]*lavasession.RPCProviderEndpoint, len(staticEndpoints))
				for i, staticEndpoint := range staticEndpoints {
					rpcProviderEndpoints[i] = staticEndpoint.ToBase()
					rpcProviderEndpoints[i].Name = staticEndpoint.Name
				}
			} else {
				// Use old endpoint parsing for backwards compatibility
				rpcProviderEndpoints, err = ParseEndpoints(viper.GetViper(), geolocation)
				if err != nil || len(rpcProviderEndpoints) == 0 {
					return utils.LavaFormatError("invalid endpoints definition", err, utils.Attribute{Key: "endpoint_strings", Value: strings.Join(endpoints_strings, "")})
				}
			}

			if len(rpcProviderEndpoints) == 0 {
				return utils.LavaFormatError("no endpoints defined", err, utils.Attribute{Key: "endpoint_strings", Value: strings.Join(endpoints_strings, "")})
			}
			for _, endpoint := range rpcProviderEndpoints {
				for _, nodeUrl := range endpoint.NodeUrls {
					if nodeUrl.Url == "" {
						utils.LavaFormatError("invalid endpoint definition, empty url in nodeUrl", err, utils.Attribute{Key: "endpoint", Value: endpoint})
					}
				}
			}
			// handle flags, pass necessary fields
			ctx := context.Background()

			networkChainId := viper.GetString(flags.FlagChainID)
			if networkChainId == app.Name {
				clientTomlConfig, err := config.ReadFromClientConfig(clientCtx)
				if err == nil {
					if clientTomlConfig.ChainID != "" {
						networkChainId = clientTomlConfig.ChainID
					}
				}
			}
			utils.LavaFormatInfo("Running with chain-id:" + networkChainId)

			clientCtx = clientCtx.WithChainID(networkChainId)
			err = common.VerifyAndHandleUnsupportedFlags(cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to verify cmd flags", err)
			}

			txFactory, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to create tx factory", err)
			}
			gasPricesStr := viper.GetString(flags.FlagGasPrices)
			if gasPricesStr == "" {
				gasPricesStr = statetracker.DefaultGasPrice
			}
			txFactory = txFactory.WithGasPrices(statetracker.DefaultGasPrice)
			txFactory = txFactory.WithGasAdjustment(viper.GetFloat64(flags.FlagGasAdjustment))
			utils.LavaFormatInfo("Setting gas for tx Factory", utils.LogAttr("gas-prices", gasPricesStr), utils.LogAttr("gas-adjustment", txFactory.GasAdjustment()))

			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			// check if the command includes --pprof-address
			pprofAddressFlagUsed := cmd.Flags().Lookup("pprof-address").Changed
			if pprofAddressFlagUsed {
				// get pprof server ip address (default value: "")
				pprofServerAddress, err := cmd.Flags().GetString("pprof-address")
				if err != nil {
					utils.LavaFormatFatal("failed to read pprof address flag", err)
				}

				// start pprof HTTP server
				err = performance.StartPprofServer(pprofServerAddress)
				if err != nil {
					return utils.LavaFormatError("failed to start pprof HTTP server", err)
				}
			}
			// check if the command includes --pyroscope-address
			pyroscopeAddressFlagUsed := cmd.Flags().Lookup(performance.PyroscopeAddressFlagName).Changed
			if pyroscopeAddressFlagUsed {
				pyroscopeServerAddress, err := cmd.Flags().GetString(performance.PyroscopeAddressFlagName)
				if err != nil {
					utils.LavaFormatFatal("failed to read pyroscope address flag", err)
				}
				pyroscopeAppName, err := cmd.Flags().GetString(performance.PyroscopeAppNameFlagName)
				if err != nil || pyroscopeAppName == "" {
					pyroscopeAppName = "lavap-provider"
				}
				mutexProfileFraction, err := cmd.Flags().GetInt(performance.PyroscopeMutexProfileFractionFlagName)
				if err != nil {
					mutexProfileFraction = performance.DefaultMutexProfileFraction
				}
				blockProfileRate, err := cmd.Flags().GetInt(performance.PyroscopeBlockProfileRateFlagName)
				if err != nil {
					blockProfileRate = performance.DefaultBlockProfileRate
				}
				tagsStr, _ := cmd.Flags().GetString(performance.PyroscopeTagsFlagName)
				tags := performance.ParseTags(tagsStr)
				err = performance.StartPyroscope(pyroscopeAppName, pyroscopeServerAddress, mutexProfileFraction, blockProfileRate, tags)
				if err != nil {
					return utils.LavaFormatError("failed to start pyroscope profiler", err)
				}
			}

			utils.LavaFormatInfo("lavap Binary Version: " + upgrade.GetCurrentVersion().ProviderVersion)
			rand.InitRandomSeed()
			var cache *performance.Cache = nil
			cacheAddr := viper.GetString(performance.CacheFlagName)
			cacheLatestBlockEnabled := viper.GetBool(performance.CacheLatestBlockFlagName)
			if cacheAddr != "" {
				cache, err = performance.InitCache(ctx, cacheAddr)
				if err != nil {
					utils.LavaFormatError("Failed To Connect to cache at address", err, utils.Attribute{Key: "address", Value: cacheAddr})
				} else {
					utils.LavaFormatInfo("cache service connected", utils.Attribute{Key: "address", Value: cacheAddr}, utils.Attribute{Key: "cacheLatestBlockEnabled", Value: cacheLatestBlockEnabled})
				}
			}
			numberOfNodeParallelConnections, err := cmd.Flags().GetUint(chainproxy.ParallelConnectionsFlag)
			if err != nil {
				utils.LavaFormatFatal("error fetching chainproxy.ParallelConnectionsFlag", err)
			}
			for _, endpoint := range rpcProviderEndpoints {
				utils.LavaFormatDebug("endpoint description", utils.Attribute{Key: "endpoint", Value: endpoint})
			}
			stickinessHeaderName := viper.GetString(StickinessHeaderName)
			if stickinessHeaderName != "" {
				RPCProviderStickinessHeaderName = stickinessHeaderName
			}
			relayLoadLimit := viper.GetUint64(common.RateLimitRequestPerSecondFlag)
			prometheusListenAddr := viper.GetString(metrics.MetricsListenFlagName)
			rewardStoragePath := viper.GetString(rewardserver.RewardServerStorageFlagName)
			rewardTTL := viper.GetDuration(rewardserver.RewardTTLFlagName)
			shardID := viper.GetUint(ShardIDFlagName)
			rewardsSnapshotThreshold := viper.GetUint(rewardserver.RewardsSnapshotThresholdFlagName)
			rewardsSnapshotTimeoutSec := viper.GetUint(rewardserver.RewardsSnapshotTimeoutSecFlagName)
			enableRelaysHealth := viper.GetBool(common.RelaysHealthEnableFlag)
			relaysHealthInterval := viper.GetDuration(common.RelayHealthIntervalFlag)
			healthCheckURLPath := viper.GetString(HealthCheckURLPathFlagName)
			offlineSpecPath := viper.GetString(common.UseStaticSpecFlag)
			githubToken := viper.GetString(common.GitHubTokenFlag)
			epochDuration := viper.GetDuration(common.EpochDurationFlag)
			enableMemoryLogs := viper.GetBool(common.EnableMemoryLogsFlag)
			memoryutils.EnableMemoryLogs(enableMemoryLogs)

			// If running with --static-providers, enable standalone mode
			if staticProvider {
				utils.LavaFormatWarning("Running in static provider mode (standalone, no Lava blockchain connection)", nil)
				// Note: Epoch duration will default to 15 minutes (StandaloneEpochDuration) if not specified

				// Automatically skip relay signing in static provider mode unless explicitly set
				// This saves CPU and memory since rewards are not claimed in standalone mode
				if !viper.IsSet(common.SkipRelaySigningFlag) {
					lavaprotocol.SkipRelaySigning = true
					utils.LavaFormatInfo("[SkipRelaySigning] Static provider mode: automatically enabling skip-relay-signing for performance",
						utils.Attribute{Key: "skipRelaySigning", Value: lavaprotocol.SkipRelaySigning},
						utils.Attribute{Key: "reason", Value: "auto-enabled for static provider mode"},
					)
				} else {
					// Flag was explicitly set, log the value
					explicitValue := viper.GetBool(common.SkipRelaySigningFlag)
					lavaprotocol.SkipRelaySigning = explicitValue
					utils.LavaFormatInfo("[SkipRelaySigning] Static provider mode: using explicit flag value",
						utils.Attribute{Key: "skipRelaySigning", Value: lavaprotocol.SkipRelaySigning},
						utils.Attribute{Key: "source", Value: "command-line/config file"},
					)
				}
			}

			// Load test mode configuration
			testMode := viper.GetBool("test_mode")
			testResponsesFile := viper.GetString("test_responses")

			if testMode && testResponsesFile == "" {
				return utils.LavaFormatError("test_responses file is required when test_mode is enabled", nil)
			}

			// Create resource limiter options
			enableResourceLimiter, _ := cmd.Flags().GetBool("enable-resource-limiter")
			cuThreshold, _ := cmd.Flags().GetUint64("resource-limiter-cu-threshold")
			heavyMaxConcurrent, _ := cmd.Flags().GetInt64("heavy-max-concurrent")
			heavyQueueSize, _ := cmd.Flags().GetInt("heavy-queue-size")
			normalMaxConcurrent, _ := cmd.Flags().GetInt64("normal-max-concurrent")

			resourceLimiterOptions := &resourceLimiterOptions{
				enabled:             enableResourceLimiter,
				cuThreshold:         cuThreshold,
				heavyMaxConcurrent:  heavyMaxConcurrent,
				heavyQueueSize:      heavyQueueSize,
				normalMaxConcurrent: normalMaxConcurrent,
			}

			// Get memory GC threshold
			memoryGCThresholdGB, err := cmd.Flags().GetFloat64(common.MemoryGCThresholdGBFlagName)
			if err != nil {
				utils.LavaFormatWarning("failed to read memory GC threshold flag, using default (0 = disabled)", err)
				memoryGCThresholdGB = 0
			}

			rpcProviderHealthCheckMetricsOptions := rpcProviderHealthCheckMetricsOptions{
				enableRelaysHealth,
				relaysHealthInterval,
				healthCheckURLPath,
			}

			rpcProviderStartOptions := rpcProviderStartOptions{
				ctx,
				txFactory,
				clientCtx,
				rpcProviderEndpoints,
				cache,
				numberOfNodeParallelConnections,
				prometheusListenAddr,
				rewardStoragePath,
				rewardTTL,
				shardID,
				rewardsSnapshotThreshold,
				rewardsSnapshotTimeoutSec,
				&rpcProviderHealthCheckMetricsOptions,
				staticProvider,
				offlineSpecPath,
				githubToken,
				relayLoadLimit,
				testMode,
				testResponsesFile,
				epochDuration,
				resourceLimiterOptions,
				memoryGCThresholdGB,
			}

			verificationsResponseCache, err := ristretto.NewCache(
				&ristretto.Config[string, []*pairingtypes.Verification]{
					NumCounters:        CacheNumCounters,
					MaxCost:            CacheMaxCost,
					BufferItems:        64,
					IgnoreInternalCost: true,
				})
			if err != nil {
				utils.LavaFormatFatal("failed setting up cache for verificationsResponseCache", err)
			}
			rpcProvider := RPCProvider{
				verificationsResponseCache: verificationsResponseCache,
				testMode:                   testMode,
				testResponsesFile:          testResponsesFile,
				cacheLatestBlockEnabled:    cacheLatestBlockEnabled,
			}

			// Load test mode config if enabled
			if testMode {
				utils.LavaFormatInfo("Test mode enabled", utils.LogAttr("testResponsesFile", testResponsesFile))
			}

			err = rpcProvider.Start(&rpcProviderStartOptions)
			return err
		},
	}

	// RPCProvider command flags
	flags.AddTxFlagsToCmd(cmdRPCProvider)
	// Note: --from is validated in PreRunE (required only when NOT using --static-providers)
	cmdRPCProvider.Flags().Bool(common.StaticProvidersConfigName, false, "set the provider as static, allowing it to get requests from anyone, and skipping rewards, can be used for local tests")
	cmdRPCProvider.Flags().Bool(common.SaveConfigFlagName, false, "save cmd args to a config file")
	cmdRPCProvider.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCProvider.MarkFlagRequired(common.GeolocationFlag)
	cmdRPCProvider.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCProvider.Flags().String(performance.PyroscopeAddressFlagName, "", "pyroscope server address for continuous profiling (e.g., http://pyroscope:4040)")
	cmdRPCProvider.Flags().String(performance.PyroscopeAppNameFlagName, "lavap-provider", "pyroscope application name for identifying this service")
	cmdRPCProvider.Flags().Int(performance.PyroscopeMutexProfileFractionFlagName, performance.DefaultMutexProfileFraction, "mutex profile sampling rate (1 in N mutex events)")
	cmdRPCProvider.Flags().Int(performance.PyroscopeBlockProfileRateFlagName, performance.DefaultBlockProfileRate, "block profile rate in nanoseconds (1 records all blocking events)")
	cmdRPCProvider.Flags().String(performance.PyroscopeTagsFlagName, "", "comma-separated list of tags in key=value format (e.g., instance=provider-1,region=us-east)")
	cmdRPCProvider.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdRPCProvider.Flags().Bool(performance.CacheLatestBlockFlagName, false, "enable caching for latest block requests (default: false)")
	cmdRPCProvider.Flags().Uint(chainproxy.ParallelConnectionsFlag, chainproxy.NumberOfParallelConnections, "parallel connections")
	cmdRPCProvider.Flags().String(flags.FlagLogLevel, "debug", "log level")
	cmdRPCProvider.Flags().String(metrics.MetricsListenFlagName, metrics.DisabledFlagOption, "the address to expose prometheus metrics (such as localhost:7779)")
	cmdRPCProvider.Flags().String(rewardserver.RewardServerStorageFlagName, rewardserver.DefaultRewardServerStorage, "the path to store reward server data")
	cmdRPCProvider.Flags().Duration(rewardserver.RewardTTLFlagName, rewardserver.DefaultRewardTTL, "reward time to live")
	cmdRPCProvider.Flags().Uint(ShardIDFlagName, DefaultShardID, "shard id")
	cmdRPCProvider.Flags().Uint(rewardserver.RewardsSnapshotThresholdFlagName, rewardserver.DefaultRewardsSnapshotThreshold, "the number of rewards to wait until making snapshot of the rewards memory")
	cmdRPCProvider.Flags().Uint(rewardserver.RewardsSnapshotTimeoutSecFlagName, rewardserver.DefaultRewardsSnapshotTimeoutSec, "the seconds to wait until making snapshot of the rewards memory")
	cmdRPCProvider.Flags().String(StickinessHeaderName, RPCProviderStickinessHeaderName, "the name of the header to be attached to requests for stickiness by consumer, used for consistency")
	cmdRPCProvider.Flags().Bool("test_mode", false, "enable test mode - provider returns predefined responses instead of querying real nodes")
	cmdRPCProvider.Flags().String("test_responses", "", "path to JSON file containing test responses for different API methods (required when test_mode is enabled)")
	cmdRPCProvider.Flags().Uint64Var(&chaintracker.PollingMultiplier, chaintracker.PollingMultiplierFlagName, 1, "when set, forces the chain tracker to poll more often, improving the sync at the cost of more queries")
	cmdRPCProvider.Flags().DurationVar(&SpecValidationInterval, SpecValidationIntervalFlagName, SpecValidationInterval, "determines the interval of which to run validation on the spec for all connected chains")
	cmdRPCProvider.Flags().DurationVar(&SpecValidationIntervalDisabledChains, SpecValidationIntervalDisabledChainsFlagName, SpecValidationIntervalDisabledChains, "determines the interval of which to run validation on the spec for all disabled chains, determines recovery time")
	cmdRPCProvider.Flags().Bool(common.RelaysHealthEnableFlag, true, "enables relays health check")
	cmdRPCProvider.Flags().Duration(common.RelayHealthIntervalFlag, RelayHealthIntervalFlagDefault, "interval between relay health checks")
	cmdRPCProvider.Flags().String(HealthCheckURLPathFlagName, HealthCheckURLPathFlagDefault, "the url path for the provider's grpc health check")
	cmdRPCProvider.Flags().DurationVar(&updaters.TimeOutForFetchingLavaBlocks, common.TimeOutForFetchingLavaBlocksFlag, time.Second*5, "setting the timeout for fetching lava blocks")
	cmdRPCProvider.Flags().IntVar(&numberOfRetriesAllowedOnNodeErrors, common.SetRelayCountOnNodeErrorFlag, 2, "set the number of retries attempt on node errors")
	cmdRPCProvider.Flags().String(common.UseStaticSpecFlag, "", "load offline spec provided path to spec file, used to test specs before they are proposed on chain, example for spec with inheritance: --use-static-spec ./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/mainnet-1/specs/ethermint.json,./specs/mainnet-1/specs/ethereum.json,./specs/mainnet-1/specs/evmos.json")
	cmdRPCProvider.Flags().String(common.GitHubTokenFlag, "", "GitHub personal access token for accessing private repositories and higher API rate limits (5,000 requests/hour vs 60 for unauthenticated)")
	cmdRPCProvider.Flags().Duration(common.EpochDurationFlag, 0, "duration of each epoch for time-based epoch system (e.g., 30m, 1h). If not set, epochs are disabled")
	cmdRPCProvider.Flags().Uint64(common.RateLimitRequestPerSecondFlag, 0, "Measuring the load relative to this number for feedback - per second - per chain - default unlimited. Given Y simultaneous relay calls, a value of X  and will measure Y/X load rate.")
	cmdRPCProvider.Flags().BoolVar(&chainlib.SkipWebsocketVerification, common.SkipWebsocketVerificationFlag, false, "skip websocket verification")
	cmdRPCProvider.Flags().BoolVar(&lavaprotocol.SkipRelaySigning, common.SkipRelaySigningFlag, lavaprotocol.SkipRelaySigning, "skip cryptographic signing of relay responses to reduce CPU and memory usage (use only with static providers)")
	cmdRPCProvider.Flags().BoolVar(&metrics.ShowProviderEndpointInProviderMetrics, common.ShowProviderEndpointInMetricsFlagName, metrics.ShowProviderEndpointInProviderMetrics, "show provider endpoint in provider metrics")
	cmdRPCProvider.Flags().Bool("enable-resource-limiter", false, "Enable method-specific resource limiting to prevent OOM from high-CU requests")
	cmdRPCProvider.Flags().Uint64("resource-limiter-cu-threshold", 100, "CU threshold above which methods are considered 'heavy' (default: 100)")
	cmdRPCProvider.Flags().Int64("heavy-max-concurrent", 2, "Max concurrent heavy (high-CU/debug/trace) method calls")
	cmdRPCProvider.Flags().Int("heavy-queue-size", 5, "Queue size for heavy methods")
	cmdRPCProvider.Flags().Int64("normal-max-concurrent", 100, "Max concurrent normal method calls")
	cmdRPCProvider.Flags().Bool(common.EnableMemoryLogsFlag, false, "enable memory tracking logs")
	cmdRPCProvider.Flags().Float64(common.MemoryGCThresholdGBFlagName, 0, "Memory GC threshold in GB - triggers GC when heap in use exceeds this value (0 = disabled)")
	common.AddRollingLogConfig(cmdRPCProvider)
	return cmdRPCProvider
}

func addMissingInternalPaths(rpcProviderEndpoint *lavasession.RPCProviderEndpoint, chainParser chainlib.ChainParser, chainID, apiInterface string, underConfiguredInternalPaths []string) {
	// Find root URLs (those with empty internal paths)
	var httpRootUrl, wssRootUrl *common.NodeUrl
	for _, nodeUrl := range rpcProviderEndpoint.NodeUrls {
		if nodeUrl.InternalPath == "" {
			if strings.HasPrefix(strings.ToLower(nodeUrl.Url), "wss://") || strings.HasPrefix(strings.ToLower(nodeUrl.Url), "ws://") {
				wssRootUrl = &nodeUrl
			} else {
				httpRootUrl = &nodeUrl
			}
		}
	}

	// Add missing paths using the appropriate root URL as template
	for _, missingPath := range underConfiguredInternalPaths {
		isWS := false
		for _, connectionType := range []string{"POST", ""} {
			// check subscription exists, we only care for subscription API's because otherwise we use http anyway.
			collectionKey := chainlib.CollectionKey{
				InternalPath:   missingPath,
				Addon:          "",
				ConnectionType: connectionType,
			}

			if chainParser.IsTagInCollection(spectypes.FUNCTION_TAG_SUBSCRIBE, collectionKey) {
				isWS = true
			}
		}

		if isWS {
			if wssRootUrl != nil {
				newUrl := *wssRootUrl // Create copy of root URL
				newUrl.InternalPath = missingPath
				newUrl.Url += missingPath
				rpcProviderEndpoint.NodeUrls = append(rpcProviderEndpoint.NodeUrls, newUrl)
			}
		} else if httpRootUrl != nil {
			newUrl := *httpRootUrl // Create copy of root URL
			newUrl.InternalPath = missingPath
			newUrl.Url += missingPath
			rpcProviderEndpoint.NodeUrls = append(rpcProviderEndpoint.NodeUrls, newUrl)
		}
	}

	utils.LavaFormatDebug("Added missing internal paths to NodeUrls",
		utils.LogAttr("chainID", chainID),
		utils.LogAttr("apiInterface", apiInterface),
		utils.LogAttr("addedPaths", strings.Join(underConfiguredInternalPaths, ",")))
}
