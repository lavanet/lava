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
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/performance"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/protocol/statetracker"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	scoreutils "github.com/lavanet/lava/v5/utils/score"
	epochstoragetypes "github.com/lavanet/lava/v5/types/epoch"
	planstypes "github.com/lavanet/lava/v5/types/plans"
	protocoltypes "github.com/lavanet/lava/v5/types/protocol"
	spectypes "github.com/lavanet/lava/v5/types/spec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DefaultRPCSmartRouterFileName = "rpcsmartrouter.yml"
	DebugRelaysFlagName           = "debug-relays"
	DebugProbesFlagName           = "debug-probes"
	reportsSendBEAddress          = "reports-be-address"

	// lavaAppName is the application name, previously app.Name.
	lavaAppName = "lava"
	// lavaDefaultNodeHome is the default home directory, previously lavaDefaultNodeHome (~/.lava).
	lavaDefaultNodeHome = "$HOME/." + lavaAppName
)

var (
	Yaml_config_properties         = []string{"network-address", "chain-id", "api-interface"}
	RelaysHealthEnableFlagDefault  = true
	RelayHealthIntervalFlagDefault = 5 * time.Minute

	// StaticProviderDummyStake is used for stake-based provider selection weighting.
	// For static providers that do NOT specify an explicit stake, we keep this at 0 so CalcWeightsByStake
	// can apply the legacy "static provider boost" behavior (see lavasession package).
	StaticProviderDummyStake = int64(0)
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
	MetricsListenAddress  string
	RelayServerAddress    string
	RelayKafkaAddress     string
	RelayKafkaTopic       string
	RelayKafkaUsername    string
	RelayKafkaPassword    string
	RelayKafkaMechanism   string
	RelayKafkaTLSEnabled  bool
	RelayKafkaTLSInsecure bool
	ReportsAddressFlag    string
	OptimizerQoSAddress   string
	OptimizerQoSListen    bool
}
type RPCSmartRouter struct {
	// Smart router doesn't need blockchain state tracking
	epochTimer             *common.EpochTimer
	mu                     sync.Mutex                                                      // protects the four maps below during parallel endpoint setup
	sessionManagers        map[string]*lavasession.ConsumerSessionManager                  // key: chainID-apiInterface
	providerSessions       map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider // key: chainID-apiInterface
	backupProviderSessions map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider // key: chainID-apiInterface

	// Server references for per-endpoint ChainTracker cleanup on epoch updates
	rpcServers map[string]*RPCSmartRouterServer // key: chainID-apiInterface
}

type rpcSmartRouterStartOptions struct {
	rpcEndpoints             []*lavasession.RPCEndpoint
	cache                    *performance.Cache
	strategy                 provideroptimizer.Strategy
	maxConcurrentProviders   uint
	analyticsServerAddresses AnalyticsServerAddresses
	cmdFlags                 common.ConsumerCmdFlags
	stateShare               bool
	staticProvidersList      []*lavasession.RPCStaticProviderEndpoint // define static providers as primary providers
	backupProvidersList      []*lavasession.RPCStaticProviderEndpoint // define backup providers as emergency fallback when no providers available
	geoLocation              uint64
	weightedSelectorConfig   provideroptimizer.WeightedSelectorConfig
}

// spawns a new RPCSmartRouter server with all its processes and internals ready for communications
func (rpsr *RPCSmartRouter) Start(ctx context.Context, options *rpcSmartRouterStartOptions) (err error) {
	if common.IsTestMode(ctx) {
		testModeWarn("RPCSmartRouter running tests")
	}

	// Initialize session managers and provider sessions maps for epoch timer callbacks
	rpsr.sessionManagers = make(map[string]*lavasession.ConsumerSessionManager)
	rpsr.providerSessions = make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider)
	rpsr.backupProviderSessions = make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider)
	rpsr.rpcServers = make(map[string]*RPCSmartRouterServer)

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
	// SmartRouterMetricsManager is the single metrics owner for the smart router.
	// It serves its own HTTP endpoint and implements ConsumerMetricsManagerInf so it
	// can be passed to RPCConsumerLogs, ConsumerSessionManager, etc., eliminating the
	// need for a ConsumerMetricsManager (and all its lava_consumer_* metrics).
	smartRouterMetricsManager := metrics.NewSmartRouterMetricsManager(metrics.SmartRouterMetricsManagerOptions{
		NetworkAddress:     options.analyticsServerAddresses.MetricsListenAddress,
		StartHTTPServer:    true,
		OptimizerQoSClient: smartRouterOptimizerQoSClient,
	})

	rpcSmartRouterMetrics, err := metrics.NewRPCConsumerLogs(smartRouterMetricsManager, smartRouterUsageServeManager, smartRouterKafkaClient, smartRouterOptimizerQoSClient)
	if err != nil {
		utils.LavaFormatFatal("failed creating RPCSmartRouter logs", err)
	}

	smartRouterMetricsManager.SetVersion(protocoltypes.DefaultVersion.ConsumerTarget)
	smartRouterMetricsManager.StartSelectionStatsUpdater(ctx, metrics.OptimizerQosServerSamplingInterval)

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

	errCh := make(chan error, parallelJobs)

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
	// Register ONE global epoch callback that updates ALL session managers
	// This prevents multiple UpdateAllProviders calls with the same epoch to the same session manager
	rpsr.epochTimer.RegisterCallback(rpsr.updateEpoch)

	// Log that epoch timer is configured for all session managers
	utils.LavaFormatInfo("RPCSmartRouter: Registered epoch timer callback for all session managers",
		utils.LogAttr("sessionManagerCount", len(rpsr.sessionManagers)),
	)

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
	smartRouterMetricsManager *metrics.SmartRouterMetricsManager,
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
	if len(options.cmdFlags.StaticSpecPaths) > 0 {
		// Load spec from static file/directory/URL sources
		err = statetracker.RegisterForSpecUpdatesOrSetStaticSpecsWithToken(ctx, chainParser, options.cmdFlags.StaticSpecPaths, *rpcEndpoint, nil, options.cmdFlags.GitHubToken, options.cmdFlags.GitLabToken)
		if err != nil {
			err = utils.LavaFormatError("failed loading static spec", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
			errCh <- err
			return err
		}
	} else {
		err = utils.LavaFormatError("no static spec paths configured; smart router requires --static-spec-paths to load chain specs", nil, utils.Attribute{Key: "chainID", Value: chainID})
		errCh <- err
		return err
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
	newOptimizer := provideroptimizer.NewProviderOptimizer(options.strategy, averageBlockTime, options.maxConcurrentProviders, smartRouterOptimizerQoSClient, chainID)
	newOptimizer.ConfigureWeightedSelector(options.weightedSelectorConfig)
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

	// Set callback to get Lava blockchain block height for RelaySession.Epoch
	// Smart router doesn't connect to blockchain, so calculate approximate block height from epoch
	// Epoch duration is 15 minutes (900 seconds), and Lava block time is ~15 seconds
	// So each epoch is approximately 60 blocks (900 / 15)
	sessionManager.SetLavaBlockHeightCallback(func() int64 {
		currentEpoch := rpsr.epochTimer.GetCurrentEpoch()
		// Approximate blocks per epoch: epochDuration / averageBlockTime
		blocksPerEpoch := int64(rpsr.epochTimer.GetEpochDuration().Seconds() / 15) // 15 second Lava block time
		return int64(currentEpoch) * blocksPerEpoch
	})

	// Store session manager in router for epoch timer callbacks
	sessionManagerKey := rpcEndpoint.Key() // chainID-apiInterface
	rpsr.mu.Lock()
	rpsr.sessionManagers[sessionManagerKey] = sessionManager
	rpsr.mu.Unlock()

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

				// Create DirectRPCConnection for smart router (direct mode)
				// Use default parallel connections for HTTP connection pooling
				// Pass ApiInterface for proper protocol detection (bare host:port → gRPC when interface is gRPC)
				directConn, err := lavasession.NewDirectRPCConnection(
					ctx,
					url,
					uint(lavasession.DefaultMaximumStreamsOverASingleConnection),
					provider.ApiInterface, // Used for protocol detection when URL has no scheme
				)
				if err != nil {
					utils.LavaFormatWarning("failed to create direct RPC connection", err,
						utils.LogAttr("url", url.Url),
						utils.LogAttr("provider", provider.Name),
					)
					continue
				}

				utils.LavaFormatInfo("created direct RPC connection",
					utils.LogAttr("url", url.Url),
					utils.LogAttr("protocol", directConn.GetProtocol()),
					utils.LogAttr("provider", provider.Name),
				)

				endpoint := &lavasession.Endpoint{
					NetworkAddress:    url.Url,
					Enabled:           true,
					Addons:            extensions,
					Extensions:        extensions,
					Connections:       nil,                                           // rpcconsumer only - not used in smart router
					DirectConnections: []lavasession.DirectRPCConnection{directConn}, // Smart router uses direct RPC
					Geolocation:       planstypes.Geolocation(provider.Geolocation),
				}
				endpoints = append(endpoints, endpoint)

				// Register endpoint with metrics manager for info metric visibility
				if smartRouterMetricsManager != nil {
					smartRouterMetricsManager.RegisterEndpoint(
						rpcEndpoint.ChainID,
						rpcEndpoint.ApiInterface,
						url.Url,       // raw URL — stored in endpoint_url label; used for URL->name resolution in ChainTracker callbacks
						provider.Name, // provider name — used as endpoint_id in all Prometheus metrics
					)
				}
			}

			// Create provider session with static configuration.
			// If stake is specified in the static provider config, use it (ulava).
			// Otherwise keep stake=0 so CalcWeightsByStake applies the legacy static-provider boost.
			stake := provider.Stake
			if stake < 0 {
				stake = 0
			}
			stakeAmount := StaticProviderDummyStake
			if stake > 0 {
				stakeAmount = stake
			}
			providerEntry := lavasession.NewConsumerSessionWithProvider(
				provider.Name,
				endpoints,
				999999999, // High compute units for availability
				1,         // Fixed epoch (smart router doesn't track blockchain epochs)
				stakeAmount,
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
	rpsr.mu.Lock()
	rpsr.providerSessions[sessionManagerKey] = providerSessions
	if len(backupProviderSessions) > 0 {
		rpsr.backupProviderSessions[sessionManagerKey] = backupProviderSessions
	}
	rpsr.mu.Unlock()

	var relaysMonitor *metrics.RelaysMonitor
	if options.cmdFlags.RelaysHealthEnableFlag {
		relaysMonitor = metrics.NewRelaysMonitor(options.cmdFlags.RelaysHealthIntervalFlag, rpcEndpoint.ChainID, rpcEndpoint.ApiInterface)
		relaysMonitorAggregator.RegisterRelaysMonitor(rpcEndpoint.String(), relaysMonitor)
	}

	rpcSmartRouterServer := &RPCSmartRouterServer{}

	// Create WebSocket subscription manager
	// Uses interface type to support both provider-based (ConsumerWSSubscriptionManager)
	// and direct RPC (DirectWSSubscriptionManager) implementations
	var wsSubscriptionManager chainlib.WSSubscriptionManager

	// Collect ALL WebSocket-capable endpoints from static providers for direct subscriptions
	// WebSocket URLs are identified by ws:// or wss:// prefix
	var wsEndpoints []*common.NodeUrl
	for _, provider := range relevantStaticProviderList {
		for i := range provider.NodeUrls {
			url := strings.ToLower(provider.NodeUrls[i].Url)
			if strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://") {
				wsEndpoints = append(wsEndpoints, &provider.NodeUrls[i])
				utils.LavaFormatInfo("Found WebSocket endpoint for direct subscriptions",
					utils.LogAttr("url", provider.NodeUrls[i].Url),
					utils.LogAttr("provider", provider.Name),
					utils.LogAttr("chainID", provider.ChainID),
				)
			}
		}
	}

	// Create DirectWSSubscriptionManager if WebSocket endpoints are available
	// Otherwise fall back to provider-based subscription manager
	if len(wsEndpoints) > 0 {
		directWSManager := NewDirectWSSubscriptionManager(
			smartRouterMetricsManager,
			spectypes.APIInterfaceJsonRPC, // WebSocket subscriptions use JSON-RPC
			rpcEndpoint.ChainID,
			rpcEndpoint.ApiInterface,
			wsEndpoints,
			optimizer, // Pass optimizer for endpoint selection
			nil,       // Use default WebSocket config (configurable via CLI flags later)
		)
		// Start background cleanup goroutine
		directWSManager.Start(ctx)
		wsSubscriptionManager = directWSManager
		utils.LavaFormatInfo("Using DirectWSSubscriptionManager for direct WebSocket subscriptions",
			utils.LogAttr("chainID", rpcEndpoint.ChainID),
			utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
			utils.LogAttr("wsEndpointCount", len(wsEndpoints)),
			utils.LogAttr("optimizerEnabled", optimizer != nil),
		)
	} else {
		// No WebSocket endpoints configured - use NoOp manager that returns clear errors
		// Smart router does NOT fall back to provider-based subscriptions (per implementation plan)
		// Provider-based subscriptions are only for rpcconsumer, not rpcsmartrouter
		wsSubscriptionManager = NewNoOpWSSubscriptionManager(rpcEndpoint.ChainID, rpcEndpoint.ApiInterface)
		utils.LavaFormatInfo("No WebSocket endpoints configured for direct subscriptions",
			utils.LogAttr("chainID", rpcEndpoint.ChainID),
			utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
			utils.LogAttr("hint", "Add ws:// or wss:// URLs to static-providers-list to enable subscriptions"),
		)
	}

	// Create gRPC streaming subscription manager for gRPC server-streaming methods
	// This supports Cosmos Event Streaming, Solana Geyser, and other gRPC streaming protocols
	var grpcEndpoints []*common.NodeUrl
	if rpcEndpoint.ApiInterface == spectypes.APIInterfaceGrpc {
		// Collect gRPC endpoints from static providers
		for _, provider := range relevantStaticProviderList {
			if provider.ApiInterface == spectypes.APIInterfaceGrpc {
				for i := range provider.NodeUrls {
					grpcEndpoints = append(grpcEndpoints, &provider.NodeUrls[i])
					utils.LavaFormatInfo("Found gRPC endpoint for streaming subscriptions",
						utils.LogAttr("url", provider.NodeUrls[i].Url),
						utils.LogAttr("provider", provider.Name),
						utils.LogAttr("chainID", provider.ChainID),
					)
				}
			}
		}
	}

	// Initialize DirectGRPCSubscriptionManager if gRPC endpoints are available
	if len(grpcEndpoints) > 0 {
		grpcSubManager := NewDirectGRPCSubscriptionManager(
			smartRouterMetricsManager, // Metrics manager for tracking
			rpcEndpoint.ChainID,
			rpcEndpoint.ApiInterface,
			grpcEndpoints,
			optimizer, // Pass optimizer for endpoint selection (same as WS manager)
			nil,       // Use default GRPCStreamingConfig
		)
		// Start background cleanup goroutine
		grpcSubManager.Start(ctx)
		rpcSmartRouterServer.grpcSubscriptionManager = grpcSubManager
		utils.LavaFormatInfo("Using DirectGRPCSubscriptionManager for gRPC streaming subscriptions",
			utils.LogAttr("chainID", rpcEndpoint.ChainID),
			utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
			utils.LogAttr("grpcEndpointCount", len(grpcEndpoints)),
			utils.LogAttr("optimizerEnabled", optimizer != nil),
		)
	}

	// ============================================================================
	// PHASE 1: Static Provider Validation
	// ============================================================================
	// Validate ALL static providers BEFORE creating chain tracker (matches provider behavior).
	// Only validates providers matching this endpoint's api-interface.
	// See: the provider's validation approach for reference.
	if len(relevantStaticProviderList) > 0 {
		utils.LavaFormatInfo("Validating static providers",
			utils.LogAttr("chain", rpcEndpoint.ChainID),
			utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
			utils.LogAttr("providerCount", len(relevantStaticProviderList)),
		)

		validatedCount := 0
		for _, staticProvider := range relevantStaticProviderList {
			// Skip providers with different api-interface (validated by their own endpoint)
			if staticProvider.ApiInterface != rpcEndpoint.ApiInterface {
				utils.LavaFormatDebug("Skipping provider - different api-interface",
					utils.LogAttr("provider", staticProvider.Name),
					utils.LogAttr("providerInterface", staticProvider.ApiInterface),
					utils.LogAttr("endpointInterface", rpcEndpoint.ApiInterface),
				)
				continue
			}
			validatedCount++

			// Prepare ALL URLs for validation together (matches provider behavior).
			// ChainRouter requires both with-addon and without-addon routes for addon URLs
			// (see chain_router.go:258 which appends "" to addons list).
			verificationNodeUrls := []common.NodeUrl{}
			for _, nodeUrl := range staticProvider.NodeUrls {
				verificationNodeUrls = append(verificationNodeUrls, nodeUrl)
				// For addon URLs, also add a non-addon copy for routing flexibility
				if len(nodeUrl.Addons) > 0 {
					noAddonUrl := nodeUrl
					noAddonUrl.Addons = []string{}
					verificationNodeUrls = append(verificationNodeUrls, noAddonUrl)
				}
			}

			verificationEndpoint := &lavasession.RPCProviderEndpoint{
				NetworkAddress: staticProvider.NetworkAddress,
				ChainID:        staticProvider.ChainID,
				ApiInterface:   staticProvider.ApiInterface,
				Geolocation:    staticProvider.Geolocation,
				NodeUrls:       verificationNodeUrls,
			}

			// Create chain router with all URLs for complete supportedMap (HTTP + WebSocket)
			parallelConnections := uint(lavasession.DefaultMaximumStreamsOverASingleConnection)
			verificationRouter, err := chainlib.GetChainRouter(ctx, parallelConnections, verificationEndpoint, chainParser)
			if err != nil {
				err = utils.LavaFormatError("[PANIC] failed creating chain router for verification", err,
					utils.LogAttr("chain", rpcEndpoint.ChainID),
					utils.LogAttr("provider", staticProvider.Name),
				)
				errCh <- err
				return err
			}

			// Create full ChainFetcher for verification (respects severity, skip-verifications)
			verificationFetcher := chainlib.NewChainFetcher(ctx, &chainlib.ChainFetcherOptions{
				ChainRouter: verificationRouter,
				ChainParser: chainParser,
				Endpoint:    verificationEndpoint,
				Cache:       nil,
			})

			utils.LavaFormatInfo("Validating static provider",
				utils.LogAttr("name", staticProvider.Name),
				utils.LogAttr("chain", rpcEndpoint.ChainID),
				utils.LogAttr("urlCount", len(staticProvider.NodeUrls)),
			)

			err = verificationFetcher.Validate(ctx)
			if err != nil {
				err = utils.LavaFormatError("[PANIC] static provider validation failed", err,
					utils.LogAttr("chain", rpcEndpoint.ChainID),
					utils.LogAttr("provider", staticProvider.Name),
				)
				errCh <- err
				return err
			}

			utils.LavaFormatInfo("Static provider validated successfully",
				utils.LogAttr("chain", rpcEndpoint.ChainID),
				utils.LogAttr("provider", staticProvider.Name),
			)
		}

		utils.LavaFormatInfo("All providers validated for api-interface",
			utils.LogAttr("chain", rpcEndpoint.ChainID),
			utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
			utils.LogAttr("validated", validatedCount),
			utils.LogAttr("total", len(relevantStaticProviderList)),
		)
	}

	// ============================================================================
	// PHASE 1B: Backup Provider Validation (non-fatal)
	// ============================================================================
	// Validate backup providers using the same logic as PHASE 1, but treat all
	// failures as non-fatal warnings. A broken backup should never block startup —
	// static providers must still serve. Operators are clearly notified at startup
	// so they can fix backup endpoints before they are actually needed in an emergency.
	// Providers that fail validation are excluded from the registered backup list.
	if len(relevantBackupProviderList) > 0 {
		utils.LavaFormatInfo("Validating backup providers (non-fatal)",
			utils.LogAttr("chain", rpcEndpoint.ChainID),
			utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
			utils.LogAttr("backupCount", len(relevantBackupProviderList)),
		)

		failedBackupNames := make(map[string]struct{})
		validatedBackups := 0
		for _, backupProvider := range relevantBackupProviderList {
			if backupProvider.ApiInterface != rpcEndpoint.ApiInterface {
				utils.LavaFormatDebug("Skipping backup provider - different api-interface",
					utils.LogAttr("provider", backupProvider.Name),
					utils.LogAttr("providerInterface", backupProvider.ApiInterface),
					utils.LogAttr("endpointInterface", rpcEndpoint.ApiInterface),
				)
				continue
			}
			validatedBackups++

			// Build verificationNodeUrls with addon expansion (identical to PHASE 1)
			verificationNodeUrls := []common.NodeUrl{}
			for _, nodeUrl := range backupProvider.NodeUrls {
				verificationNodeUrls = append(verificationNodeUrls, nodeUrl)
				if len(nodeUrl.Addons) > 0 {
					noAddonUrl := nodeUrl
					noAddonUrl.Addons = []string{}
					verificationNodeUrls = append(verificationNodeUrls, noAddonUrl)
				}
			}

			verificationEndpoint := &lavasession.RPCProviderEndpoint{
				NetworkAddress: backupProvider.NetworkAddress,
				ChainID:        backupProvider.ChainID,
				ApiInterface:   backupProvider.ApiInterface,
				Geolocation:    backupProvider.Geolocation,
				NodeUrls:       verificationNodeUrls,
			}

			parallelConnections := uint(lavasession.DefaultMaximumStreamsOverASingleConnection)
			verificationRouter, err := chainlib.GetChainRouter(ctx, parallelConnections, verificationEndpoint, chainParser)
			if err != nil {
				failedBackupNames[backupProvider.Name] = struct{}{}
				utils.LavaFormatWarning("backup provider: failed creating chain router — excluding from backup list", err,
					utils.LogAttr("chain", rpcEndpoint.ChainID),
					utils.LogAttr("provider", backupProvider.Name),
				)
				continue
			}

			verificationFetcher := chainlib.NewChainFetcher(ctx, &chainlib.ChainFetcherOptions{
				ChainRouter: verificationRouter,
				ChainParser: chainParser,
				Endpoint:    verificationEndpoint,
				Cache:       nil,
			})

			utils.LavaFormatInfo("Validating backup provider",
				utils.LogAttr("name", backupProvider.Name),
				utils.LogAttr("chain", rpcEndpoint.ChainID),
				utils.LogAttr("urlCount", len(backupProvider.NodeUrls)),
			)

			if err = verificationFetcher.Validate(ctx); err != nil {
				failedBackupNames[backupProvider.Name] = struct{}{}
				utils.LavaFormatWarning("backup provider validation failed — excluding from backup list", err,
					utils.LogAttr("chain", rpcEndpoint.ChainID),
					utils.LogAttr("provider", backupProvider.Name),
				)
				continue
			}

			utils.LavaFormatInfo("Backup provider validated successfully",
				utils.LogAttr("chain", rpcEndpoint.ChainID),
				utils.LogAttr("provider", backupProvider.Name),
			)
		}

		if len(failedBackupNames) > 0 {
			utils.LavaFormatWarning("ATTENTION: some backup providers failed validation and were excluded — they will not be used during emergency failover",
				nil,
				utils.LogAttr("chain", rpcEndpoint.ChainID),
				utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
				utils.LogAttr("failed", len(failedBackupNames)),
				utils.LogAttr("validated", validatedBackups),
			)

			// Rebuild backup sessions excluding failed providers, then re-register so the
			// session manager never holds references to endpoints that cannot be reached.
			// PublicLavaAddress in ConsumerSessionsWithProvider is set from provider.Name
			// (see NewConsumerSessionWithProvider in convertProvidersToSessions above).
			filteredBackupSessions := make(map[uint64]*lavasession.ConsumerSessionsWithProvider)
			for idx, session := range backupProviderSessions {
				if _, failed := failedBackupNames[session.PublicLavaAddress]; !failed {
					filteredBackupSessions[idx] = session
				}
			}

			if err = sessionManager.UpdateAllProviders(currentEpoch, providerSessions, filteredBackupSessions); err != nil {
				utils.LavaFormatWarning("failed to re-register filtered backup providers", err,
					utils.LogAttr("chain", rpcEndpoint.ChainID),
				)
			} else {
				rpsr.mu.Lock()
				rpsr.backupProviderSessions[sessionManagerKey] = filteredBackupSessions
				rpsr.mu.Unlock()
			}
		} else {
			utils.LavaFormatInfo("All backup providers validated",
				utils.LogAttr("chain", rpcEndpoint.ChainID),
				utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface),
				utils.LogAttr("validated", validatedBackups),
			)
		}
	}

	// ============================================================================
	// PHASE 2: Chain Tracker Setup
	// ============================================================================
	// Create ChainTracker for latest block tracking using first provider.
	// ChainTracker polls for latest block and maintains block history for sync verification.
	var chainTracker chaintracker.IChainTracker
	if len(relevantStaticProviderList) > 0 {
		firstProvider := relevantStaticProviderList[0]

		// Minimal endpoint for ChainTracker (no addons needed, only polls latest block)
		chainTrackerEndpoint := &lavasession.RPCProviderEndpoint{
			NetworkAddress: firstProvider.NetworkAddress,
			ChainID:        firstProvider.ChainID,
			ApiInterface:   firstProvider.ApiInterface,
			Geolocation:    firstProvider.Geolocation,
			NodeUrls: []common.NodeUrl{
				{
					Url:        firstProvider.NodeUrls[0].Url,
					AuthConfig: firstProvider.NodeUrls[0].AuthConfig,
					Addons:     []string{},
				},
			},
		}

		parallelConnections := uint(lavasession.DefaultMaximumStreamsOverASingleConnection)
		chainRouter, err := chainlib.GetChainRouter(ctx, parallelConnections, chainTrackerEndpoint, chainParser)
		if err != nil {
			utils.LavaFormatWarning("Failed to create chain router for chain tracker", err,
				utils.LogAttr("chain", rpcEndpoint.ChainID),
			)
		} else {
			// Full ChainFetcher for chain tracker (matches provider behavior)
			chainFetcher := chainlib.NewChainFetcher(ctx, &chainlib.ChainFetcherOptions{
				ChainRouter: chainRouter,
				ChainParser: chainParser,
				Endpoint:    chainTrackerEndpoint,
				Cache:       options.cache,
			})

			_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
			blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)

			chainTrackerConfig := chaintracker.ChainTrackerConfig{
				BlocksToSave:          blocksToSaveChainTracker,
				AverageBlockTime:      averageBlockTime,
				ServerBlockMemory:     chaintracker.ChainTrackerDefaultMemory + blocksToSaveChainTracker,
				ChainId:               rpcEndpoint.ChainID,
				ParseDirectiveEnabled: chainParser.ParseDirectiveEnabled(),
			}

			chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
			if err != nil {
				utils.LavaFormatWarning("Failed to create chain tracker, sync tracking disabled", err,
					utils.LogAttr("chain", rpcEndpoint.ChainID),
				)
				chainTracker = nil
			} else {
				go func() {
					err := chainTracker.StartAndServe(ctx)
					if err != nil {
						utils.LavaFormatError("Chain tracker failed", err,
							utils.LogAttr("chain", rpcEndpoint.ChainID),
						)
					}
				}()

				utils.LavaFormatInfo("Chain tracker started",
					utils.LogAttr("chain", rpcEndpoint.ChainID),
					utils.LogAttr("pollingInterval", averageBlockTime/time.Duration(chaintracker.MostFrequentPollingMultiplier)),
					utils.LogAttr("blocksToSave", blocksToSaveChainTracker),
				)
			}
		}
	}

	if chainTracker == nil {
		utils.LavaFormatInfo("Starting without chain tracker (sync tracking disabled)",
			utils.LogAttr("chain", rpcEndpoint.ChainID),
		)
	}

	utils.LavaFormatInfo("RPCSmartRouter Listening", utils.Attribute{Key: "endpoints", Value: rpcEndpoint.String()})
	// Convert smartRouterIdentifier string to empty sdk.AccAddress for smart router
	err = rpcSmartRouterServer.ServeRPCRequests(ctx, rpcEndpoint, chainParser, chainTracker, sessionManager, options.cache, rpcSmartRouterMetrics, smartRouterConsistency, relaysMonitor, options.cmdFlags, options.stateShare, wsSubscriptionManager, smartRouterMetricsManager)
	if err != nil {
		err = utils.LavaFormatError("failed serving rpc requests", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
		errCh <- err
		return err
	}

	// Store server reference for per-endpoint ChainTracker cleanup on epoch updates
	rpsr.mu.Lock()
	rpsr.rpcServers[sessionManagerKey] = rpcSmartRouterServer
	rpsr.mu.Unlock()

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
		all configs should be located in the local running directory /config or ` + lavaDefaultNodeHome + `
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
			var err error
			// set viper
			config_name := DefaultRPCSmartRouterFileName
			if len(args) == 1 {
				config_name = args[0] // name of config file (without extension)
			}
			viper.SetConfigName(config_name)
			viper.SetConfigType("yml")
			viper.AddConfigPath(".")
			viper.AddConfigPath("./config")
			viper.AddConfigPath(lavaDefaultNodeHome)

			// set log format
			logFormat := viper.GetString("log-format")
			utils.JsonFormat = logFormat == "json"
			// set rolling log.
			closeLoggerOnFinish := common.SetupRollingLogger()
			defer closeLoggerOnFinish()

			utils.LavaFormatInfo("RPCSmartRouter started:", utils.Attribute{Key: "args", Value: strings.Join(args, ",")})

			// setting the insecure option on provider dial, this should be used in development only!
			lavasession.AllowInsecureConnectionToProviders = viper.GetBool(lavasession.AllowInsecureConnectionToProvidersFlag)
			if lavasession.AllowInsecureConnectionToProviders {
				utils.LavaFormatWarning("AllowInsecureConnectionToProviders is set to true, this should be used only in development", nil, utils.Attribute{Key: lavasession.AllowInsecureConnectionToProvidersFlag, Value: lavasession.AllowInsecureConnectionToProviders})
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
			} else if err = viper.ReadInConfig(); err != nil {
				utils.LavaFormatFatal("could not load config file", err, utils.Attribute{Key: "expected_config_name", Value: viper.ConfigFileUsed()})
			} else {
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

			logLevel, err := cmd.Flags().GetString("log-level")
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
			// check if the command includes --pyroscope-address
			pyroscopeAddressFlagUsed := cmd.Flags().Lookup(performance.PyroscopeAddressFlagName).Changed
			if pyroscopeAddressFlagUsed {
				pyroscopeServerAddress, err := cmd.Flags().GetString(performance.PyroscopeAddressFlagName)
				if err != nil {
					utils.LavaFormatFatal("failed to read pyroscope address flag", err)
				}
				pyroscopeAppName, err := cmd.Flags().GetString(performance.PyroscopeAppNameFlagName)
				if err != nil || pyroscopeAppName == "" {
					pyroscopeAppName = "lavap-smartrouter"
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

			// Parse direct RPC endpoints (new key: "direct-rpc", backward compat: "static-providers")
			var directRPCEndpoints []*lavasession.RPCStaticProviderEndpoint
			directRPCConfigKey := common.DirectRPCConfigName
			if !viper.IsSet(directRPCConfigKey) {
				directRPCConfigKey = common.StaticProvidersConfigName // backward compat
			}
			if viper.IsSet(directRPCConfigKey) {
				directRPCEndpoints, err = ParseStaticProviderEndpoints(viper.GetViper(), directRPCConfigKey, geolocation)
				if err != nil {
					return utils.LavaFormatError("invalid direct-rpc endpoints definition", err)
				}
				for _, endpoint := range directRPCEndpoints {
					utils.LavaFormatInfo("Direct RPC Endpoint:",
						utils.Attribute{Key: "Name", Value: endpoint.Name},
						utils.Attribute{Key: "Stake", Value: endpoint.Stake},
						utils.Attribute{Key: "Urls", Value: endpoint.NodeUrls},
						utils.Attribute{Key: "Chain ID", Value: endpoint.ChainID},
						utils.Attribute{Key: "API Interface", Value: endpoint.ApiInterface})
				}
			}

			// Parse backup direct RPC endpoints (new key: "backup-direct-rpc", backward compat: "backup-providers")
			var backupDirectRPCEndpoints []*lavasession.RPCStaticProviderEndpoint
			backupConfigKey := common.BackupDirectRPCConfigName
			if !viper.IsSet(backupConfigKey) {
				backupConfigKey = common.BackupProvidersConfigName // backward compat
			}
			if viper.IsSet(backupConfigKey) {
				utils.LavaFormatInfo("Backup direct-rpc config found", utils.Attribute{Key: "configKey", Value: backupConfigKey})
				backupDirectRPCEndpoints, err = ParseStaticProviderEndpoints(viper.GetViper(), backupConfigKey, geolocation)
				if err != nil {
					return utils.LavaFormatError("invalid backup-direct-rpc endpoints definition", err)
				}
				for _, endpoint := range backupDirectRPCEndpoints {
					utils.LavaFormatInfo("Backup Direct RPC Endpoint:",
						utils.Attribute{Key: "Name", Value: endpoint.Name},
						utils.Attribute{Key: "Urls", Value: endpoint.NodeUrls},
						utils.Attribute{Key: "Chain ID", Value: endpoint.ChainID},
						utils.Attribute{Key: "API Interface", Value: endpoint.ApiInterface})
				}
			}

			if len(directRPCEndpoints) == 0 {
				return utils.LavaFormatError(
					"smart router requires direct-rpc endpoints configuration",
					nil,
					utils.Attribute{Key: "hint", Value: "add 'direct-rpc' section to config file"},
				)
			}

			for _, endpoint := range rpcEndpoints {
				hasDirectRPC := false
				for _, directEndpoint := range directRPCEndpoints {
					if directEndpoint.ChainID == endpoint.ChainID &&
						directEndpoint.ApiInterface == endpoint.ApiInterface {
						hasDirectRPC = true
						break
					}
				}

				if !hasDirectRPC {
					return utils.LavaFormatError(
						"no direct-rpc endpoints configured for listener",
						nil,
						utils.Attribute{Key: "chainID", Value: endpoint.ChainID},
						utils.Attribute{Key: "apiInterface", Value: endpoint.ApiInterface},
						utils.Attribute{Key: "hint", Value: "add endpoint in 'direct-rpc' section"},
					)
				}
			}

			rpcSmartRouter := RPCSmartRouter{}
			utils.LavaFormatInfo("lavap Binary Version: " + protocoltypes.DefaultVersion.ConsumerTarget)
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
				MetricsListenAddress:  viper.GetString(metrics.MetricsListenFlagName),
				RelayServerAddress:    viper.GetString(metrics.RelayServerFlagName),
				RelayKafkaAddress:     viper.GetString(metrics.RelayKafkaFlagName),
				RelayKafkaTopic:       viper.GetString(metrics.RelayKafkaTopicFlagName),
				RelayKafkaUsername:    viper.GetString(metrics.RelayKafkaUsernameFlagName),
				RelayKafkaPassword:    viper.GetString(metrics.RelayKafkaPasswordFlagName),
				RelayKafkaMechanism:   viper.GetString(metrics.RelayKafkaMechanismFlagName),
				RelayKafkaTLSEnabled:  viper.GetBool(metrics.RelayKafkaTLSEnabledFlagName),
				RelayKafkaTLSInsecure: viper.GetBool(metrics.RelayKafkaTLSInsecureFlagName),
				ReportsAddressFlag:    viper.GetString(reportsSendBEAddress),
				OptimizerQoSAddress:   viper.GetString(common.OptimizerQosServerAddressFlag),
				OptimizerQoSListen:    viper.GetBool(common.OptimizerQosListenFlag),
			}

			maxConcurrentProviders := viper.GetUint(common.MaximumConcurrentProvidersFlagName)
			if err := scoreutils.SetProbeUpdateWeight(viper.GetFloat64(common.ProbeUpdateWeightFlagName)); err != nil {
				return err
			}
			weightedSelectorConfig := provideroptimizer.DefaultWeightedSelectorConfig()
			weightedSelectorConfig.AvailabilityWeight = viper.GetFloat64(common.ProviderOptimizerAvailabilityWeight)
			weightedSelectorConfig.LatencyWeight = viper.GetFloat64(common.ProviderOptimizerLatencyWeight)
			weightedSelectorConfig.SyncWeight = viper.GetFloat64(common.ProviderOptimizerSyncWeight)
			weightedSelectorConfig.StakeWeight = viper.GetFloat64(common.ProviderOptimizerStakeWeight)
			weightedSelectorConfig.MinSelectionChance = viper.GetFloat64(common.ProviderOptimizerMinSelectionChance)
			weightedSelectorConfig.Strategy = strategyFlag.Strategy

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
				StaticSpecPaths:          viper.GetStringSlice(common.UseStaticSpecFlag),
				GitHubToken:              viper.GetString(common.GitHubTokenFlag),
				GitLabToken:              viper.GetString(common.GitLabTokenFlag),
				EpochDuration:            epochDuration,
				EnableSelectionStats:     viper.GetBool(common.EnableSelectionStatsHeaderFlag),
			}

			rpcSmartRouterSharedState := viper.GetBool(common.SharedStateFlag)
			err = rpcSmartRouter.Start(ctx, &rpcSmartRouterStartOptions{
				rpcEndpoints:             rpcEndpoints,
				cache:                    cache,
				strategy:                 strategyFlag.Strategy,
				maxConcurrentProviders:   maxConcurrentProviders,
				analyticsServerAddresses: analyticsServerAddresses,
				cmdFlags:                 consumerPropagatedFlags,
				stateShare:               rpcSmartRouterSharedState,
				staticProvidersList:      directRPCEndpoints,
				backupProvidersList:      backupDirectRPCEndpoints,
				geoLocation:              geolocation,
				weightedSelectorConfig:   weightedSelectorConfig,
			})

			return err
		},
	}

	// RPCSmartRouter command flags - no blockchain flags needed
	cmdRPCSmartRouter.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCSmartRouter.Flags().Uint(common.MaximumConcurrentProvidersFlagName, 3, "max number of concurrent providers to communicate with")
	cmdRPCSmartRouter.MarkFlagRequired(common.GeolocationFlag)
	cmdRPCSmartRouter.Flags().Bool(lavasession.AllowInsecureConnectionToProvidersFlag, false, "allow insecure provider-dialing. used for development and testing")
	cmdRPCSmartRouter.Flags().Uint64Var(&lavasession.MaximumStreamsOverASingleConnection, lavasession.MaximumStreamsOverASingleConnectionFlag, lavasession.DefaultMaximumStreamsOverASingleConnection, "maximum number of parallel streams over a single provider connection")
	cmdRPCSmartRouter.Flags().Bool(common.TestModeFlagName, false, "test mode causes rpcconsumer to send dummy data and print all of the metadata in it's listeners")
	cmdRPCSmartRouter.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCSmartRouter.Flags().String(performance.PyroscopeAddressFlagName, "", "pyroscope server address for continuous profiling (e.g., http://pyroscope:4040)")
	cmdRPCSmartRouter.Flags().String(performance.PyroscopeAppNameFlagName, "lavap-smartrouter", "pyroscope application name for identifying this service")
	cmdRPCSmartRouter.Flags().Int(performance.PyroscopeMutexProfileFractionFlagName, performance.DefaultMutexProfileFraction, "mutex profile sampling rate (1 in N mutex events)")
	cmdRPCSmartRouter.Flags().Int(performance.PyroscopeBlockProfileRateFlagName, performance.DefaultBlockProfileRate, "block profile rate in nanoseconds (1 records all blocking events)")
	cmdRPCSmartRouter.Flags().String(performance.PyroscopeTagsFlagName, "", "comma-separated list of tags in key=value format (e.g., instance=router-1,region=us-east)")
	cmdRPCSmartRouter.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdRPCSmartRouter.Flags().Var(&strategyFlag, "strategy", fmt.Sprintf("the strategy to use to pick providers (%s)", strings.Join(strategyNames, "|")))
	defaultWeightedConfig := provideroptimizer.DefaultWeightedSelectorConfig()
	cmdRPCSmartRouter.Flags().Float64(common.ProviderOptimizerAvailabilityWeight, defaultWeightedConfig.AvailabilityWeight, "weight assigned to provider availability when computing selection scores")
	cmdRPCSmartRouter.Flags().Float64(common.ProviderOptimizerLatencyWeight, defaultWeightedConfig.LatencyWeight, "weight assigned to provider latency when computing selection scores")
	cmdRPCSmartRouter.Flags().Float64(common.ProviderOptimizerSyncWeight, defaultWeightedConfig.SyncWeight, "weight assigned to provider sync freshness when computing selection scores")
	cmdRPCSmartRouter.Flags().Float64(common.ProviderOptimizerStakeWeight, defaultWeightedConfig.StakeWeight, "weight assigned to provider stake when computing selection scores")
	cmdRPCSmartRouter.Flags().Float64(common.ProviderOptimizerMinSelectionChance, defaultWeightedConfig.MinSelectionChance, "minimum selection probability for any provider regardless of score")
	if err := viper.BindPFlag(common.ProviderOptimizerAvailabilityWeight, cmdRPCSmartRouter.Flags().Lookup(common.ProviderOptimizerAvailabilityWeight)); err != nil {
		utils.LavaFormatFatal("failed binding availability weight flag", err)
	}
	if err := viper.BindPFlag(common.ProviderOptimizerLatencyWeight, cmdRPCSmartRouter.Flags().Lookup(common.ProviderOptimizerLatencyWeight)); err != nil {
		utils.LavaFormatFatal("failed binding latency weight flag", err)
	}
	if err := viper.BindPFlag(common.ProviderOptimizerSyncWeight, cmdRPCSmartRouter.Flags().Lookup(common.ProviderOptimizerSyncWeight)); err != nil {
		utils.LavaFormatFatal("failed binding sync weight flag", err)
	}
	if err := viper.BindPFlag(common.ProviderOptimizerStakeWeight, cmdRPCSmartRouter.Flags().Lookup(common.ProviderOptimizerStakeWeight)); err != nil {
		utils.LavaFormatFatal("failed binding stake weight flag", err)
	}
	if err := viper.BindPFlag(common.ProviderOptimizerMinSelectionChance, cmdRPCSmartRouter.Flags().Lookup(common.ProviderOptimizerMinSelectionChance)); err != nil {
		utils.LavaFormatFatal("failed binding min selection chance flag", err)
	}
	cmdRPCSmartRouter.Flags().String(metrics.MetricsListenFlagName, metrics.DisabledFlagOption, "the address to expose prometheus metrics (such as localhost:7779)")
	cmdRPCSmartRouter.Flags().String(metrics.RelayServerFlagName, metrics.DisabledFlagOption, "the http address of the relay usage server api endpoint (example http://127.0.0.1:8080)")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaFlagName, metrics.DisabledFlagOption, "the kafka address for sending relay metrics (example localhost:9092)")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaTopicFlagName, "lava-relay-metrics", "the kafka topic for sending relay metrics")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaUsernameFlagName, "", "kafka username for SASL authentication")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaPasswordFlagName, "", "kafka password for SASL authentication")
	cmdRPCSmartRouter.Flags().String(metrics.RelayKafkaMechanismFlagName, "SCRAM-SHA-512", "kafka SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)")
	cmdRPCSmartRouter.Flags().Bool(metrics.RelayKafkaTLSEnabledFlagName, false, "enable TLS for kafka connections")
	cmdRPCSmartRouter.Flags().Bool(metrics.RelayKafkaTLSInsecureFlagName, false, "skip TLS certificate verification for kafka connections")
	cmdRPCSmartRouter.Flags().Bool(DebugRelaysFlagName, false, "adding debug information to relays")
	cmdRPCSmartRouter.Flags().Bool(common.EnableSelectionStatsHeaderFlag, false, "enable selection stats header for debugging provider selection")
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
	cmdRPCSmartRouter.Flags().String(reportsSendBEAddress, "", "address to send reports to")
	cmdRPCSmartRouter.Flags().BoolVar(&lavasession.DebugProbes, DebugProbesFlagName, false, "adding information to probes")
	cmdRPCSmartRouter.Flags().StringArray(common.UseStaticSpecFlag, nil, "load specs from file, directory, or remote URL (GitHub/GitLab). Can be specified multiple times; later sources override earlier ones for same chain ID")
	cmdRPCSmartRouter.Flags().String(common.GitHubTokenFlag, "", "GitHub personal access token for accessing private repositories and higher API rate limits (5,000 requests/hour vs 60 for unauthenticated)")
	cmdRPCSmartRouter.Flags().String(common.GitLabTokenFlag, "", "GitLab personal access token for accessing private repositories (supports gitlab.com and self-hosted instances)")
	cmdRPCSmartRouter.Flags().Duration(common.EpochDurationFlag, 0, "duration of each epoch for time-based epoch system (e.g., 30m, 1h). If not set, epochs are disabled")
	cmdRPCSmartRouter.Flags().IntVar(&relaycore.RelayRetryLimit, common.SetRelayRetryLimitFlag, 2, "max total relay retry attempts across all error types (node and protocol errors combined; 0 disables retries)")
	cmdRPCSmartRouter.Flags().BoolVar(&rpcInterfaceMessages.BatchNodeErrorOnAny, common.BatchNodeErrorOnAnyFlag, false, "if true, batch requests are treated as node errors if ANY sub-request fails; if false (default), only if ALL fail")
	// optimizer qos reports
	cmdRPCSmartRouter.Flags().String(common.OptimizerQosServerAddressFlag, "", "address to send optimizer qos reports to")
	cmdRPCSmartRouter.Flags().Bool(common.OptimizerQosListenFlag, false, "enable listening for optimizer qos reports on metrics endpoint i.e GET -> localhost:7779/provider_optimizer_metrics")
	cmdRPCSmartRouter.Flags().DurationVar(&metrics.OptimizerQosServerPushInterval, common.OptimizerQosServerPushIntervalFlag, time.Minute*5, "interval to push optimizer qos reports")
	cmdRPCSmartRouter.Flags().DurationVar(&metrics.OptimizerQosServerSamplingInterval, common.OptimizerQosServerSamplingIntervalFlag, time.Second*1, "interval to sample optimizer qos reports")
	// metrics
	cmdRPCSmartRouter.Flags().BoolVar(&metrics.ShowProviderEndpointInMetrics, common.ShowProviderEndpointInMetricsFlagName, metrics.ShowProviderEndpointInMetrics, "show provider endpoint in consumer metrics")
	// websocket flags
	cmdRPCSmartRouter.Flags().IntVar(&chainlib.WebSocketRateLimit, common.RateLimitWebSocketFlag, chainlib.WebSocketRateLimit, "rate limit (per second) websocket requests per user connection, default is unlimited")
	cmdRPCSmartRouter.Flags().Int64Var(&chainlib.MaximumNumberOfParallelWebsocketConnectionsPerIp, common.LimitParallelWebsocketConnectionsPerIpFlag, chainlib.MaximumNumberOfParallelWebsocketConnectionsPerIp, "limit number of parallel connections to websocket, per ip, default is unlimited (0)")
	cmdRPCSmartRouter.Flags().Int64Var(&chainlib.MaxIdleTimeInSeconds, common.LimitWebsocketIdleTimeFlag, chainlib.MaxIdleTimeInSeconds, "limit the idle time in seconds for a websocket connection, default is 20 minutes ( 20 * 60 )")
	cmdRPCSmartRouter.Flags().DurationVar(&chainlib.WebSocketBanDuration, common.BanDurationForWebsocketRateLimitExceededFlag, chainlib.WebSocketBanDuration, "once websocket rate limit is reached, user will be banned Xfor a duration, default no ban")

	cmdRPCSmartRouter.Flags().BoolVar(&chainlib.SkipWebsocketVerification, common.SkipWebsocketVerificationFlag, chainlib.SkipWebsocketVerification, "skip websocket verification for chains that require ws/wss endpoints")

	cmdRPCSmartRouter.Flags().BoolVar(&lavasession.PeriodicProbeProviders, common.PeriodicProbeProvidersFlagName, lavasession.PeriodicProbeProviders, "enable periodic probing of providers")
	cmdRPCSmartRouter.Flags().DurationVar(&lavasession.PeriodicProbeProvidersInterval, common.PeriodicProbeProvidersIntervalFlagName, lavasession.PeriodicProbeProvidersInterval, "interval for periodic probing of providers")
	cmdRPCSmartRouter.Flags().Float64(common.ProbeUpdateWeightFlagName, scoreutils.DefaultProbeUpdateWeight, "weight multiplier for provider-optimizer probe updates (liveness/latency); must be > 0")
	if err := viper.BindPFlag(common.ProbeUpdateWeightFlagName, cmdRPCSmartRouter.Flags().Lookup(common.ProbeUpdateWeightFlagName)); err != nil {
		utils.LavaFormatFatal("failed binding probe update weight flag", err)
	}

	cmdRPCSmartRouter.Flags().DurationVar(&common.DefaultTimeout, common.DefaultProcessingTimeoutFlagName, common.DefaultTimeout, "default timeout for relay processing (e.g., 30s, 1m)")
	cmdRPCSmartRouter.Flags().IntVar(&lavasession.MaxSessionsAllowedPerProvider, common.MaxSessionsPerProviderFlagName, lavasession.MaxSessionsAllowedPerProvider, "max number of sessions allowed per provider")

	// batch request size limit
	cmdRPCSmartRouter.Flags().IntVar(&chainlib.MaxBatchRequestSize, common.MaxBatchRequestSizeFlag, common.DefaultMaxBatchRequestSize, "max number of requests allowed within a batch request, 0 means unlimited")
	cmdRPCSmartRouter.Flags().BoolVar(&relaycore.DisableBatchRequestRetry, common.DisableBatchRequestRetryFlag, true, "disable retries for batch requests (JSON-RPC batches)")

	common.AddRollingLogConfig(cmdRPCSmartRouter)
	// Log level/format flags (previously provided by cosmos-sdk AddTxFlagsToCmd)
	cmdRPCSmartRouter.Flags().String("log-level", "info", "log level (debug|info|warn|error|fatal)")
	cmdRPCSmartRouter.Flags().String("log-format", "text", "log format (text|json)")
	return cmdRPCSmartRouter
}

func (rpsr *RPCSmartRouter) updateEpoch(epoch uint64) {
	// Update all session managers for this epoch
	for chainKey, sm := range rpsr.sessionManagers {
		sessionManager := sm
		chainKeyLog := chainKey
		oldProviderSessions := rpsr.providerSessions[chainKey]
		oldBackupSessions := rpsr.backupProviderSessions[chainKey]

		utils.LavaFormatInfo("ConsumerSessionManager: Epoch update triggered",
			utils.LogAttr("epoch", epoch),
			utils.LogAttr("chainKey", chainKeyLog),
			utils.LogAttr("time", time.Now().Format("15:04:05 MST")),
		)

		// Create FRESH ConsumerSessionsWithProvider objects to avoid session accumulation
		// This is critical: reusing the same objects causes sessions to accumulate in the Sessions map
		// until hitting the 1000-session limit, causing "No pairings available" errors
		freshProviderSessions := make(map[uint64]*lavasession.ConsumerSessionsWithProvider)
		for idx, oldSession := range oldProviderSessions {
			// Create new session with same configuration but fresh Sessions map
			freshSession := lavasession.NewConsumerSessionWithProvider(
				oldSession.PublicLavaAddress,
				oldSession.Endpoints, // Endpoints are safe to reuse
				oldSession.MaxComputeUnits,
				epoch, // New epoch
				oldSession.GetProviderStakeSize(),
			)
			freshSession.StaticProvider = oldSession.StaticProvider
			freshProviderSessions[idx] = freshSession

			utils.LavaFormatDebug("Created fresh provider session for epoch",
				utils.LogAttr("provider", freshSession.PublicLavaAddress),
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("chainKey", chainKeyLog))
		}

		// Create fresh backup sessions
		freshBackupSessions := make(map[uint64]*lavasession.ConsumerSessionsWithProvider)
		for idx, oldSession := range oldBackupSessions {
			freshSession := lavasession.NewConsumerSessionWithProvider(
				oldSession.PublicLavaAddress,
				oldSession.Endpoints,
				oldSession.MaxComputeUnits,
				epoch,
				oldSession.GetProviderStakeSize(),
			)
			freshSession.StaticProvider = oldSession.StaticProvider
			freshBackupSessions[idx] = freshSession

			utils.LavaFormatDebug("Created fresh backup provider session for epoch",
				utils.LogAttr("provider", freshSession.PublicLavaAddress),
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("chainKey", chainKeyLog))
		}

		// Update stored sessions with fresh objects
		rpsr.providerSessions[chainKey] = freshProviderSessions
		if len(freshBackupSessions) > 0 {
			rpsr.backupProviderSessions[chainKey] = freshBackupSessions
		}

		// Update session manager with fresh provider sessions
		// This triggers cleanup and provider unblocking while preventing session accumulation
		err := sessionManager.UpdateAllProviders(epoch, freshProviderSessions, freshBackupSessions)
		if err != nil {
			utils.LavaFormatError("Failed to update providers on epoch change", err,
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("chainKey", chainKeyLog),
			)
		}

		// Cleanup stale ChainTrackers for endpoints no longer in the provider sessions
		// This must happen AFTER UpdateAllProviders so connections are closed first
		if server, exists := rpsr.rpcServers[chainKey]; exists && server != nil {
			rpsr.cleanupStaleTrackers(chainKey, server, freshProviderSessions, freshBackupSessions)
		}
	}
}

// cleanupStaleTrackers removes ChainTrackers for endpoints that are no longer in the current provider sessions.
// This prevents resource leaks from trackers polling endpoints that have been removed during epoch updates.
func (rpsr *RPCSmartRouter) cleanupStaleTrackers(
	chainKey string,
	server *RPCSmartRouterServer,
	providerSessions map[uint64]*lavasession.ConsumerSessionsWithProvider,
	backupSessions map[uint64]*lavasession.ConsumerSessionsWithProvider,
) {
	if server.endpointChainTrackerManager == nil {
		return
	}

	// Build set of current endpoint URLs from both primary and backup providers
	currentEndpoints := make(map[string]bool)
	for _, provider := range providerSessions {
		for _, endpoint := range provider.Endpoints {
			currentEndpoints[endpoint.NetworkAddress] = true
		}
	}
	for _, provider := range backupSessions {
		for _, endpoint := range provider.Endpoints {
			currentEndpoints[endpoint.NetworkAddress] = true
		}
	}

	// Get all tracked endpoints and remove stale ones
	trackedEndpoints := server.endpointChainTrackerManager.GetAllEndpoints()
	removedCount := 0
	for _, trackedURL := range trackedEndpoints {
		if !currentEndpoints[trackedURL] {
			utils.LavaFormatInfo("removing stale ChainTracker on epoch update",
				utils.LogAttr("endpoint", trackedURL),
				utils.LogAttr("chainKey", chainKey),
			)
			server.endpointChainTrackerManager.RemoveTracker(trackedURL)
			removedCount++
		}
	}

	if removedCount > 0 {
		utils.LavaFormatInfo("epoch update: cleaned up stale ChainTrackers",
			utils.LogAttr("chainKey", chainKey),
			utils.LogAttr("removed", removedCount),
			utils.LogAttr("remaining", server.endpointChainTrackerManager.GetEndpointCount()),
		)
	}
}

func testModeWarn(desc string) {
	utils.LavaFormatWarning("------------------------------test mode --------------------------------\n\t\t\t"+
		desc+"\n\t\t\t"+
		"------------------------------test mode --------------------------------\n", nil)
}
