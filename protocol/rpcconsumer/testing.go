package rpcconsumer

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chaintracker"
	commonlib "github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func startTesting(ctx context.Context, clientCtx client.Context, txFactory tx.Factory, rpcEndpoints []*lavasession.RPCProviderEndpoint, parallelConnections uint) error {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	stateTracker, err := statetracker.NewConsumerStateTracker(ctx, txFactory, clientCtx, lavaChainFetcher)
	if err != nil {
		return err
	}
	for _, rpcProviderEndpoint := range rpcEndpoints {
		go func(rpcProviderEndpoint *lavasession.RPCProviderEndpoint) error {
			chainParser, err := chainlib.NewChainParser(rpcProviderEndpoint.ApiInterface)
			if err != nil {
				utils.LavaFormatError("panic severity critical error, aborting support for chain api due to invalid chain parser, continuing with others", err, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()})
			}
			stateTracker.RegisterForSpecUpdates(ctx, chainParser, lavasession.RPCEndpoint{ChainID: rpcProviderEndpoint.ChainID, ApiInterface: rpcProviderEndpoint.ApiInterface})
			chainProxy, err := chainlib.GetChainRouter(ctx, parallelConnections, rpcProviderEndpoint, chainParser)
			if err != nil {
				return utils.LavaFormatError("panic severity critical error, failed creating chain proxy, continuing with others endpoints", err, utils.Attribute{Key: "parallelConnections", Value: uint64(parallelConnections)}, utils.Attribute{Key: "rpcProviderEndpoint", Value: rpcProviderEndpoint})
			}
			printOnNewLatestCallback := func(block int64, hash string) {
				utils.LavaFormatInfo("Received a new Block",
					utils.Attribute{Key: "block", Value: block},
					utils.Attribute{Key: "hash", Value: hash},
					utils.Attribute{Key: "chain", Value: rpcProviderEndpoint.ChainID},
					utils.Attribute{Key: "apiInterface", Value: rpcProviderEndpoint.ApiInterface},
				)
			}
			_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
			blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)
			chainTrackerConfig := chaintracker.ChainTrackerConfig{
				BlocksToSave:      blocksToSaveChainTracker,
				AverageBlockTime:  averageBlockTime,
				ServerBlockMemory: rpcprovider.ChainTrackerDefaultMemory + blocksToSaveChainTracker,
				NewLatestCallback: printOnNewLatestCallback,
			}
			chainFetcher := chainlib.NewChainFetcher(ctx, chainProxy, chainParser, rpcProviderEndpoint, nil)
			chainTracker, err := chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
			if err != nil {
				return utils.LavaFormatError("panic severity critical error, aborting support for chain api due to node access, continuing with other endpoints", err, utils.Attribute{Key: "chainTrackerConfig", Value: chainTrackerConfig}, utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint})
			}
			_ = chainTracker // let the chain tracker work and make queries
			return nil
		}(rpcProviderEndpoint)
	}
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("test rpcconsumer ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("test rpcconsumer signalChan")
	}
	return nil
}

func CreateTestRPCConsumerCobraCommand() *cobra.Command {
	cmdTestRPCConsumer := &cobra.Command{
		Use:     `rpcconsumer {listen-ip:listen-port spec-chain-id api-interface} ... `,
		Short:   `test an rpc consumer by making calls in the chain api interface requested`,
		Long:    `sets up a client that requests for blocks in the requested api on the listen port to perform tests on an rpcconsumer that is active`,
		Example: `rpcconsumer "http://127.0.0.1:3333 ETH1 jsonrpc http://127.0.0.1:3334 LAV1 rest 127.0.0.1:3334 LAV1 grpc"`,
		Args: func(cmd *cobra.Command, args []string) error {
			argLen := len(args)
			if argLen == 0 || argLen%len(Yaml_config_properties) != 0 {
				return fmt.Errorf("invalid number of arguments, needs to be a repeated groups of %d, while arg count is %d", len(Yaml_config_properties), argLen)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("RPCConsumer Test started", utils.Attribute{Key: "args", Value: strings.Join(args, ";")})
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			// handle flags, pass necessary fields
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.LoggingLevel(logLevel)
			var viper_endpoints *viper.Viper
			viper_endpoints, err = commonlib.ParseEndpointArgs(args, Yaml_config_properties, commonlib.EndpointsConfigName)
			if err != nil {
				return utils.LavaFormatError("invalid endpoints arguments", err, utils.Attribute{Key: "endpoint_strings", Value: strings.Join(args, "")})
			}
			viper.MergeConfigMap(viper_endpoints.AllSettings())
			var rpcEndpoints []*lavasession.RPCEndpoint
			rpcEndpoints, err = ParseEndpoints(viper.GetViper(), 1)
			if err != nil || len(rpcEndpoints) == 0 {
				return utils.LavaFormatError("invalid endpoints definition", err)
			}
			modifiedProviderEndpoints := make([]*lavasession.RPCProviderEndpoint, len(rpcEndpoints))
			for idx := range modifiedProviderEndpoints {
				endpoint := rpcEndpoints[idx]
				modifiedProviderEndpoints[idx] = &lavasession.RPCProviderEndpoint{
					NetworkAddress: lavasession.NetworkAddressData{Address: ""},
					ChainID:        endpoint.ChainID,
					ApiInterface:   endpoint.ApiInterface,
					Geolocation:    1, // doesn't matter
					NodeUrls:       []commonlib.NodeUrl{{Url: endpoint.NetworkAddress}},
				}
			}
			clientCtx = clientCtx.WithChainID(networkChainId)
			txFactory, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to create txFactory", err)
			}
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.Seed(time.Now().UnixNano())
			numberOfNodeParallelConnections, err := cmd.Flags().GetUint(chainproxy.ParallelConnectionsFlag)
			if err != nil {
				utils.LavaFormatFatal("error fetching chainproxy.ParallelConnectionsFlag", err)
			}
			return startTesting(ctx, clientCtx, txFactory, modifiedProviderEndpoints, numberOfNodeParallelConnections)
		},
	}

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdTestRPCConsumer)
	cmdTestRPCConsumer.Flags().Uint(chainproxy.ParallelConnectionsFlag, chainproxy.NumberOfParallelConnections, "parallel connections")
	return cmdTestRPCConsumer
}
