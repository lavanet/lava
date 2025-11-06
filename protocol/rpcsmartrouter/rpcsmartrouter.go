// Package rpcsmartrouter provides a centralized RPC routing solution for the Lava protocol.
//
// # Architecture Overview
//
// The smart router is designed for centralized deployments where providers are statically
// configured rather than dynamically discovered through blockchain pairing. This is useful for:
//   - Enterprise deployments with known provider infrastructure
//   - Testing and development environments
//   - Use cases requiring predictable provider routing
//
// # Key Differences from rpcconsumer
//
// rpcsmartrouter (centralized):
//   - Uses pre-configured static providers from configuration files
//   - No blockchain state tracking required
//   - Provider selection based on configured weights (static providers get 10x multiplier)
//   - No epoch management or on-chain pairing updates
//
// rpcconsumer (decentralized):
//   - Discovers providers dynamically through blockchain pairing
//   - Tracks blockchain state, epochs, and provider stake
//   - Provider selection weighted by actual on-chain stake
//   - Includes conflict detection and finalization consensus
//
// # Provider Selection
//
// Static providers are configured in YAML files and automatically receive a 10x weight
// multiplier compared to blockchain providers. This ensures static providers are preferred
// in routing decisions. See StaticProviderDummyCoin for implementation details.
package rpcsmartrouter

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/app"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/performance"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/protocol/rpcprovider"
	"github.com/lavanet/lava/v5/protocol/statetracker"
	"github.com/lavanet/lava/v5/protocol/upgrade"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DefaultRPCSmartRouterFileName = "rpcsmartrouter.yml"
	DebugRelaysFlagName           = "debug-relays"
	DebugProbesFlagName           = "debug-probes"
	refererBackendAddressFlagName = "referer-be-address"
	refererMarkerFlagName         = "referer-marker"
	reportsSendBEAddress          = "reports-be-address"
)

var (
	Yaml_config_properties         = []string{"network-address", "chain-id", "api-interface"}
	RelaysHealthEnableFlagDefault  = true
	RelayHealthIntervalFlagDefault = 5 * time.Minute

	// StaticProviderDummyCoin is used for type compatibility in provider sessions.
	// Static providers don't use blockchain stake for selection; they automatically
	// receive a 10x weight multiplier in CalcWeightsByStake (see lavasession package).
	// The coin value is ignored but the object must exist to avoid nil pointer errors.
	StaticProviderDummyCoin = sdk.NewCoin("ulava", sdk.NewInt(1))
)

// staticPolicy is a simple implementation of chainlib.PolicyInf
// used to configure the chain parser with allowed extensions and addons
// derived from static provider configurations.
type staticPolicy struct {
	addons       []string
	extensions   []string
	apiInterface string
}

func (p staticPolicy) GetSupportedAddons(string) ([]string, error) {
	return p.addons, nil
}

func (p staticPolicy) GetSupportedExtensions(string) ([]epochstoragetypes.EndpointService, error) {
	services := make([]epochstoragetypes.EndpointService, 0, len(p.extensions))
	for _, ext := range p.extensions {
		services = append(services, epochstoragetypes.EndpointService{
			Extension:    ext,
			ApiInterface: p.apiInterface,
		})
	}
	return services, nil
}

type strategyValue struct {
	provideroptimizer.Strategy
}

var strategyNames = []string{
	"balanced",
	"latency",
	"sync-freshness",
	"cost",
	"privacy",
	"accuracy",
	"distributed",
}

var strategyFlag strategyValue = strategyValue{Strategy: provideroptimizer.StrategyBalanced}

func (s *strategyValue) String() string {
	return strategyNames[int(s.Strategy)]
}

func (s *strategyValue) Set(str string) error {
	for i, name := range strategyNames {
		if strings.EqualFold(str, name) {
			s.Strategy = provideroptimizer.Strategy(i)
			return nil
		}
	}
	return fmt.Errorf("invalid strategy: %s", str)
}

func (s *strategyValue) Type() string {
	return "string"
}

type AnalyticsServerAddresses struct {
	AddApiMethodCallsMetrics bool
	MetricsListenAddress     string
	RelayServerAddress       string
	RelayKafkaAddress        string
	RelayKafkaTopic          string
	RelayKafkaUsername       string
	RelayKafkaPassword       string
	RelayKafkaMechanism      string
	RelayKafkaTLSEnabled     bool
	RelayKafkaTLSInsecure    bool
	ReportsAddressFlag       string
	OptimizerQoSAddress      string
	OptimizerQoSListen       bool
}
type RPCSmartRouter struct {
	// Smart router doesn't need blockchain state tracking
	epochTimer             *common.EpochTimer
	sessionManagers        map[string]*lavasession.ConsumerSessionManager                  // key: chainID-apiInterface
	providerSessions       map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider // key: chainID-apiInterface
	backupProviderSessions map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider // key: chainID-apiInterface
}

type rpcSmartRouterStartOptions struct {
	rpcEndpoints             []*lavasession.RPCEndpoint
	requiredResponses        int
	cache                    *performance.Cache
	strategy                 provideroptimizer.Strategy
	maxConcurrentProviders   uint
	analyticsServerAddresses AnalyticsServerAddresses
	cmdFlags                 common.ConsumerCmdFlags
	stateShare               bool
	refererData              *chainlib.RefererData
	staticProvidersList      []*lavasession.RPCStaticProviderEndpoint // define static providers as primary providers
	backupProvidersList      []*lavasession.RPCStaticProviderEndpoint // define backup providers as emergency fallback when no providers available
	geoLocation              uint64
	clientCtx                client.Context    // Blockchain client context for querying specs
	privKey                  *btcec.PrivateKey // Private key for signing relay requests
	lavaChainID              string            // Lava blockchain chain ID
}

// spawns a new RPCConsumer server with all it's processes and internals ready for communications
func (rpsr *RPCSmartRouter) Start(ctx context.Context, options *rpcSmartRouterStartOptions) (err error) {
	if common.IsTestMode(ctx) {
		testModeWarn("RPCSmartRouter running tests")
	}

	// Initialize session managers and provider sessions maps for epoch timer callbacks
	rpsr.sessionManagers = make(map[string]*lavasession.ConsumerSessionManager)
	rpsr.providerSessions = make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider)
	rpsr.backupProviderSessions = make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider)

	// RPCSmartRouter always runs in standalone mode with time-based epochs
	epochDuration := options.cmdFlags.EpochDuration
	if epochDuration == 0 {
		epochDuration = common.StandaloneEpochDuration // 15 minutes default for standalone
	}

	rpsr.epochTimer = common.NewEpochTimer(epochDuration)
	currentEpoch := rpsr.epochTimer.GetCurrentEpoch()
	timeUntilNext := rpsr.epochTimer.GetTimeUntilNextEpoch()

	utils.LavaFormatInfo("RPCSmartRouter: using time-based epochs (standalone mode)",
		utils.LogAttr("epochDuration", epochDuration),
		utils.LogAttr("currentEpoch", currentEpoch),
		utils.LogAttr("timeUntilNextEpoch", timeUntilNext),
		utils.LogAttr("nextEpochTime", time.Now().Add(timeUntilNext).Format("15:04:05 MST")),
	)

	options.refererData.ReferrerClient = metrics.NewConsumerReferrerClient(options.refererData.Address)
	smartRouterReportsManager := metrics.NewConsumerReportsClient(options.analyticsServerAddresses.ReportsAddressFlag)

	// Smart router doesn't need consumer address from blockchain
	// Using a static identifier for metrics and logging
	smartRouterIdentifier := "smart-router-" + strconv.FormatUint(rand.Uint64(), 10)

	smartRouterUsageServeManager := metrics.NewConsumerRelayServerClient(options.analyticsServerAddresses.RelayServerAddress)                                                                                                                                                                                                                                                                                                                     // start up relay server reporting
	smartRouterKafkaClient := metrics.NewConsumerKafkaClient(options.analyticsServerAddresses.RelayKafkaAddress, options.analyticsServerAddresses.RelayKafkaTopic, options.analyticsServerAddresses.RelayKafkaUsername, options.analyticsServerAddresses.RelayKafkaPassword, options.analyticsServerAddresses.RelayKafkaMechanism, options.analyticsServerAddresses.RelayKafkaTLSEnabled, options.analyticsServerAddresses.RelayKafkaTLSInsecure) // start up kafka client
	var smartRouterOptimizerQoSClient *metrics.ConsumerOptimizerQoSClient
	if options.analyticsServerAddresses.OptimizerQoSAddress != "" || options.analyticsServerAddresses.OptimizerQoSListen {
		smartRouterOptimizerQoSClient = metrics.NewConsumerOptimizerQoSClient(smartRouterIdentifier, options.analyticsServerAddresses.OptimizerQoSAddress, options.geoLocation, metrics.OptimizerQosServerPushInterval) // start up optimizer qos client
		smartRouterOptimizerQoSClient.StartOptimizersQoSReportsCollecting(ctx, metrics.OptimizerQosServerSamplingInterval)
	}
	smartRouterMetricsManager := metrics.NewConsumerMetricsManager(metrics.ConsumerMetricsManagerOptions{
		NetworkAddress:             options.analyticsServerAddresses.MetricsListenAddress,
		AddMethodsApiGauge:         options.analyticsServerAddresses.AddApiMethodCallsMetrics,
		EnableQoSListener:          options.analyticsServerAddresses.OptimizerQoSListen,
		ConsumerOptimizerQoSClient: smartRouterOptimizerQoSClient,
	}) // start up prometheus metrics
	rpcSmartRouterMetrics, err := metrics.NewRPCConsumerLogs(smartRouterMetricsManager, smartRouterUsageServeManager, smartRouterKafkaClient, smartRouterOptimizerQoSClient)
	if err != nil {
		utils.LavaFormatFatal("failed creating RPCSmartRouter logs", err)
	}

	smartRouterMetricsManager.SetVersion(upgrade.GetCurrentVersion().ConsumerVersion)

	// we want one provider optimizer per chain so we will store them for reuse across rpcEndpoints
	chainMutexes := map[string]*sync.Mutex{}
	for _, endpoint := range options.rpcEndpoints {
		chainMutexes[endpoint.ChainID] = &sync.Mutex{} // create a mutex per chain for shared resources
	}

	optimizers := &common.SafeSyncMap[string, *provideroptimizer.ProviderOptimizer]{}
	smartRouterConsistencies := &common.SafeSyncMap[string, relaycore.Consistency]{}

	var wg sync.WaitGroup
	parallelJobs := len(options.rpcEndpoints)
	wg.Add(parallelJobs)

	errCh := make(chan error)

	utils.LavaFormatInfo("RPCSmartRouter identifier: " + smartRouterIdentifier)
	utils.LavaFormatInfo("RPCSmartRouter setting up endpoints", utils.Attribute{Key: "length", Value: strconv.Itoa(parallelJobs)})

	relaysMonitorAggregator := metrics.NewRelaysMonitorAggregator(options.cmdFlags.RelaysHealthIntervalFlag, smartRouterMetricsManager)
	for _, rpcEndpoint := range options.rpcEndpoints {
		go func(rpcEndpoint *lavasession.RPCEndpoint) error {
			defer wg.Done()
			err := rpsr.CreateSmartRouterEndpoint(ctx, rpcEndpoint, errCh,
				optimizers, smartRouterConsistencies, chainMutexes,
				options, smartRouterIdentifier, rpcSmartRouterMetrics, smartRouterReportsManager, smartRouterOptimizerQoSClient,
				smartRouterMetricsManager, relaysMonitorAggregator)
			return err
		}(rpcEndpoint)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}

	// Start epoch timer after all endpoints are set up
	// Register all session managers for epoch updates
	for chainKey, sm := range rpsr.sessionManagers {
		sessionManager := sm // Capture for closure
		chainKeyLog := chainKey
		providerSessions := rpsr.providerSessions[chainKey]
		backupSessions := rpsr.backupProviderSessions[chainKey]

		rpsr.epochTimer.RegisterCallback(func(epoch uint64) {
			utils.LavaFormatInfo("ConsumerSessionManager: Epoch update triggered",
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("chainKey", chainKeyLog),
				utils.LogAttr("time", time.Now().Format("15:04:05 MST")),
			)

			// Update PairingEpoch for all static provider sessions
			// This is critical to prevent epoch mismatch errors during session creation
			for _, providerSession := range providerSessions {
				providerSession.Lock.Lock()
				providerSession.PairingEpoch = epoch
				providerSession.Lock.Unlock()
			}
			for _, backupSession := range backupSessions {
				backupSession.Lock.Lock()
				backupSession.PairingEpoch = epoch
				backupSession.Lock.Unlock()
			}

			// Update session manager with current pairing (static in standalone mode)
			// The pairing doesn't change, but this triggers cleanup and provider unblocking
			err := sessionManager.UpdateAllProviders(epoch, providerSessions, backupSessions)
			if err != nil {
				utils.LavaFormatError("Failed to update providers on epoch change", err,
					utils.LogAttr("epoch", epoch),
					utils.LogAttr("chainKey", chainKeyLog),
				)
			}
		})

		utils.LavaFormatInfo("RPCSmartRouter: Registered session manager for epoch updates",
			utils.LogAttr("chainKey", chainKey),
		)
	}

	// Start the epoch timer
	rpsr.epochTimer.Start(ctx)

	relaysMonitorAggregator.StartMonitoring(ctx)

	utils.LavaFormatInfo("RPCSmartRouter done setting up all endpoints, ready for requests")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	return nil
}

func (rpsr *RPCSmartRouter) CreateSmartRouterEndpoint(
	ctx context.Context,
	rpcEndpoint *lavasession.RPCEndpoint,
	errCh chan error,
	optimizers *common.SafeSyncMap[string, *provideroptimizer.ProviderOptimizer],
	smartRouterConsistencies *common.SafeSyncMap[string, relaycore.Consistency],
	chainMutexes map[string]*sync.Mutex,
	options *rpcSmartRouterStartOptions,
	smartRouterIdentifier string,
	rpcSmartRouterMetrics *metrics.RPCConsumerLogs,
	smartRouterReportsManager *metrics.ConsumerReportsClient,
	smartRouterOptimizerQoSClient *metrics.ConsumerOptimizerQoSClient,
	smartRouterMetricsManager *metrics.ConsumerMetricsManager,
	relaysMonitorAggregator *metrics.RelaysMonitorAggregator,
) error {
	chainParser, err := chainlib.NewChainParser(rpcEndpoint.ApiInterface)
	if err != nil {
		err = utils.LavaFormatError("failed creating chain parser", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
		errCh <- err
		return err
	}
	chainID := rpcEndpoint.ChainID

	// Load spec from static file or query from blockchain
	// Smart router queries spec once during initialization (no ongoing updates)
	if options.cmdFlags.StaticSpecPath != "" {
		// Load spec from static file/directory
		err = statetracker.RegisterForSpecUpdatesOrSetStaticSpecWithToken(ctx, chainParser, options.cmdFlags.StaticSpecPath, *rpcEndpoint, nil, options.cmdFlags.GitHubToken)
		if err != nil {
			err = utils.LavaFormatError("failed loading static spec", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
			errCh <- err
			return err
		}
	} else {
		// Query spec once from blockchain (no ongoing updates like rpcconsumer)
		specQueryClient := spectypes.NewQueryClient(options.clientCtx)
		chainSpecResponse, err := specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
			ChainID: chainID,
		})
		if err != nil {
			err = utils.LavaFormatError("failed querying chain spec from blockchain", err, utils.Attribute{Key: "chainID", Value: chainID})
			errCh <- err
			return err
		}
		chainParser.SetSpec(chainSpecResponse.Spec)
		utils.LavaFormatInfo("Loaded spec from blockchain", utils.Attribute{Key: "chainID", Value: chainID})
	}

	// Filter the relevant static providers
	relevantStaticProviderList := []*lavasession.RPCStaticProviderEndpoint{}
	for _, staticProvider := range options.staticProvidersList {
		if staticProvider.ChainID == rpcEndpoint.ChainID {
			relevantStaticProviderList = append(relevantStaticProviderList, staticProvider)
		}
	}

	// Filter backup providers for this chain (needed for policy derivation)
	relevantBackupProviderList := []*lavasession.RPCStaticProviderEndpoint{}
	for _, backupProvider := range options.backupProvidersList {
		if backupProvider.ChainID == rpcEndpoint.ChainID {
			relevantBackupProviderList = append(relevantBackupProviderList, backupProvider)
		}
	}

	if len(relevantStaticProviderList) == 0 && len(relevantBackupProviderList) == 0 {
		err = utils.LavaFormatError("no static or backup providers configured for chain", nil,
			utils.Attribute{Key: "chainID", Value: chainID})
		errCh <- err
		return err
	}

	// Auto-derive policy from BOTH static and backup providers' addons
	// This configures the extension parser and allowed addons based on what ALL providers support
	addonsMap := make(map[string]struct{})
	extensionsMap := make(map[string]struct{})

	// IMPORTANT: Always allow the default addon (empty string) for standard APIs
	// Without this, all standard requests without explicit addons will fail validation
	addonsMap[""] = struct{}{}

	// Scan static providers for addons
	for _, staticProvider := range relevantStaticProviderList {
		for _, nodeUrl := range staticProvider.NodeUrls {
			for _, addon := range nodeUrl.Addons {
				// Add the addon itself to policy
				addonsMap[addon] = struct{}{}
				// If provider has "archive" addon, also allow "archive" extension
				// This enables the archive retry mechanism to work correctly
				if addon == "archive" {
					extensionsMap["archive"] = struct{}{}
				}
				// Future addon->extension mappings can be added here
			}
		}
	}

	// Scan backup providers for addons (same logic as static providers)
	for _, backupProvider := range relevantBackupProviderList {
		for _, nodeUrl := range backupProvider.NodeUrls {
			for _, addon := range nodeUrl.Addons {
				addonsMap[addon] = struct{}{}
				if addon == "archive" {
					extensionsMap["archive"] = struct{}{}
				}
			}
		}
	}

	// Convert maps to slices for the policy struct
	addons := make([]string, 0, len(addonsMap))
	for addon := range addonsMap {
		addons = append(addons, addon)
	}
	extensions := make([]string, 0, len(extensionsMap))
	for ext := range extensionsMap {
		extensions = append(extensions, ext)
	}

	// Apply the derived policy to the chain parser if we found any addons or extensions
	if len(addons) > 0 || len(extensions) > 0 {
		policy := staticPolicy{
			addons:       addons,
			extensions:   extensions,
			apiInterface: rpcEndpoint.ApiInterface,
		}
		err = chainParser.SetPolicy(policy, chainID, rpcEndpoint.ApiInterface)
		if err != nil {
			utils.LavaFormatWarning("Failed to set auto-derived policy", err,
				utils.Attribute{Key: "chainID", Value: chainID},
				utils.Attribute{Key: "apiInterface", Value: rpcEndpoint.ApiInterface})
		} else {
			utils.LavaFormatInfo("Auto-derived policy from static providers",
				utils.Attribute{Key: "chainID", Value: chainID},
				utils.Attribute{Key: "apiInterface", Value: rpcEndpoint.ApiInterface},
				utils.Attribute{Key: "addons", Value: addons},
				utils.Attribute{Key: "extensions", Value: extensions})
		}
	}

	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	var optimizer *provideroptimizer.ProviderOptimizer
	var smartRouterConsistency relaycore.Consistency

	// Create chain assets with mutex protection
	chainMutexes[chainID].Lock()
	defer chainMutexes[chainID].Unlock()

	// Create / Use existing optimizer
	// QoS selection is disabled by default (false) - can be enabled via command flag in the future
	newOptimizer := provideroptimizer.NewProviderOptimizer(options.strategy, averageBlockTime, options.maxConcurrentProviders, smartRouterOptimizerQoSClient, chainID, false)
	optimizer, loaded, err := optimizers.LoadOrStore(chainID, newOptimizer)
	if err != nil {
		errCh <- err
		return utils.LavaFormatError("failed loading optimizer", err, utils.LogAttr("endpoint", rpcEndpoint.Key()))
	}

	if !loaded && smartRouterOptimizerQoSClient != nil {
		// if this is a new optimizer, register it in the smartRouterOptimizerQoSClient
		smartRouterOptimizerQoSClient.RegisterOptimizer(optimizer, chainID)
	}

	// Create / Use existing Consistency
	newSmartRouterConsistency := relaycore.NewConsistency(chainID)
	smartRouterConsistency, _, err = smartRouterConsistencies.LoadOrStore(chainID, newSmartRouterConsistency)
	if err != nil {
		errCh <- err
		return utils.LavaFormatError("failed loading consumer consistency", err, utils.LogAttr("endpoint", rpcEndpoint.Key()))
	}

	// Create active subscription provider storage for each unique chain
	activeSubscriptionProvidersStorage := lavasession.NewActiveSubscriptionProvidersStorage()
	sessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, smartRouterMetricsManager, smartRouterReportsManager, smartRouterIdentifier, activeSubscriptionProvidersStorage)

	// Store session manager in router for epoch timer callbacks (thread-safe via CreateSmartRouterEndpoint mutex)
	sessionManagerKey := rpcEndpoint.Key() // chainID-apiInterface
	if rpsr.sessionManagers == nil {
		rpsr.sessionManagers = make(map[string]*lavasession.ConsumerSessionManager)
	}
	rpsr.sessionManagers[sessionManagerKey] = sessionManager

	if lavasession.PeriodicProbeProviders {
		go sessionManager.PeriodicProbeProviders(ctx, lavasession.PeriodicProbeProvidersInterval)
	}

	// Helper function to convert provider endpoints to sessions
	convertProvidersToSessions := func(providerList []*lavasession.RPCStaticProviderEndpoint) map[uint64]*lavasession.ConsumerSessionsWithProvider {
		sessions := make(map[uint64]*lavasession.ConsumerSessionsWithProvider)
		for idx, provider := range providerList {
			// Only process providers matching this endpoint's API interface
			if provider.ApiInterface != rpcEndpoint.ApiInterface || provider.ChainID != rpcEndpoint.ChainID {
				continue
			}

			endpoints := []*lavasession.Endpoint{}
			for _, url := range provider.NodeUrls {
				extensions := map[string]struct{}{}
				for _, extension := range url.Addons {
					extensions[extension] = struct{}{}
				}

				endpoint := &lavasession.Endpoint{
					NetworkAddress: url.Url,
					Enabled:        true,
					Addons:         extensions,
					Extensions:     extensions,
					Connections:    []*lavasession.EndpointConnection{},
				}
				endpoints = append(endpoints, endpoint)
			}

			// Create provider session with static configuration
			// Static providers get 10x weight multiplier automatically (see CalcWeightsByStake)
			providerEntry := lavasession.NewConsumerSessionWithProvider(
				provider.Name,
				endpoints,
				999999999,               // High compute units for availability
				1,                       // Fixed epoch (smart router doesn't track blockchain epochs)
				StaticProviderDummyCoin, // Placeholder coin (value ignored, object required for type safety)
			)
			providerEntry.StaticProvider = true
			sessions[uint64(idx)] = providerEntry
		}
		return sessions
	}

	// Convert static providers to ConsumerSessionsWithProvider format
	providerSessions := convertProvidersToSessions(relevantStaticProviderList)

	// Convert backup providers to sessions (already filtered above during policy derivation)
	var backupProviderSessions map[uint64]*lavasession.ConsumerSessionsWithProvider
	if len(relevantBackupProviderList) > 0 {
		backupProviderSessions = convertProvidersToSessions(relevantBackupProviderList)
		utils.LavaFormatInfo("Configured backup providers for endpoint",
			utils.Attribute{Key: "chainID", Value: chainID},
			utils.Attribute{Key: "apiInterface", Value: rpcEndpoint.ApiInterface},
			utils.Attribute{Key: "backupCount", Value: len(backupProviderSessions)})
	}

	// Get current epoch for initial provider session setup
	currentEpoch := rpsr.epochTimer.GetCurrentEpoch()

	// Update PairingEpoch for all provider sessions to current epoch
	for _, providerSession := range providerSessions {
		providerSession.Lock.Lock()
		providerSession.PairingEpoch = currentEpoch
		providerSession.Lock.Unlock()
	}
	for _, backupSession := range backupProviderSessions {
		backupSession.Lock.Lock()
		backupSession.PairingEpoch = currentEpoch
		backupSession.Lock.Unlock()
	}

	// Update the session manager with static providers and backup providers
	err = sessionManager.UpdateAllProviders(currentEpoch, providerSessions, backupProviderSessions)
	if err != nil {
		errCh <- err
		return utils.LavaFormatError("failed updating static providers", err)
	}

	// Store provider sessions for epoch updates
	rpsr.providerSessions[sessionManagerKey] = providerSessions
	if len(backupProviderSessions) > 0 {
		rpsr.backupProviderSessions[sessionManagerKey] = backupProviderSessions
	}

	var relaysMonitor *metrics.RelaysMonitor
	if options.cmdFlags.RelaysHealthEnableFlag {
		relaysMonitor = metrics.NewRelaysMonitor(options.cmdFlags.RelaysHealthIntervalFlag, rpcEndpoint.ChainID, rpcEndpoint.ApiInterface)
		relaysMonitorAggregator.RegisterRelaysMonitor(rpcEndpoint.String(), relaysMonitor)
	}

	rpcSmartRouterServer := &RPCSmartRouterServer{}

	var wsSubscriptionManager *chainlib.ConsumerWSSubscriptionManager
	var specMethodType string
	if rpcEndpoint.ApiInterface == spectypes.APIInterfaceJsonRPC {
		specMethodType = http.MethodPost
	}
	wsSubscriptionManager = chainlib.NewConsumerWSSubscriptionManager(sessionManager, rpcSmartRouterServer, options.refererData, specMethodType, chainParser, activeSubscriptionProvidersStorage, smartRouterMetricsManager)

	utils.LavaFormatInfo("RPCSmartRouter Listening", utils.Attribute{Key: "endpoints", Value: rpcEndpoint.String()})
	// Convert smartRouterIdentifier string to empty sdk.AccAddress for smart router
	emptyConsumerAddr := []byte{}
	err = rpcSmartRouterServer.ServeRPCRequests(ctx, rpcEndpoint, chainParser, sessionManager, options.requiredResponses, options.privKey, options.lavaChainID, options.cache, rpcSmartRouterMetrics, emptyConsumerAddr, smartRouterConsistency, relaysMonitor, options.cmdFlags, options.stateShare, options.refererData, smartRouterReportsManager, wsSubscriptionManager)
	if err != nil {
		err = utils.LavaFormatError("failed serving rpc requests", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
		errCh <- err
		return err
	}
	return nil
}

func ParseEndpoints(viper_endpoints *viper.Viper, geolocation uint64) (endpoints []*lavasession.RPCEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(common.EndpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal endpoints", err, utils.Attribute{Key: "viper_endpoints", Value: viper_endpoints.AllSettings()})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation
		if endpoint.HealthCheckPath == "" {
			endpoint.HealthCheckPath = common.DEFAULT_HEALTH_PATH
		}
	}
	return endpoints, err
}

func CreateRPCSmartRouterCobraCommand() *cobra.Command {
	cmdRPCSmartRouter := &cobra.Command{
		Use:   "rpcsmartrouter [config-file] | { {listen-ip:listen-port spec-chain-id api-interface} ... }",
		Short: `rpcsmartrouter sets up a centralized server with static providers to perform api requests`,
		Long: `rpcsmartrouter sets up a centralized server with static and backup providers to perform api requests through the lava protocol.
		This is the smart router mode that uses pre-configured static providers instead of dynamically discovering providers on-chain.
		all configs should be located in the local running directory /config or ` + app.DefaultNodeHome + `
		if no arguments are passed, assumes default config file: ` + DefaultRPCSmartRouterFileName + `
		if one argument is passed, its assumed the config file name
		`,
		Example: `required flags: --geolocation 1 --static-providers ...
rpcsmartrouter <flags>
rpcsmartrouter rpcsmartrouter_conf <flags>
rpcsmartrouter 127.0.0.1:3333 OSMOSIS tendermintrpc 127.0.0.1:3334 OSMOSIS rest <flags>
rpcsmartrouter smartrouter_examples/full_smartrouter_example.yml --cache-be "127.0.0.1:7778" --geolocation 1 [--debug-relays] --log_level <debug|warn|...>`,
		Args: func(cmd *cobra.Command, args []string) error {
			// Optionally run one of the validators provided by cobra
			if err := cobra.RangeArgs(0, 1)(cmd, args); err == nil {
				// zero or one argument is allowed
				return nil
			}
			if len(args)%len(Yaml_config_properties) != 0 {
				return fmt.Errorf("invalid number of arguments, either its a single config file or repeated groups of 3 HOST:PORT chain-id api-interface")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo(common.ProcessStartLogText)
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			// set viper
			config_name := DefaultRPCSmartRouterFileName
			if len(args) == 1 {
				config_name = args[0] // name of config file (without extension)
			}
			viper.SetConfigName(config_name)
			viper.SetConfigType("yml")
			viper.AddConfigPath(".")
			viper.AddConfigPath("./config")
			viper.AddConfigPath(app.DefaultNodeHome)

			// set log format
			logFormat := viper.GetString(flags.FlagLogFormat)
			utils.JsonFormat = logFormat == "json"
			// set rolling log.
			closeLoggerOnFinish := common.SetupRollingLogger()
			defer closeLoggerOnFinish()

			utils.LavaFormatInfo("RPCConsumer started:", utils.Attribute{Key: "args", Value: strings.Join(args, ",")})

			// setting the insecure option on provider dial, this should be used in development only!
			lavasession.AllowInsecureConnectionToProviders = viper.GetBool(lavasession.AllowInsecureConnectionToProvidersFlag)
			if lavasession.AllowInsecureConnectionToProviders {
				utils.LavaFormatWarning("AllowInsecureConnectionToProviders is set to true, this should be used only in development", nil, utils.Attribute{Key: lavasession.AllowInsecureConnectionToProvidersFlag, Value: lavasession.AllowInsecureConnectionToProviders})
			}
			lavasession.AllowGRPCCompressionForConsumerProviderCommunication = viper.GetBool(lavasession.AllowGRPCCompressionFlag)
			if lavasession.AllowGRPCCompressionForConsumerProviderCommunication {
				utils.LavaFormatInfo("AllowGRPCCompressionForConsumerProviderCommunication is set to true, messages will be compressed", utils.Attribute{Key: lavasession.AllowGRPCCompressionFlag, Value: lavasession.AllowGRPCCompressionForConsumerProviderCommunication})
			}

			var rpcEndpoints []*lavasession.RPCEndpoint
			var viper_endpoints *viper.Viper
			if len(args) > 1 {
				viper_endpoints, err = common.ParseEndpointArgs(args, Yaml_config_properties, common.EndpointsConfigName)
				if err != nil {
					return utils.LavaFormatError("invalid endpoints arguments", err, utils.Attribute{Key: "endpoint_strings", Value: strings.Join(args, "")})
				}
				viper.MergeConfigMap(viper_endpoints.AllSettings())
				err := viper.SafeWriteConfigAs(DefaultRPCSmartRouterFileName)
				if err != nil {
					utils.LavaFormatInfo("did not create new config file, if it's desired remove the config file", utils.Attribute{Key: "file_name", Value: viper.ConfigFileUsed()})
				} else {
					utils.LavaFormatInfo("created new config file", utils.Attribute{Key: "file_name", Value: DefaultRPCSmartRouterFileName})
				}
			} else {
				err = viper.ReadInConfig()
				if err != nil {
					utils.LavaFormatFatal("could not load config file", err, utils.Attribute{Key: "expected_config_name", Value: viper.ConfigFileUsed()})
				}
				utils.LavaFormatInfo("read config file successfully", utils.Attribute{Key: "expected_config_name", Value: viper.ConfigFileUsed()})
			}
			geolocation, err := cmd.Flags().GetUint64(lavasession.GeolocationFlag)
			if err != nil {
				utils.LavaFormatFatal("failed to read geolocation flag, required flag", err)
			}
			rpcEndpoints, err = ParseEndpoints(viper.GetViper(), geolocation)
			if err != nil || len(rpcEndpoints) == 0 {
				return utils.LavaFormatError("invalid endpoints definition", err)
			}
			// handle flags, pass necessary fields
			ctx := context.Background()

			// Smart router doesn't need blockchain chain ID
			utils.LavaFormatInfo("Running Smart Router")

			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			test_mode, err := cmd.Flags().GetBool(common.TestModeFlagName)
			if err != nil {
				utils.LavaFormatFatal("failed to read test_mode flag", err)
			}
			ctx = context.WithValue(ctx, common.Test_mode_ctx_key{}, test_mode)
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
			// Note: VerifyAndHandleUnsupportedFlags is not called here because rpcsmartrouter
			// doesn't use blockchain transaction flags (--fees, --from, etc.) since it operates
			// in centralized mode without blockchain interactions

			// check if StaticProvidersConfigName exists in viper, if it does parse it with ParseStaticProviderEndpoints function
			var staticProviderEndpoints []*lavasession.RPCStaticProviderEndpoint
			if viper.IsSet(common.StaticProvidersConfigName) {
				staticProviderEndpoints, err = rpcprovider.ParseStaticProviderEndpoints(viper.GetViper(), common.StaticProvidersConfigName, geolocation)
				if err != nil {
					return utils.LavaFormatError("invalid static providers definition", err)
				}
				for _, endpoint := range staticProviderEndpoints {
					utils.LavaFormatInfo("Static Provider Endpoint:",
						utils.Attribute{Key: "Name", Value: endpoint.Name},
						utils.Attribute{Key: "Urls", Value: endpoint.NodeUrls},
						utils.Attribute{Key: "Chain ID", Value: endpoint.ChainID},
						utils.Attribute{Key: "API Interface", Value: endpoint.ApiInterface})
				}
			}

			// check if BackupProvidersConfigName exists in viper, if it does parse it with ParseStaticProviderEndpoints function
			var backupProviderEndpoints []*lavasession.RPCStaticProviderEndpoint
			if viper.IsSet(common.BackupProvidersConfigName) {
				utils.LavaFormatInfo("Backup Providers Config Name exists", utils.Attribute{Key: "Backup Providers Config Name", Value: common.BackupProvidersConfigName})
				backupProviderEndpoints, err = rpcprovider.ParseStaticProviderEndpoints(viper.GetViper(), common.BackupProvidersConfigName, geolocation)
				if err != nil {
					return utils.LavaFormatError("invalid backup providers definition", err)
				}
				for _, endpoint := range backupProviderEndpoints {
					utils.LavaFormatInfo("Backup Provider Endpoint:",
						utils.Attribute{Key: "Name", Value: endpoint.Name},
						utils.Attribute{Key: "Urls", Value: endpoint.NodeUrls},
						utils.Attribute{Key: "Chain ID", Value: endpoint.ChainID},
						utils.Attribute{Key: "API Interface", Value: endpoint.ApiInterface})
				}
			}

			if len(staticProviderEndpoints) == 0 {
				return utils.LavaFormatError(
					"smart router requires static providers configuration",
					nil,
					utils.Attribute{Key: "hint", Value: "add 'static-providers' section to config file"},
				)
			}

			// After parsing both endpoints and providers:
			for _, endpoint := range rpcEndpoints {
				hasProvider := false
				for _, provider := range staticProviderEndpoints {
					if provider.ChainID == endpoint.ChainID &&
						provider.ApiInterface == endpoint.ApiInterface {
						hasProvider = true
						break
					}
				}

				if !hasProvider {
					return utils.LavaFormatError(
						"no static providers configured for endpoint",
						nil,
						utils.Attribute{Key: "chainID", Value: endpoint.ChainID},
						utils.Attribute{Key: "apiInterface", Value: endpoint.ApiInterface},
						utils.Attribute{Key: "hint", Value: "add provider in 'static-providers' section"},
					)
				}
			}

			rpcSmartRouter := RPCSmartRouter{}
			requiredResponses := 1 // TODO: handle secure flag, for a majority between providers
			utils.LavaFormatInfo("lavap Binary Version: " + upgrade.GetCurrentVersion().ConsumerVersion)
			rand.InitRandomSeed()

			var cache *performance.Cache = nil
			cacheAddr, err := cmd.Flags().GetString(performance.CacheFlagName)
			if err != nil {
				utils.LavaFormatError("Failed To Get Cache Address flag", err, utils.Attribute{Key: "flags", Value: cmd.Flags()})
			} else if cacheAddr != "" {
				cache, err = performance.InitCache(ctx, cacheAddr)
				if err != nil {
					utils.LavaFormatError("Failed To Connect to cache at address", err, utils.Attribute{Key: "address", Value: cacheAddr})
				} else {
					utils.LavaFormatInfo("cache service connected", utils.Attribute{Key: "address", Value: cacheAddr})
				}
			}
			if strategyFlag.Strategy != provideroptimizer.StrategyBalanced {
				utils.LavaFormatInfo("Working with selection strategy: " + strategyFlag.String())
			}

			analyticsServerAddresses := AnalyticsServerAddresses{
				AddApiMethodCallsMetrics: viper.GetBool(metrics.AddApiMethodCallsMetrics),
				MetricsListenAddress:     viper.GetString(metrics.MetricsListenFlagName),
				RelayServerAddress:       viper.GetString(metrics.RelayServerFlagName),
				RelayKafkaAddress:        viper.GetString(metrics.RelayKafkaFlagName),
				RelayKafkaTopic:          viper.GetString(metrics.RelayKafkaTopicFlagName),
				RelayKafkaUsername:       viper.GetString(metrics.RelayKafkaUsernameFlagName),
				RelayKafkaPassword:       viper.GetString(metrics.RelayKafkaPasswordFlagName),
				RelayKafkaMechanism:      viper.GetString(metrics.RelayKafkaMechanismFlagName),
				RelayKafkaTLSEnabled:     viper.GetBool(metrics.RelayKafkaTLSEnabledFlagName),
				RelayKafkaTLSInsecure:    viper.GetBool(metrics.RelayKafkaTLSInsecureFlagName),
				ReportsAddressFlag:       viper.GetString(reportsSendBEAddress),
				OptimizerQoSAddress:      viper.GetString(common.OptimizerQosServerAddressFlag),
				OptimizerQoSListen:       viper.GetBool(common.OptimizerQosListenFlag),
			}

			var refererData *chainlib.RefererData
			if viper.GetString(refererBackendAddressFlagName) != "" || viper.GetString(refererMarkerFlagName) != "" {
				refererData = &chainlib.RefererData{
					Address: viper.GetString(refererBackendAddressFlagName), // address is used to send to a backend if necessary
					Marker:  viper.GetString(refererMarkerFlagName),         // marker is necessary to unwrap paths
				}
			}

			maxConcurrentProviders := viper.GetUint(common.MaximumConcurrentProvidersFlagName)

			// RPCSmartRouter always runs in standalone mode
			epochDuration := viper.GetDuration(common.EpochDurationFlag)
			if epochDuration == 0 {
				epochDuration = common.StandaloneEpochDuration // 15 minutes default for standalone
				utils.LavaFormatInfo("RPCSmartRouter: using default epoch duration for standalone mode",
					utils.LogAttr("epochDuration", epochDuration),
				)
			}

			consumerPropagatedFlags := common.ConsumerCmdFlags{
				HeadersFlag:              viper.GetString(common.CorsHeadersFlag),
				CredentialsFlag:          viper.GetString(common.CorsCredentialsFlag),
				OriginFlag:               viper.GetString(common.CorsOriginFlag),
				MethodsFlag:              viper.GetString(common.CorsMethodsFlag),
				CDNCacheDuration:         viper.GetString(common.CDNCacheDurationFlag),
				RelaysHealthEnableFlag:   viper.GetBool(common.RelaysHealthEnableFlag),
				RelaysHealthIntervalFlag: viper.GetDuration(common.RelayHealthIntervalFlag),
				DebugRelays:              viper.GetBool(DebugRelaysFlagName),
				StaticSpecPath:           viper.GetString(common.UseStaticSpecFlag),
				GitHubToken:              viper.GetString(common.GitHubTokenFlag),
				EpochDuration:            epochDuration,
			}

			// Get private key for signing relay requests
			// Smart router uses ephemeral key in static provider mode since providers
			// are configured to accept requests from anyone (no signature verification)
			keyName, err := sigs.GetKeyName(clientCtx)
			var privKey *btcec.PrivateKey
			if err != nil {
				// If no key in keyring, generate ephemeral key for static provider mode
				utils.LavaFormatWarning("No key found in keyring, generating ephemeral key for signing (static provider mode)", err)
				privKey, err = btcec.NewPrivateKey()
				if err != nil {
					utils.LavaFormatFatal("failed generating ephemeral private key", err)
				}
			} else {
				// Use key from keyring if available
				privKey, err = sigs.GetPrivKey(clientCtx, keyName)
				if err != nil {
					utils.LavaFormatWarning("Failed getting key from keyring, generating ephemeral key for signing (static provider mode)", err, utils.Attribute{Key: "keyName", Value: keyName})
					privKey, err = btcec.NewPrivateKey()
					if err != nil {
						utils.LavaFormatFatal("failed generating ephemeral private key", err)
					}
				}
			}

			// Get Lava chain ID
			lavaChainID := clientCtx.ChainID

			rpcSmartRouterSharedState := viper.GetBool(common.SharedStateFlag)
			err = rpcSmartRouter.Start(ctx, &rpcSmartRouterStartOptions{
				rpcEndpoints:             rpcEndpoints,
				requiredResponses:        requiredResponses,
				cache:                    cache,
				strategy:                 strategyFlag.Strategy,
				maxConcurrentProviders:   maxConcurrentProviders,
				analyticsServerAddresses: analyticsServerAddresses,
				cmdFlags:                 consumerPropagatedFlags,
				stateShare:               rpcSmartRouterSharedState,
				refererData:              refererData,
				staticProvidersList:      staticProviderEndpoints,
				backupProvidersList:      backupProviderEndpoints,
				geoLocation:              geolocation,
				clientCtx:                clientCtx,
				privKey:                  privKey,
				lavaChainID:              lavaChainID,
			})
			return err
		},
	}

	// RPCSmartRouter command flags - no blockchain flags needed
	cmdRPCSmartRouter.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCSmartRouter.Flags().Uint(common.MaximumConcurrentProvidersFlagName, 3, "max number of concurrent providers to communicate with")
	cmdRPCSmartRouter.MarkFlagRequired(common.GeolocationFlag)
	cmdRPCSmartRouter.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdRPCSmartRouter.Flags().Bool(lavasession.AllowInsecureConnectionToProvidersFlag, false, "allow insecure provider-dialing. used for development and testing")
	cmdRPCSmartRouter.Flags().Bool(lavasession.AllowGRPCCompressionFlag, false, "allow messages to be compressed when communicating between the consumer and provider")
	cmdRPCSmartRouter.Flags().Bool(common.TestModeFlagName, false, "test mode causes rpcconsumer to send dummy data and print all of the metadata in it's listeners")
	cmdRPCSmartRouter.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCSmartRouter.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdRPCSmartRouter.Flags().Var(&strategyFlag, "strategy", fmt.Sprintf("the strategy to use to pick providers (%s)", strings.Join(strategyNames, "|")))
	cmdRPCSmartRouter.Flags().String(metrics.MetricsListenFlagName, metrics.DisabledFlagOption, "the address to expose prometheus metrics (such as localhost:7779)")
	cmdRPCSmartRouter.Flags().Bool(metrics.AddApiMethodCallsMetrics, false, "adding a counter gauge for each method called per chain per api interface")
	cmdRPCSmartRouter.Flags().String(metrics.RelayServerFlagName, metrics.DisabledFlagOption, "the http address of the relay usage server api endpoint (example http://127.0.0.1:8080)")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaFlagName, metrics.DisabledFlagOption, "the kafka address for sending relay metrics (example localhost:9092)")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaTopicFlagName, "lava-relay-metrics", "the kafka topic for sending relay metrics")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaUsernameFlagName, "", "kafka username for SASL authentication")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaPasswordFlagName, "", "kafka password for SASL authentication")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaMechanismFlagName, "SCRAM-SHA-512", "kafka SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)")
	cmdRPCSmartRouter.Flags().Bool(metrics.RelayKafkaTLSEnabledFlagName, false, "enable TLS for kafka connections")
	cmdRPCSmartRouter.Flags().Bool(metrics.RelayKafkaTLSInsecureFlagName, false, "skip TLS certificate verification for kafka connections")
	cmdRPCSmartRouter.Flags().Bool(DebugRelaysFlagName, false, "adding debug information to relays")
	// CORS related flags
	cmdRPCSmartRouter.Flags().String(common.CorsCredentialsFlag, "true", "Set up CORS allowed credentials,default \"true\"")
	cmdRPCSmartRouter.Flags().String(common.CorsHeadersFlag, "", "Set up CORS allowed headers, * for all, default simple cors specification headers")
	cmdRPCSmartRouter.Flags().String(common.CorsOriginFlag, "*", "Set up CORS allowed origin, enabled * by default")
	cmdRPCSmartRouter.Flags().String(common.CorsMethodsFlag, "GET,POST,PUT,DELETE,OPTIONS", "set up Allowed OPTIONS methods, defaults to: \"GET,POST,PUT,DELETE,OPTIONS\"")
	cmdRPCSmartRouter.Flags().String(common.CDNCacheDurationFlag, "86400", "set up preflight options response cache duration, default 86400 (24h in seconds)")
	cmdRPCSmartRouter.Flags().Bool(common.SharedStateFlag, false, "Share the consumer consistency state with the cache service. this should be used with cache backend enabled if you want to state sync multiple rpc consumers")
	// relays health check related flags
	cmdRPCSmartRouter.Flags().Bool(common.RelaysHealthEnableFlag, RelaysHealthEnableFlagDefault, "enables relays health check")
	cmdRPCSmartRouter.Flags().Duration(common.RelayHealthIntervalFlag, RelayHealthIntervalFlagDefault, "interval between relay health checks")
	cmdRPCSmartRouter.Flags().String(refererBackendAddressFlagName, "", "address to send referer to")
	cmdRPCSmartRouter.Flags().String(refererMarkerFlagName, "lava-referer-", "the string marker to identify referer")
	cmdRPCSmartRouter.Flags().String(reportsSendBEAddress, "", "address to send reports to")
	cmdRPCSmartRouter.Flags().BoolVar(&lavasession.DebugProbes, DebugProbesFlagName, false, "adding information to probes")
	cmdRPCSmartRouter.Flags().String(common.UseStaticSpecFlag, "", "load offline spec provided path to spec file, used to test specs before they are proposed on chain")
	cmdRPCSmartRouter.Flags().String(common.GitHubTokenFlag, "", "GitHub personal access token for accessing private repositories and higher API rate limits (5,000 requests/hour vs 60 for unauthenticated)")
	cmdRPCSmartRouter.Flags().Duration(common.EpochDurationFlag, 0, "duration of each epoch for time-based epoch system (e.g., 30m, 1h). If not set, epochs are disabled")
	cmdRPCSmartRouter.Flags().IntVar(&relaycore.RelayCountOnNodeError, common.SetRelayCountOnNodeErrorFlag, 2, "set the number of retries attempt on node errors")
	// optimizer metrics
	cmdRPCSmartRouter.Flags().Float64Var(&provideroptimizer.ATierChance, common.SetProviderOptimizerBestTierPickChance, provideroptimizer.ATierChance, "set the chances for picking a provider from the best group, default is 75% -> 0.75")
	cmdRPCSmartRouter.Flags().Float64Var(&provideroptimizer.LastTierChance, common.SetProviderOptimizerWorstTierPickChance, provideroptimizer.LastTierChance, "set the chances for picking a provider from the worse group, default is 0% -> 0.0")
	cmdRPCSmartRouter.Flags().IntVar(&provideroptimizer.OptimizerNumTiers, common.SetProviderOptimizerNumberOfTiersToCreate, provideroptimizer.OptimizerNumTiers, "set the number of groups to create, default is 4")
	cmdRPCSmartRouter.Flags().IntVar(&provideroptimizer.MinimumEntries, common.SetProviderOptimizerNumberOfProvidersPerTier, provideroptimizer.MinimumEntries, "set the number of providers to have in each tier, default is 5")
	// optimizer qos reports
	cmdRPCSmartRouter.Flags().String(common.OptimizerQosServerAddressFlag, "", "address to send optimizer qos reports to")
	cmdRPCSmartRouter.Flags().Bool(common.OptimizerQosListenFlag, false, "enable listening for optimizer qos reports on metrics endpoint i.e GET -> localhost:7779/provider_optimizer_metrics")
	cmdRPCSmartRouter.Flags().DurationVar(&metrics.OptimizerQosServerPushInterval, common.OptimizerQosServerPushIntervalFlag, time.Minute*5, "interval to push optimizer qos reports")
	cmdRPCSmartRouter.Flags().DurationVar(&metrics.OptimizerQosServerSamplingInterval, common.OptimizerQosServerSamplingIntervalFlag, time.Second*1, "interval to sample optimizer qos reports")
	cmdRPCSmartRouter.Flags().BoolVar(&provideroptimizer.AutoAdjustTiers, common.SetProviderOptimizerAutoAdjustTiers, provideroptimizer.AutoAdjustTiers, "optimizer enable auto adjust tiers, this flag will fix the tiers based on the number of providers in the pairing, defaults to (false)")
	// metrics
	cmdRPCSmartRouter.Flags().BoolVar(&metrics.ShowProviderEndpointInMetrics, common.ShowProviderEndpointInMetricsFlagName, metrics.ShowProviderEndpointInMetrics, "show provider endpoint in consumer metrics")
	// websocket flags
	cmdRPCSmartRouter.Flags().IntVar(&chainlib.WebSocketRateLimit, common.RateLimitWebSocketFlag, chainlib.WebSocketRateLimit, "rate limit (per second) websocket requests per user connection, default is unlimited")
	cmdRPCSmartRouter.Flags().Int64Var(&chainlib.MaximumNumberOfParallelWebsocketConnectionsPerIp, common.LimitParallelWebsocketConnectionsPerIpFlag, chainlib.MaximumNumberOfParallelWebsocketConnectionsPerIp, "limit number of parallel connections to websocket, per ip, default is unlimited (0)")
	cmdRPCSmartRouter.Flags().Int64Var(&chainlib.MaxIdleTimeInSeconds, common.LimitWebsocketIdleTimeFlag, chainlib.MaxIdleTimeInSeconds, "limit the idle time in seconds for a websocket connection, default is 20 minutes ( 20 * 60 )")
	cmdRPCSmartRouter.Flags().DurationVar(&chainlib.WebSocketBanDuration, common.BanDurationForWebsocketRateLimitExceededFlag, chainlib.WebSocketBanDuration, "once websocket rate limit is reached, user will be banned Xfor a duration, default no ban")
	cmdRPCSmartRouter.Flags().BoolVar(&chainlib.SkipPolicyVerification, common.SkipPolicyVerificationFlag, chainlib.SkipPolicyVerification, "skip policy verifications, this flag will skip onchain policy verification and will use the static provider list")

	cmdRPCSmartRouter.Flags().BoolVar(&lavasession.PeriodicProbeProviders, common.PeriodicProbeProvidersFlagName, lavasession.PeriodicProbeProviders, "enable periodic probing of providers")
	cmdRPCSmartRouter.Flags().DurationVar(&lavasession.PeriodicProbeProvidersInterval, common.PeriodicProbeProvidersIntervalFlagName, lavasession.PeriodicProbeProvidersInterval, "interval for periodic probing of providers")

	common.AddRollingLogConfig(cmdRPCSmartRouter)
	return cmdRPCSmartRouter
}

func testModeWarn(desc string) {
	utils.LavaFormatWarning("------------------------------test mode --------------------------------\n\t\t\t"+
		desc+"\n\t\t\t"+
		"------------------------------test mode --------------------------------\n", nil)
}
