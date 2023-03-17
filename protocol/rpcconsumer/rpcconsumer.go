package rpcconsumer

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/performance"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Yaml_config_properties     = []string{"network-address", "chain-id", "api-interface"}
	DefaultRPCConsumerFileName = "rpcconsumer.yml"
)

type ConsumerStateTrackerInf interface {
	RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager)
	RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser, chainID string) error
	RegisterFinalizationConsensusForUpdates(context.Context, *lavaprotocol.FinalizationConsensus)
	TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) error
}

type RPCConsumer struct {
	consumerStateTracker ConsumerStateTrackerInf
	rpcConsumerServers   map[string]*RPCConsumerServer
}

// spawns a new RPCConsumer server with all it's processes and internals ready for communications
func (rpcc *RPCConsumer) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, rpcEndpoints []*lavasession.RPCEndpoint, requiredResponses int, vrf_sk vrf.PrivateKey, cache *performance.Cache) (err error) {
	if common.IsTestMode(ctx) {
		testModeWarn("RPCConsumer running tests")
	}
	// spawn up ConsumerStateTracker
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	consumerStateTracker, err := statetracker.NewConsumerStateTracker(ctx, txFactory, clientCtx, lavaChainFetcher)
	if err != nil {
		return err
	}
	rpcc.consumerStateTracker = consumerStateTracker
	rpcc.rpcConsumerServers = make(map[string]*RPCConsumerServer, len(rpcEndpoints))
	lavaChainID := clientCtx.ChainID
	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		utils.LavaFormatFatal("failed getting key name from clientCtx", err, nil)
	}
	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		utils.LavaFormatFatal("failed getting private key from key name", err, &map[string]string{"keyName": keyName})
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)

	var addr sdk.AccAddress
	err = addr.Unmarshal(clientKey.GetPubKey().Address())
	if err != nil {
		utils.LavaFormatFatal("failed unmarshaling public address", err, &map[string]string{"keyName": keyName, "pubkey": clientKey.GetPubKey().Address().String()})
	}
	utils.LavaFormatInfo("RPCConsumer pubkey: "+addr.String(), nil)
	utils.LavaFormatInfo("RPCConsumer setting up endpoints", &map[string]string{"length": strconv.Itoa(len(rpcEndpoints))})
	for _, rpcEndpoint := range rpcEndpoints {
		consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint)
		key := rpcEndpoint.Key()
		rpcc.consumerStateTracker.RegisterConsumerSessionManagerForPairingUpdates(ctx, consumerSessionManager)
		chainParser, err := chainlib.NewChainParser(rpcEndpoint.ApiInterface)
		if err != nil {
			return err
		}
		err = consumerStateTracker.RegisterChainParserForSpecUpdates(ctx, chainParser, rpcEndpoint.ChainID)
		if err != nil {
			return err
		}
		finalizationConsensus := &lavaprotocol.FinalizationConsensus{}
		consumerStateTracker.RegisterFinalizationConsensusForUpdates(ctx, finalizationConsensus)
		rpcc.rpcConsumerServers[key] = &RPCConsumerServer{}
		utils.LavaFormatInfo("RPCConsumer Listening", &map[string]string{"endpoints": rpcEndpoint.String()})
		rpcc.rpcConsumerServers[key].ServeRPCRequests(ctx, rpcEndpoint, rpcc.consumerStateTracker, chainParser, finalizationConsensus, consumerSessionManager, requiredResponses, privKey, vrf_sk, lavaChainID, cache)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	return nil
}

func ParseEndpoints(viper_endpoints *viper.Viper, geolocation uint64) (endpoints []*lavasession.RPCEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(common.EndpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal endpoints", err, &map[string]string{"viper_endpoints": fmt.Sprintf("%v", viper_endpoints.AllSettings())})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation
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
rpcconsumer 127.0.0.1:3333 COS3 tendermintrpc 127.0.0.1:3334 COS3 rest <flags>`,
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
			utils.LavaFormatInfo("RPCConsumer started", &map[string]string{"args": strings.Join(args, ",")})
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			config_name := DefaultRPCConsumerFileName
			if len(args) == 1 {
				config_name = args[0] // name of config file (without extension)
			}
			viper.SetConfigName(config_name)
			viper.SetConfigType("yml")
			viper.AddConfigPath(".")
			viper.AddConfigPath("./config")
			viper.AddConfigPath(app.DefaultNodeHome)
			var rpcEndpoints []*lavasession.RPCEndpoint
			var endpoints_strings []string
			var viper_endpoints *viper.Viper
			if len(args) > 1 {
				viper_endpoints, err = common.ParseEndpointArgs(args, Yaml_config_properties, common.EndpointsConfigName)
				if err != nil {
					return utils.LavaFormatError("invalid endpoints arguments", err, &map[string]string{"endpoint_strings": strings.Join(args, "")})
				}
				viper.MergeConfigMap(viper_endpoints.AllSettings())
				err := viper.SafeWriteConfigAs(DefaultRPCConsumerFileName)
				if err != nil {
					utils.LavaFormatInfo("did not create new config file, if it's desired remove the config file", &map[string]string{"file_name": viper.ConfigFileUsed()})
				} else {
					utils.LavaFormatInfo("created new config file", &map[string]string{"file_name": DefaultRPCConsumerFileName})
				}
			} else {
				err = viper.ReadInConfig()
				if err != nil {
					utils.LavaFormatFatal("could not load config file", err, &map[string]string{"expected_config_name": viper.ConfigFileUsed()})
				}
				utils.LavaFormatInfo("read config file successfully", &map[string]string{"expected_config_name": viper.ConfigFileUsed()})
			}
			geolocation, err := cmd.Flags().GetUint64(lavasession.GeolocationFlag)
			if err != nil {
				utils.LavaFormatFatal("failed to read geolocation flag, required flag", err, nil)
			}
			rpcEndpoints, err = ParseEndpoints(viper.GetViper(), geolocation)
			if err != nil || len(rpcEndpoints) == 0 {
				return utils.LavaFormatError("invalid endpoints definition", err, &map[string]string{"endpoint_strings": strings.Join(endpoints_strings, "")})
			}
			// handle flags, pass necessary fields
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err, nil)
			}
			utils.LoggingLevel(logLevel)

			test_mode, err := cmd.Flags().GetBool(common.TestModeFlagName)
			if err != nil {
				utils.LavaFormatFatal("failed to read test_mode flag", err, nil)
			}
			ctx = context.WithValue(ctx, common.TestModeFlagName, test_mode)
			// check if the command includes --pprof-address
			pprofAddressFlagUsed := cmd.Flags().Lookup("pprof-address").Changed
			if pprofAddressFlagUsed {
				// get pprof server ip address (default value: "")
				pprofServerAddress, err := cmd.Flags().GetString("pprof-address")
				if err != nil {
					utils.LavaFormatFatal("failed to read pprof address flag", err, nil)
				}

				// start pprof HTTP server
				err = performance.StartPprofServer(pprofServerAddress)
				if err != nil {
					return utils.LavaFormatError("failed to start pprof HTTP server", err, nil)
				}
			}
			clientCtx = clientCtx.WithChainID(networkChainId)
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			rpcConsumer := RPCConsumer{}
			requiredResponses := 1 // TODO: handle secure flag, for a majority between providers
			utils.LavaFormatInfo("lavad Binary Version: "+version.Version, nil)
			rand.Seed(time.Now().UnixNano())
			vrf_sk, _, err := utils.GetOrCreateVRFKey(clientCtx)
			if err != nil {
				utils.LavaFormatFatal("failed getting or creating a VRF key", err, nil)
			}
			var cache *performance.Cache = nil
			cacheAddr, err := cmd.Flags().GetString(performance.CacheFlagName)
			if err != nil {
				utils.LavaFormatError("Failed To Get Cache Address flag", err, &map[string]string{"flags": fmt.Sprintf("%v", cmd.Flags())})
			} else if cacheAddr != "" {
				cache, err = performance.InitCache(ctx, cacheAddr)
				if err != nil {
					utils.LavaFormatError("Failed To Connect to cache at address", err, &map[string]string{"address": cacheAddr})
				} else {
					utils.LavaFormatInfo("cache service connected", &map[string]string{"address": cacheAddr})
				}
			}
			err = rpcConsumer.Start(ctx, txFactory, clientCtx, rpcEndpoints, requiredResponses, vrf_sk, cache)
			return err
		},
	}

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdRPCConsumer)
	cmdRPCConsumer.MarkFlagRequired(flags.FlagFrom)
	cmdRPCConsumer.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdRPCConsumer.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCConsumer.MarkFlagRequired(common.GeolocationFlag)
	cmdRPCConsumer.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdRPCConsumer.Flags().Bool(common.TestModeFlagName, false, "test mode causes rpcconsumer to send dummy data and print all of the metadata in it's listeners")
	cmdRPCConsumer.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCConsumer.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")

	return cmdRPCConsumer
}

func testModeWarn(desc string) {
	utils.LavaFormatWarning("------------------------------test mode --------------------------------\n\t\t\t"+
		desc+"\n\t\t\t"+
		"------------------------------test mode --------------------------------\n", nil, nil)
}
