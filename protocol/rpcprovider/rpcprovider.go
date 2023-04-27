package rpcprovider

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/performance"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ChainTrackerDefaultMemory  = 100
	DEFAULT_ALLOWED_MISSING_CU = 0.2
)

var (
	Yaml_config_properties     = []string{"network-address", "chain-id", "api-interface", "node-urls.url"}
	DefaultRPCProviderFileName = "rpcprovider.yml"
)

type ProviderStateTrackerInf interface {
	RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser, chainID string) error
	RegisterReliabilityManagerForVoteUpdates(ctx context.Context, voteUpdatable statetracker.VoteUpdatable, endpointP *lavasession.RPCProviderEndpoint)
	RegisterForEpochUpdates(ctx context.Context, epochUpdatable statetracker.EpochUpdatable)
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, dataReliabilityProofs []*pairingtypes.VRFData, description string) error
	SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData) error
	SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData) error
	LatestBlock() int64
	GetVrfPkAndMaxCuForUser(ctx context.Context, consumerAddress string, chainID string, epocu uint64) (vrfPk *utils.VrfPubKey, maxCu uint64, err error)
	VerifyPairing(ctx context.Context, consumerAddress string, providerAddress string, epoch uint64, chainID string) (valid bool, index, total int64, err error)
	GetProvidersCountForConsumer(ctx context.Context, consumerAddress string, epoch uint64, chainID string) (uint32, error)
	GetEpochSize(ctx context.Context) (uint64, error)
	EarliestBlockInMemory(ctx context.Context) (uint64, error)
	RegisterPaymentUpdatableForPayments(ctx context.Context, paymentUpdatable statetracker.PaymentUpdatable)
	GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
	GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
}

type RPCProvider struct {
	providerStateTracker ProviderStateTrackerInf
	rpcProviderListeners map[string]*ProviderListener
	lock                 sync.Mutex
}

func (rpcp *RPCProvider) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, rpcProviderEndpoints []*lavasession.RPCProviderEndpoint, cache *performance.Cache, parallelConnections uint) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	rpcp.rpcProviderListeners = make(map[string]*ProviderListener)
	// single state tracker
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	providerStateTracker, err := statetracker.NewProviderStateTracker(ctx, txFactory, clientCtx, lavaChainFetcher)
	if err != nil {
		return err
	}
	rpcp.providerStateTracker = providerStateTracker
	// single reward server
	rewardServer := rewardserver.NewRewardServer(providerStateTracker)
	rpcp.providerStateTracker.RegisterForEpochUpdates(ctx, rewardServer)
	rpcp.providerStateTracker.RegisterPaymentUpdatableForPayments(ctx, rewardServer)
	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		utils.LavaFormatFatal("failed getting key name from clientCtx", err)
	}
	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		utils.LavaFormatFatal("failed getting private key from key name", err, utils.Attribute{Key: "keyName", Value: keyName})
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)
	lavaChainID := clientCtx.ChainID
	var addr sdk.AccAddress
	err = addr.Unmarshal(clientKey.GetPubKey().Address())
	if err != nil {
		utils.LavaFormatFatal("failed unmarshaling public address", err, utils.Attribute{Key: "keyName", Value: keyName}, utils.Attribute{Key: "pubkey", Value: clientKey.GetPubKey().Address()})
	}
	utils.LavaFormatInfo("RPCProvider pubkey: " + addr.String())
	utils.LavaFormatInfo("RPCProvider setting up endpoints", utils.Attribute{Key: "count", Value: strconv.Itoa(len(rpcProviderEndpoints))})
	blockMemorySize, err := rpcp.providerStateTracker.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx) // get the number of blocks to keep in PSM.
	if err != nil {
		utils.LavaFormatFatal("Failed fetching GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment in RPCProvider Start", err)
	}

	// pre loop to handle synchronous actions
	chainMutexes := map[string]*sync.Mutex{}
	for idx, endpoint := range rpcProviderEndpoints {
		chainMutexes[endpoint.ChainID] = &sync.Mutex{} // create a mutex per chain for shared resources
		if idx > 0 && endpoint.NetworkAddress == "" {  // handle undefined addresses as the previous endpoint for shared listeners
			endpoint.NetworkAddress = rpcProviderEndpoints[idx-1].NetworkAddress
		}
	}
	var stateTrackersPerChain sync.Map
	var wg sync.WaitGroup
	parallelJobs := len(rpcProviderEndpoints)
	wg.Add(parallelJobs)
	disabledEndpoints := make(chan *lavasession.RPCProviderEndpoint, parallelJobs)

	for _, rpcProviderEndpoint := range rpcProviderEndpoints {
		go func(rpcProviderEndpoint *lavasession.RPCProviderEndpoint) error {
			defer wg.Done()
			err := rpcProviderEndpoint.Validate()
			if err != nil {
				return utils.LavaFormatError("panic severity critical error, aborting support for chain api due to invalid node url definition, continuing with others", err, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()})
			}
			chainID := rpcProviderEndpoint.ChainID
			providerSessionManager := lavasession.NewProviderSessionManager(rpcProviderEndpoint, blockMemorySize)
			rpcp.providerStateTracker.RegisterForEpochUpdates(ctx, providerSessionManager)
			chainParser, err := chainlib.NewChainParser(rpcProviderEndpoint.ApiInterface)
			if err != nil {
				disabledEndpoints <- rpcProviderEndpoint
				return utils.LavaFormatError("panic severity critical error, aborting support for chain api due to invalid chain parser, continuing with others", err, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()})
			}
			providerStateTracker.RegisterChainParserForSpecUpdates(ctx, chainParser, chainID)
			_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
			chainProxy, err := chainlib.GetChainProxy(ctx, parallelConnections, rpcProviderEndpoint, chainParser)
			if err != nil {
				disabledEndpoints <- rpcProviderEndpoint
				return utils.LavaFormatError("panic severity critical error, failed creating chain proxy, continuing with others endpoints", err, utils.Attribute{Key: "parallelConnections", Value: uint64(parallelConnections)}, utils.Attribute{Key: "rpcProviderEndpoint", Value: rpcProviderEndpoint})
			}

			_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
			var chainTracker *chaintracker.ChainTracker

			// in order to utilize shared resources between chains we need go routines with the same chain to wait for one another here
			chainCommonSetup := func() error {
				chainMutexes[chainID].Lock()
				defer chainMutexes[chainID].Unlock()
				chainTrackerInf, found := stateTrackersPerChain.Load(chainID)
				if !found {
					blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)
					chainTrackerConfig := chaintracker.ChainTrackerConfig{
						BlocksToSave:      blocksToSaveChainTracker,
						AverageBlockTime:  averageBlockTime,
						ServerBlockMemory: ChainTrackerDefaultMemory + blocksToSaveChainTracker,
					}
					chainFetcher := chainlib.NewChainFetcher(ctx, chainProxy, chainParser, rpcProviderEndpoint)
					chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
					if err != nil {
						return utils.LavaFormatError("panic severity critical error, aborting support for chain api due to node access, continuing with other endpoints", err, utils.Attribute{Key: "chainTrackerConfig", Value: chainTrackerConfig}, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint})
					}
					stateTrackersPerChain.Store(rpcProviderEndpoint.ChainID, chainTracker)
				} else {
					var ok bool
					chainTracker, ok = chainTrackerInf.(*chaintracker.ChainTracker)
					if !ok {
						utils.LavaFormatFatal("invalid usage of syncmap, could not cast result into a chaintracker", nil)
					}
					utils.LavaFormatDebug("reusing chain tracker", utils.Attribute{Key: "chain", Value: rpcProviderEndpoint.ChainID})
				}

				return nil
			}
			err = chainCommonSetup()
			if err != nil {
				disabledEndpoints <- rpcProviderEndpoint
				return err
			}

			reliabilityManager := reliabilitymanager.NewReliabilityManager(chainTracker, providerStateTracker, addr.String(), chainProxy, chainParser)
			providerStateTracker.RegisterReliabilityManagerForVoteUpdates(ctx, reliabilityManager, rpcProviderEndpoint)

			rpcProviderServer := &RPCProviderServer{}
			rpcProviderServer.ServeRPCRequests(ctx, rpcProviderEndpoint, chainParser, rewardServer, providerSessionManager, reliabilityManager, privKey, cache, chainProxy, providerStateTracker, addr, lavaChainID, DEFAULT_ALLOWED_MISSING_CU)
			// set up grpc listener
			var listener *ProviderListener
			func() {
				rpcp.lock.Lock()
				defer rpcp.lock.Unlock()
				var ok bool
				listener, ok = rpcp.rpcProviderListeners[rpcProviderEndpoint.NetworkAddress]
				if !ok {
					utils.LavaFormatDebug("creating new listener", utils.Attribute{Key: "NetworkAddress", Value: rpcProviderEndpoint.NetworkAddress})
					listener = NewProviderListener(ctx, rpcProviderEndpoint.NetworkAddress)
					rpcp.rpcProviderListeners[rpcProviderEndpoint.NetworkAddress] = listener
				}
			}()
			if listener == nil {
				utils.LavaFormatFatal("listener not defined, cant register RPCProviderServer", nil, utils.Attribute{Key: "RPCProviderEndpoint", Value: rpcProviderEndpoint.String()})
			}
			listener.RegisterReceiver(rpcProviderServer, rpcProviderEndpoint)
			utils.LavaFormatDebug("provider finished setting up endpoint", utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.Key()})
			return nil
		}(rpcProviderEndpoint) // continue on error
	}
	wg.Wait()
	close(disabledEndpoints)
	utils.LavaFormatInfo("RPCProvider done setting up endpoints, ready for service")
	disabledEndpointsList := []*lavasession.RPCProviderEndpoint{}
	for disabledEndpoint := range disabledEndpoints {
		disabledEndpointsList = append(disabledEndpointsList, disabledEndpoint)
	}
	if len(disabledEndpointsList) > 0 {
		utils.LavaFormatError(utils.FormatStringerList("RPCProvider Runnig with disabled Endpoints:", disabledEndpointsList), nil)
		if len(disabledEndpointsList) == parallelJobs {
			utils.LavaFormatFatal("all endpoints are disabled", nil)
		}
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

	return nil
}

func ParseEndpoints(viper_endpoints *viper.Viper, geolocation uint64) (endpoints []*lavasession.RPCProviderEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(common.EndpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal endpoints", err, utils.Attribute{Key: "viper_endpoints", Value: viper_endpoints.AllSettings()})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation
	}
	return
}

func CreateRPCProviderCobraCommand() *cobra.Command {
	cmdRPCProvider := &cobra.Command{
		Use:   `rpcprovider [config-file] | { {listen-ip:listen-port spec-chain-id api-interface "comma-separated-node-urls"} ... }`,
		Short: `rpcprovider sets up a server to listen for rpc-consumers requests from the lava protocol send them to a configured node and respond with the reply`,
		Long: `rpcprovider sets up a server to listen for rpc-consumers requests from the lava protocol send them to a configured node and respond with the reply
		all configs should be located in` + app.DefaultNodeHome + "/config or the local running directory" + ` 
		if no arguments are passed, assumes default config file: ` + DefaultRPCProviderFileName + `
		if one argument is passed, its assumed the config file name
		`,
		Example: `required flags: --geolocation 1 --from alice
optional: --save-conf
rpcprovider <flags>
rpcprovider rpcprovider_conf.yml <flags>
rpcprovider 127.0.0.1:3333 ETH1 jsonrpc wss://www.eth-node.com:80 <flags>
rpcprovider 127.0.0.1:3333 COS3 tendermintrpc "wss://www.node-path.com:80,https://www.node-path.com:80" 127.0.0.1:3333 COS3 rest https://www.node-path.com:1317 <flags>`,
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
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("RPCProvider started", utils.Attribute{Key: "args", Value: strings.Join(args, ",")})
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
			rpcProviderEndpoints, err = ParseEndpoints(viper.GetViper(), geolocation)
			if err != nil || len(rpcProviderEndpoints) == 0 {
				return utils.LavaFormatError("invalid endpoints definition", err, utils.Attribute{Key: "endpoint_strings", Value: strings.Join(endpoints_strings, "")})
			}
			// handle flags, pass necessary fields
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			clientCtx = clientCtx.WithChainID(networkChainId)
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.LoggingLevel(logLevel)

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

			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.Seed(time.Now().UnixNano())
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
			numberOfNodeParallelConnections, err := cmd.Flags().GetUint(chainproxy.ParallelConnectionsFlag)
			if err != nil {
				utils.LavaFormatFatal("error fetching chainproxy.ParallelConnectionsFlag", err)
			}
			for _, endpoint := range rpcProviderEndpoints {
				utils.LavaFormatDebug("endpoint description", utils.Attribute{Key: "endpoint", Value: endpoint})
			}
			rpcProvider := RPCProvider{}
			err = rpcProvider.Start(ctx, txFactory, clientCtx, rpcProviderEndpoints, cache, numberOfNodeParallelConnections)
			return err
		},
	}

	// RPCProvider command flags
	flags.AddTxFlagsToCmd(cmdRPCProvider)
	cmdRPCProvider.MarkFlagRequired(flags.FlagFrom)
	cmdRPCProvider.Flags().Bool(common.SaveConfigFlagName, false, "save cmd args to a config file")
	cmdRPCProvider.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdRPCProvider.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCProvider.MarkFlagRequired(common.GeolocationFlag)
	cmdRPCProvider.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCProvider.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdRPCProvider.Flags().Uint(chainproxy.ParallelConnectionsFlag, chainproxy.NumberOfParallelConnections, "parallel connections")
	cmdRPCProvider.Flags().String(flags.FlagLogLevel, "debug", "log level")

	return cmdRPCProvider
}
