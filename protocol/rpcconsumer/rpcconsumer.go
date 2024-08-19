package rpcconsumer

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

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/app"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavaprotocol/finalizationconsensus"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/protocol/metrics"
	"github.com/lavanet/lava/v2/protocol/performance"
	"github.com/lavanet/lava/v2/protocol/provideroptimizer"
	"github.com/lavanet/lava/v2/protocol/rpcprovider"
	"github.com/lavanet/lava/v2/protocol/statetracker"
	"github.com/lavanet/lava/v2/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v2/protocol/upgrade"
	"github.com/lavanet/lava/v2/utils"
	specutils "github.com/lavanet/lava/v2/utils/keeper"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/lavanet/lava/v2/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	plantypes "github.com/lavanet/lava/v2/x/plans/types"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DefaultRPCConsumerFileName    = "rpcconsumer.yml"
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
)

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

var strategyFlag strategyValue = strategyValue{Strategy: provideroptimizer.STRATEGY_BALANCED}

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

type ConsumerStateTrackerInf interface {
	RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf)
	RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager, staticProvidersList []*lavasession.RPCProviderEndpoint)
	RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error
	RegisterFinalizationConsensusForUpdates(context.Context, *finalizationconsensus.FinalizationConsensus)
	RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error
	TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, conflictHandler common.ConflictHandlerInterface) error
	GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error)
	GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error)
	GetLatestVirtualEpoch() uint64
}

type AnalyticsServerAddressess struct {
	AddApiMethodCallsMetrics bool
	MetricsListenAddress     string
	RelayServerAddress       string
	ReportsAddressFlag       string
}
type RPCConsumer struct {
	consumerStateTracker ConsumerStateTrackerInf
}

type rpcConsumerStartOptions struct {
	txFactory                 tx.Factory
	clientCtx                 client.Context
	rpcEndpoints              []*lavasession.RPCEndpoint
	requiredResponses         int
	cache                     *performance.Cache
	strategy                  provideroptimizer.Strategy
	maxConcurrentProviders    uint
	analyticsServerAddressess AnalyticsServerAddressess
	cmdFlags                  common.ConsumerCmdFlags
	stateShare                bool
	refererData               *chainlib.RefererData
	staticProvidersList       []*lavasession.RPCProviderEndpoint // define static providers as backup to lava providers
}

// spawns a new RPCConsumer server with all it's processes and internals ready for communications
func (rpcc *RPCConsumer) Start(ctx context.Context, options *rpcConsumerStartOptions) (err error) {
	if common.IsTestMode(ctx) {
		testModeWarn("RPCConsumer running tests")
	}
	options.refererData.ReferrerClient = metrics.NewConsumerReferrerClient(options.refererData.Address)
	consumerReportsManager := metrics.NewConsumerReportsClient(options.analyticsServerAddressess.ReportsAddressFlag)
	consumerMetricsManager := metrics.NewConsumerMetricsManager(metrics.ConsumerMetricsManagerOptions{NetworkAddress: options.analyticsServerAddressess.MetricsListenAddress, AddMethodsApiGauge: options.analyticsServerAddressess.AddApiMethodCallsMetrics}) // start up prometheus metrics
	consumerUsageserveManager := metrics.NewConsumerRelayServerClient(options.analyticsServerAddressess.RelayServerAddress)                                                                                                                                    // start up relay server reporting
	rpcConsumerMetrics, err := metrics.NewRPCConsumerLogs(consumerMetricsManager, consumerUsageserveManager)
	if err != nil {
		utils.LavaFormatFatal("failed creating RPCConsumer logs", err)
	}
	consumerMetricsManager.SetVersion(upgrade.GetCurrentVersion().ConsumerVersion)

	// spawn up ConsumerStateTracker
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, options.clientCtx)
	consumerStateTracker, err := statetracker.NewConsumerStateTracker(ctx, options.txFactory, options.clientCtx, lavaChainFetcher, consumerMetricsManager, options.cmdFlags.DisableConflictTransactions)
	if err != nil {
		utils.LavaFormatFatal("failed to create a NewConsumerStateTracker", err)
	}
	rpcc.consumerStateTracker = consumerStateTracker

	lavaChainID := options.clientCtx.ChainID
	keyName, err := sigs.GetKeyName(options.clientCtx)
	if err != nil {
		utils.LavaFormatFatal("failed getting key name from clientCtx", err)
	}
	privKey, err := sigs.GetPrivKey(options.clientCtx, keyName)
	if err != nil {
		utils.LavaFormatFatal("failed getting private key from key name", err, utils.Attribute{Key: "keyName", Value: keyName})
	}
	clientKey, _ := options.clientCtx.Keyring.Key(keyName)

	pubkey, err := clientKey.GetPubKey()
	if err != nil {
		utils.LavaFormatFatal("failed getting public key from key name", err, utils.Attribute{Key: "keyName", Value: keyName})
	}

	var consumerAddr sdk.AccAddress
	err = consumerAddr.Unmarshal(pubkey.Address())
	if err != nil {
		utils.LavaFormatFatal("failed unmarshaling public address", err, utils.Attribute{Key: "keyName", Value: keyName}, utils.Attribute{Key: "pubkey", Value: pubkey.Address()})
	}
	// we want one provider optimizer per chain so we will store them for reuse across rpcEndpoints
	chainMutexes := map[string]*sync.Mutex{}
	for _, endpoint := range options.rpcEndpoints {
		chainMutexes[endpoint.ChainID] = &sync.Mutex{} // create a mutex per chain for shared resources
	}
	var optimizers sync.Map
	var consumerConsistencies sync.Map
	var finalizationConsensuses sync.Map
	var wg sync.WaitGroup
	parallelJobs := len(options.rpcEndpoints)
	wg.Add(parallelJobs)
	errCh := make(chan error)

	consumerStateTracker.RegisterForUpdates(ctx, updaters.NewMetricsUpdater(consumerMetricsManager))
	utils.LavaFormatInfo("RPCConsumer pubkey: " + consumerAddr.String())
	utils.LavaFormatInfo("RPCConsumer setting up endpoints", utils.Attribute{Key: "length", Value: strconv.Itoa(parallelJobs)})

	// check version
	version, err := consumerStateTracker.GetProtocolVersion(ctx)
	if err != nil {
		utils.LavaFormatFatal("failed fetching protocol version from node", err)
	}
	consumerStateTracker.RegisterForVersionUpdates(ctx, version.Version, &upgrade.ProtocolVersion{})
	relaysMonitorAggregator := metrics.NewRelaysMonitorAggregator(options.cmdFlags.RelaysHealthIntervalFlag, consumerMetricsManager)
	policyUpdaters := syncMapPolicyUpdaters{}
	for _, rpcEndpoint := range options.rpcEndpoints {
		go func(rpcEndpoint *lavasession.RPCEndpoint) error {
			defer wg.Done()
			chainParser, err := chainlib.NewChainParser(rpcEndpoint.ApiInterface)
			if err != nil {
				err = utils.LavaFormatError("failed creating chain parser", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
				errCh <- err
				return err
			}
			chainID := rpcEndpoint.ChainID
			// create policyUpdaters per chain
			if policyUpdater, ok := policyUpdaters.Load(rpcEndpoint.ChainID); ok {
				err := policyUpdater.AddPolicySetter(chainParser, *rpcEndpoint)
				if err != nil {
					errCh <- err
					return utils.LavaFormatError("failed adding policy setter", err)
				}
			} else {
				policyUpdaters.Store(rpcEndpoint.ChainID, updaters.NewPolicyUpdater(chainID, consumerStateTracker, consumerAddr.String(), chainParser, *rpcEndpoint))
			}

			if options.cmdFlags.OfflineSpecPath != "" {
				// offline spec mode.
				parsedOfflineSpec, loadError := specutils.GetSpecFromPath(options.cmdFlags.OfflineSpecPath, rpcEndpoint.ChainID, nil, nil)
				if loadError != nil {
					err = utils.LavaFormatError("failed loading offline spec", err, utils.LogAttr("spec_path", options.cmdFlags.OfflineSpecPath), utils.LogAttr("spec_id", rpcEndpoint.ChainID))
				}
				utils.LavaFormatInfo("Loaded offline spec successfully", utils.LogAttr("spec_path", options.cmdFlags.OfflineSpecPath), utils.LogAttr("chain_id", parsedOfflineSpec.Index))
				chainParser.SetSpec(parsedOfflineSpec)
			} else {
				// register for spec updates
				err = rpcc.consumerStateTracker.RegisterForSpecUpdates(ctx, chainParser, *rpcEndpoint)
			}
			if err != nil {
				err = utils.LavaFormatError("failed registering for spec updates", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
				errCh <- err
				return err
			}

			_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
			var optimizer *provideroptimizer.ProviderOptimizer
			var consumerConsistency *ConsumerConsistency
			var finalizationConsensus *finalizationconsensus.FinalizationConsensus
			getOrCreateChainAssets := func() error {
				// this is locked so we don't race optimizers creation
				chainMutexes[chainID].Lock()
				defer chainMutexes[chainID].Unlock()
				value, exists := optimizers.Load(chainID)
				if !exists {
					// doesn't exist for this chain create a new one
					baseLatency := common.AverageWorldLatency / 2 // we want performance to be half our timeout or better
					optimizer = provideroptimizer.NewProviderOptimizer(options.strategy, averageBlockTime, baseLatency, options.maxConcurrentProviders)
					optimizers.Store(chainID, optimizer)
				} else {
					var ok bool
					optimizer, ok = value.(*provideroptimizer.ProviderOptimizer)
					if !ok {
						err = utils.LavaFormatError("failed loading optimizer, value is of the wrong type", nil, utils.Attribute{Key: "endpoint", Value: rpcEndpoint.Key()})
						return err
					}
				}
				value, exists = consumerConsistencies.Load(chainID)
				if !exists { // doesn't exist for this chain create a new one
					consumerConsistency = NewConsumerConsistency(chainID)
					consumerConsistencies.Store(chainID, consumerConsistency)
				} else {
					var ok bool
					consumerConsistency, ok = value.(*ConsumerConsistency)
					if !ok {
						err = utils.LavaFormatError("failed loading consumer consistency, value is of the wrong type", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint.Key()})
						return err
					}
				}

				value, exists = finalizationConsensuses.Load(chainID)
				if !exists {
					// doesn't exist for this chain create a new one
					finalizationConsensus = finalizationconsensus.NewFinalizationConsensus(rpcEndpoint.ChainID)
					consumerStateTracker.RegisterFinalizationConsensusForUpdates(ctx, finalizationConsensus)
					finalizationConsensuses.Store(chainID, finalizationConsensus)
				} else {
					var ok bool
					finalizationConsensus, ok = value.(*finalizationconsensus.FinalizationConsensus)
					if !ok {
						err = utils.LavaFormatError("failed loading finalization consensus, value is of the wrong type", nil, utils.Attribute{Key: "endpoint", Value: rpcEndpoint.Key()})
						return err
					}
				}
				return nil
			}
			err = getOrCreateChainAssets()
			if err != nil {
				errCh <- err
				return err
			}

			if finalizationConsensus == nil || optimizer == nil {
				err = utils.LavaFormatError("failed getting assets, found a nil", nil, utils.Attribute{Key: "endpoint", Value: rpcEndpoint.Key()})
				errCh <- err
				return err
			}

			// Create active subscription provider storage for each unique chain
			activeSubscriptionProvidersStorage := lavasession.NewActiveSubscriptionProvidersStorage()
			consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, consumerMetricsManager, consumerReportsManager, consumerAddr.String(), activeSubscriptionProvidersStorage)
			// Register For Updates
			rpcc.consumerStateTracker.RegisterConsumerSessionManagerForPairingUpdates(ctx, consumerSessionManager, options.staticProvidersList)

			var relaysMonitor *metrics.RelaysMonitor
			if options.cmdFlags.RelaysHealthEnableFlag {
				relaysMonitor = metrics.NewRelaysMonitor(options.cmdFlags.RelaysHealthIntervalFlag, rpcEndpoint.ChainID, rpcEndpoint.ApiInterface)
				relaysMonitorAggregator.RegisterRelaysMonitor(rpcEndpoint.String(), relaysMonitor)
			}

			rpcConsumerServer := &RPCConsumerServer{}

			var consumerWsSubscriptionManager *chainlib.ConsumerWSSubscriptionManager
			var specMethodType string
			if rpcEndpoint.ApiInterface == spectypes.APIInterfaceJsonRPC {
				specMethodType = http.MethodPost
			}
			consumerWsSubscriptionManager = chainlib.NewConsumerWSSubscriptionManager(consumerSessionManager, rpcConsumerServer, options.refererData, specMethodType, chainParser, activeSubscriptionProvidersStorage)

			utils.LavaFormatInfo("RPCConsumer Listening", utils.Attribute{Key: "endpoints", Value: rpcEndpoint.String()})
			err = rpcConsumerServer.ServeRPCRequests(ctx, rpcEndpoint, rpcc.consumerStateTracker, chainParser, finalizationConsensus, consumerSessionManager, options.requiredResponses, privKey, lavaChainID, options.cache, rpcConsumerMetrics, consumerAddr, consumerConsistency, relaysMonitor, options.cmdFlags, options.stateShare, options.refererData, consumerReportsManager, consumerWsSubscriptionManager)
			if err != nil {
				err = utils.LavaFormatError("failed serving rpc requests", err, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
				errCh <- err
				return err
			}
			return nil
		}(rpcEndpoint)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}

	relaysMonitorAggregator.StartMonitoring(ctx)

	utils.LavaFormatDebug("Starting Policy Updaters for all chains")
	for chainId := range chainMutexes {
		policyUpdater, ok := policyUpdaters.Load(chainId)
		if !ok {
			utils.LavaFormatError("could not load policy Updater for chain", nil, utils.LogAttr("chain", chainId))
			continue
		}
		consumerStateTracker.RegisterForPairingUpdates(ctx, policyUpdater, chainId)
		emergencyTracker, ok := consumerStateTracker.ConsumerEmergencyTrackerInf.(*statetracker.EmergencyTracker)
		if !ok {
			utils.LavaFormatFatal("Failed converting consumerStateTracker.ConsumerEmergencyTrackerInf to *statetracker.EmergencyTracker", nil, utils.LogAttr("chain", chainId))
		}
		consumerStateTracker.RegisterForPairingUpdates(ctx, emergencyTracker, chainId)
	}

	utils.LavaFormatInfo("RPCConsumer done setting up all endpoints, ready for requests")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
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
	return
}

func CreateRPCConsumerCobraCommand() *cobra.Command {
	cmdRPCConsumer := &cobra.Command{
		Use:   "rpcconsumer [config-file] | { {listen-ip:listen-port spec-chain-id api-interface} ... }",
		Short: `rpcconsumer sets up a server to perform api requests and sends them through the lava protocol to data providers`,
		Long: `rpcconsumer sets up a server to perform api requests and sends them through the lava protocol to data providers
		all configs should be located in the local running directory /config or ` + app.DefaultNodeHome + `
		if no arguments are passed, assumes default config file: ` + DefaultRPCConsumerFileName + `
		if one argument is passed, its assumed the config file name
		`,
		Example: `required flags: --geolocation 1 --from alice
rpcconsumer <flags>
rpcconsumer rpcconsumer_conf <flags>
rpcconsumer 127.0.0.1:3333 OSMOSIS tendermintrpc 127.0.0.1:3334 OSMOSIS rest <flags>
rpcconsumer consumer_examples/full_consumer_example.yml --cache-be "127.0.0.1:7778" --geolocation 1 [--debug-relays] --log_level <debug|warn|...> --from <wallet> --chain-id <lava-chain> --strategy latency`,
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
			config_name := DefaultRPCConsumerFileName
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
				err := viper.SafeWriteConfigAs(DefaultRPCConsumerFileName)
				if err != nil {
					utils.LavaFormatInfo("did not create new config file, if it's desired remove the config file", utils.Attribute{Key: "file_name", Value: viper.ConfigFileUsed()})
				} else {
					utils.LavaFormatInfo("created new config file", utils.Attribute{Key: "file_name", Value: DefaultRPCConsumerFileName})
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

			// check if StaticProvidersConfigName exists in viper, if it does parse it with ParseStaticProvider function
			var staticProviderEndpoints []*lavasession.RPCProviderEndpoint
			if viper.IsSet(common.StaticProvidersConfigName) {
				staticProviderEndpoints, err = rpcprovider.ParseEndpointsCustomName(viper.GetViper(), common.StaticProvidersConfigName, geolocation)
				if err != nil {
					return utils.LavaFormatError("invalid static providers definition", err)
				}
				for _, endpoint := range staticProviderEndpoints {
					utils.LavaFormatInfo("Static Provider Endpoint:", utils.Attribute{Key: "Urls", Value: endpoint.NodeUrls}, utils.Attribute{Key: "Chain ID", Value: endpoint.ChainID}, utils.Attribute{Key: "API Interface", Value: endpoint.ApiInterface})
				}
			}

			// set up the txFactory with gas adjustments and gas
			txFactory = txFactory.WithGasAdjustment(viper.GetFloat64(flags.FlagGasAdjustment))
			txFactory = txFactory.WithGasPrices(gasPricesStr)
			utils.LavaFormatInfo("Setting gas for tx Factory", utils.LogAttr("gas-prices", gasPricesStr), utils.LogAttr("gas-adjustment", txFactory.GasAdjustment()))
			rpcConsumer := RPCConsumer{}
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
			if strategyFlag.Strategy != provideroptimizer.STRATEGY_BALANCED {
				utils.LavaFormatInfo("Working with selection strategy: " + strategyFlag.String())
			}

			analyticsServerAddressess := AnalyticsServerAddressess{
				AddApiMethodCallsMetrics: viper.GetBool(metrics.AddApiMethodCallsMetrics),
				MetricsListenAddress:     viper.GetString(metrics.MetricsListenFlagName),
				RelayServerAddress:       viper.GetString(metrics.RelayServerFlagName),
				ReportsAddressFlag:       viper.GetString(reportsSendBEAddress),
			}

			var refererData *chainlib.RefererData
			if viper.GetString(refererBackendAddressFlagName) != "" || viper.GetString(refererMarkerFlagName) != "" {
				refererData = &chainlib.RefererData{
					Address: viper.GetString(refererBackendAddressFlagName), // address is used to send to a backend if necessary
					Marker:  viper.GetString(refererMarkerFlagName),         // marker is necessary to unwrap paths
				}
			}

			maxConcurrentProviders := viper.GetUint(common.MaximumConcurrentProvidersFlagName)
			consumerPropagatedFlags := common.ConsumerCmdFlags{
				HeadersFlag:                 viper.GetString(common.CorsHeadersFlag),
				CredentialsFlag:             viper.GetString(common.CorsCredentialsFlag),
				OriginFlag:                  viper.GetString(common.CorsOriginFlag),
				MethodsFlag:                 viper.GetString(common.CorsMethodsFlag),
				CDNCacheDuration:            viper.GetString(common.CDNCacheDurationFlag),
				RelaysHealthEnableFlag:      viper.GetBool(common.RelaysHealthEnableFlag),
				RelaysHealthIntervalFlag:    viper.GetDuration(common.RelayHealthIntervalFlag),
				DebugRelays:                 viper.GetBool(DebugRelaysFlagName),
				DisableConflictTransactions: viper.GetBool(common.DisableConflictTransactionsFlag),
				DisableRetryOnNodeErrors:    viper.GetBool(common.DisableRetryOnNodeErrorsFlag),
				OfflineSpecPath:             viper.GetString(common.UseOfflineSpecFlag),
			}

			// validate user is does not provide multi chain setup when using the offline spec feature.
			if consumerPropagatedFlags.OfflineSpecPath != "" && len(rpcEndpoints) > 1 {
				utils.LavaFormatFatal("offline spec modifications are supported only in single chain bootstrapping", nil, utils.LogAttr("len(rpcEndpoints)", len(rpcEndpoints)), utils.LogAttr("rpcEndpoints", rpcEndpoints))
			}

			rpcConsumerSharedState := viper.GetBool(common.SharedStateFlag)
			err = rpcConsumer.Start(ctx, &rpcConsumerStartOptions{txFactory, clientCtx, rpcEndpoints, requiredResponses, cache, strategyFlag.Strategy, maxConcurrentProviders, analyticsServerAddressess, consumerPropagatedFlags, rpcConsumerSharedState, refererData, staticProviderEndpoints})
			return err
		},
	}

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdRPCConsumer)
	cmdRPCConsumer.MarkFlagRequired(flags.FlagFrom)
	cmdRPCConsumer.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCConsumer.Flags().Uint(common.MaximumConcurrentProvidersFlagName, 3, "max number of concurrent providers to communicate with")
	cmdRPCConsumer.MarkFlagRequired(common.GeolocationFlag)
	cmdRPCConsumer.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdRPCConsumer.Flags().Bool(lavasession.AllowInsecureConnectionToProvidersFlag, false, "allow insecure provider-dialing. used for development and testing")
	cmdRPCConsumer.Flags().Bool(lavasession.AllowGRPCCompressionFlag, false, "allow messages to be compressed when communicating between the consumer and provider")
	cmdRPCConsumer.Flags().Bool(common.TestModeFlagName, false, "test mode causes rpcconsumer to send dummy data and print all of the metadata in it's listeners")
	cmdRPCConsumer.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCConsumer.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdRPCConsumer.Flags().Var(&strategyFlag, "strategy", fmt.Sprintf("the strategy to use to pick providers (%s)", strings.Join(strategyNames, "|")))
	cmdRPCConsumer.Flags().String(metrics.MetricsListenFlagName, metrics.DisabledFlagOption, "the address to expose prometheus metrics (such as localhost:7779)")
	cmdRPCConsumer.Flags().Bool(metrics.AddApiMethodCallsMetrics, false, "adding a counter gauge for each method called per chain per api interface")
	cmdRPCConsumer.Flags().String(metrics.RelayServerFlagName, metrics.DisabledFlagOption, "the http address of the relay usage server api endpoint (example http://127.0.0.1:8080)")
	cmdRPCConsumer.Flags().Bool(DebugRelaysFlagName, false, "adding debug information to relays")
	// CORS related flags
	cmdRPCConsumer.Flags().String(common.CorsCredentialsFlag, "true", "Set up CORS allowed credentials,default \"true\"")
	cmdRPCConsumer.Flags().String(common.CorsHeadersFlag, "", "Set up CORS allowed headers, * for all, default simple cors specification headers")
	cmdRPCConsumer.Flags().String(common.CorsOriginFlag, "*", "Set up CORS allowed origin, enabled * by default")
	cmdRPCConsumer.Flags().String(common.CorsMethodsFlag, "GET,POST,PUT,DELETE,OPTIONS", "set up Allowed OPTIONS methods, defaults to: \"GET,POST,PUT,DELETE,OPTIONS\"")
	cmdRPCConsumer.Flags().String(common.CDNCacheDurationFlag, "86400", "set up preflight options response cache duration, default 86400 (24h in seconds)")
	cmdRPCConsumer.Flags().Bool(common.SharedStateFlag, false, "Share the consumer consistency state with the cache service. this should be used with cache backend enabled if you want to state sync multiple rpc consumers")
	// Relays health check related flags
	cmdRPCConsumer.Flags().Bool(common.RelaysHealthEnableFlag, RelaysHealthEnableFlagDefault, "enables relays health check")
	cmdRPCConsumer.Flags().Duration(common.RelayHealthIntervalFlag, RelayHealthIntervalFlagDefault, "interval between relay health checks")
	cmdRPCConsumer.Flags().String(refererBackendAddressFlagName, "", "address to send referer to")
	cmdRPCConsumer.Flags().String(refererMarkerFlagName, "lava-referer-", "the string marker to identify referer")
	cmdRPCConsumer.Flags().String(reportsSendBEAddress, "", "address to send reports to")
	cmdRPCConsumer.Flags().BoolVar(&lavasession.DebugProbes, DebugProbesFlagName, false, "adding information to probes")
	cmdRPCConsumer.Flags().Bool(common.DisableConflictTransactionsFlag, false, "disabling conflict transactions, this flag should not be used as it harms the network's data reliability and therefore the service.")
	cmdRPCConsumer.Flags().DurationVar(&updaters.TimeOutForFetchingLavaBlocks, common.TimeOutForFetchingLavaBlocksFlag, time.Second*5, "setting the timeout for fetching lava blocks")
	cmdRPCConsumer.Flags().Bool(common.DisableRetryOnNodeErrorsFlag, false, "Disable relay retries on node errors, prevent the rpcconsumer trying a different provider")
	cmdRPCConsumer.Flags().String(common.UseOfflineSpecFlag, "", "load offline spec provided path to spec file, used to test specs before they are proposed on chain")

	common.AddRollingLogConfig(cmdRPCConsumer)
	return cmdRPCConsumer
}

func testModeWarn(desc string) {
	utils.LavaFormatWarning("------------------------------test mode --------------------------------\n\t\t\t"+
		desc+"\n\t\t\t"+
		"------------------------------test mode --------------------------------\n", nil)
}
