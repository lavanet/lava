package rpcsmartrouter

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	commonlib "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/statetracker"
	protocoltypes "github.com/lavanet/lava/v5/types/protocol"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startTesting connects to a running smart router's endpoints and starts chain
// trackers to validate that the router is proxying blocks correctly.
//
// staticSpecPaths must be non-empty: the smart router does not query the
// blockchain, so specs must be loaded from files/URLs (--use-static-spec).
// Any goroutine that fails during setup sends its error to errCh, which causes
// the command to cancel and return the error immediately.
func startTesting(ctx context.Context, rpcEndpoints []*lavasession.RPCProviderEndpoint, parallelConnections uint, staticSpecPaths []string) error {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	if len(staticSpecPaths) == 0 {
		return utils.LavaFormatError("--use-static-spec is required in smart-router mode (no blockchain connection available)", nil)
	}

	chainlib.IgnoreWsEnforcementForTestCommands = true // ignore ws panic for tests

	// errCh receives the first setup error from any goroutine; buffered so
	// senders never block even if we only read one error.
	errCh := make(chan error, len(rpcEndpoints))

	for _, rpcProviderEndpoint := range rpcEndpoints {
		go func(rpcProviderEndpoint *lavasession.RPCProviderEndpoint) {
			chainParser, err := chainlib.NewChainParser(rpcProviderEndpoint.ApiInterface)
			if err != nil {
				errCh <- utils.LavaFormatError("failed creating chain parser, aborting endpoint", err,
					utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint.String()})
				return
			}

			// In smart-router mode NewConsumerStateQuery returns a stub whose
			// GetSpec always errors.  Use static spec loading instead, mirroring
			// what the main rpcsmartrouter command does via RegisterForSpecUpdates.
			rpcEndpoint := lavasession.RPCEndpoint{
				ChainID:      rpcProviderEndpoint.ChainID,
				ApiInterface: rpcProviderEndpoint.ApiInterface,
				Geolocation:  1,
			}
			if err := statetracker.RegisterForSpecUpdatesOrSetStaticSpecsWithToken(
				ctx, chainParser, staticSpecPaths, rpcEndpoint,
				nil, // no live spec updater in static mode
				"",  // github token — not needed for local files
				"",  // gitlab token — not needed for local files
			); err != nil {
				errCh <- utils.LavaFormatError("failed loading spec for endpoint", err,
					utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID},
					utils.Attribute{Key: "apiInterface", Value: rpcProviderEndpoint.ApiInterface},
				)
				return
			}

			chainProxy, err := chainlib.GetChainRouter(ctx, parallelConnections, rpcProviderEndpoint, chainParser)
			if err != nil {
				errCh <- utils.LavaFormatError("failed creating chain proxy, aborting endpoint", err,
					utils.Attribute{Key: "parallelConnections", Value: uint64(parallelConnections)},
					utils.Attribute{Key: "rpcProviderEndpoint", Value: rpcProviderEndpoint})
				return
			}

			printOnNewLatestCallback := func(blockFrom int64, blockTo int64, hash string) {
				for block := blockFrom + 1; block <= blockTo; block++ {
					utils.LavaFormatInfo("Received a new Block",
						utils.Attribute{Key: "block", Value: block},
						utils.Attribute{Key: "hash", Value: hash},
						utils.Attribute{Key: "Chain", Value: rpcProviderEndpoint.ChainID},
						utils.Attribute{Key: "apiInterface", Value: rpcProviderEndpoint.ApiInterface},
					)
				}
			}
			consistencyErrorCallback := func(oldBlock, newBlock int64) {
				utils.LavaFormatError("Consistency issue detected", nil,
					utils.Attribute{Key: "oldBlock", Value: oldBlock},
					utils.Attribute{Key: "newBlock", Value: newBlock},
					utils.Attribute{Key: "Chain", Value: rpcProviderEndpoint.ChainID},
					utils.Attribute{Key: "apiInterface", Value: rpcProviderEndpoint.ApiInterface},
				)
			}

			_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
			blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)
			chainTrackerConfig := chaintracker.ChainTrackerConfig{
				BlocksToSave:          blocksToSaveChainTracker,
				AverageBlockTime:      averageBlockTime,
				ServerBlockMemory:     chaintracker.ChainTrackerDefaultMemory + blocksToSaveChainTracker,
				NewLatestCallback:     printOnNewLatestCallback,
				ConsistencyCallback:   consistencyErrorCallback,
				ParseDirectiveEnabled: true,
			}
			chainFetcher := chainlib.NewChainFetcher(ctx, &chainlib.ChainFetcherOptions{ChainRouter: chainProxy, ChainParser: chainParser, Endpoint: rpcProviderEndpoint, Cache: nil})
			chainTracker, err := chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
			if err != nil {
				errCh <- utils.LavaFormatError("failed creating chain tracker, aborting endpoint", err,
					utils.Attribute{Key: "chainTrackerConfig", Value: chainTrackerConfig},
					utils.Attribute{Key: "endpoint", Value: rpcProviderEndpoint})
				return
			}
			chainTracker.StartAndServe(ctx)
			_ = chainTracker // let the chain tracker work and make queries
		}(rpcProviderEndpoint)
	}

	select {
	case err := <-errCh:
		// A goroutine failed during setup — cancel all others and propagate.
		cancel()
		return err
	case <-ctx.Done():
		utils.LavaFormatInfo("test rpcsmartrouter ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("test rpcsmartrouter signalChan")
	}
	return nil
}

func CreateTestRPCSmartRouterCobraCommand() *cobra.Command {
	cmdTestRPCSmartRouter := &cobra.Command{
		Use:     `rpcsmartrouter {listen-ip:listen-port spec-chain-id api-interface} ... `,
		Short:   `test an rpc smart router by making calls in the chain api interface requested`,
		Long:    `sets up a client that requests for blocks in the requested api on the listen port to perform tests on an rpcsmartrouter that is active`,
		Example: `rpcsmartrouter "http://127.0.0.1:3333 ETH1 jsonrpc http://127.0.0.1:3334 LAV1 rest 127.0.0.1:3334 LAV1 grpc"`,
		Args: func(cmd *cobra.Command, args []string) error {
			argLen := len(args)
			if argLen == 0 || argLen%len(Yaml_config_properties) != 0 {
				return fmt.Errorf("invalid number of arguments, needs to be a repeated groups of %d, while arg count is %d", len(Yaml_config_properties), argLen)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("RPCConsumer Test started", utils.Attribute{Key: "args", Value: strings.Join(args, ";")})
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString("chain-id")
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString("log-level")
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			staticSpecPaths, err := cmd.Flags().GetStringArray(commonlib.UseStaticSpecFlag)
			if err != nil {
				return utils.LavaFormatError("failed to read --use-static-spec flag", err)
			}

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
				err := commonlib.ValidateEndpoint(endpoint.NetworkAddress, endpoint.ApiInterface)
				if err != nil {
					return err
				}
				modifiedProviderEndpoints[idx] = &lavasession.RPCProviderEndpoint{
					NetworkAddress: lavasession.NetworkAddressData{Address: ""},
					ChainID:        endpoint.ChainID,
					ApiInterface:   endpoint.ApiInterface,
					Geolocation:    1, // doesn't matter
					NodeUrls: []commonlib.NodeUrl{{
						Url: endpoint.NetworkAddress,
						AuthConfig: commonlib.AuthConfig{
							UseTLS:        viper.GetBool(chainproxy.GRPCUseTls),
							AllowInsecure: viper.GetBool(chainproxy.GRPCAllowInsecureConnection),
						},
					}},
				}
			}
			_ = networkChainId // chain-id flag kept for CLI compatibility but unused in static-spec mode
			utils.LavaFormatInfo("lavad Binary Version: " + protocoltypes.DefaultVersion.ConsumerTarget)
			rand.InitRandomSeed()
			numberOfNodeParallelConnections, err := cmd.Flags().GetUint(chainproxy.ParallelConnectionsFlag)
			if err != nil {
				utils.LavaFormatFatal("error fetching chainproxy.ParallelConnectionsFlag", err)
			}
			return startTesting(ctx, modifiedProviderEndpoints, numberOfNodeParallelConnections, staticSpecPaths)
		},
	}

	// RPCConsumer command flags (minimal set — no blockchain tx flags)
	cmdTestRPCSmartRouter.Flags().String("chain-id", "", "network chain id")
	cmdTestRPCSmartRouter.Flags().String("log-level", "info", "log level (debug|info|warn|error)")
	cmdTestRPCSmartRouter.Flags().String("log-format", "text", "log format (text|json)")
	cmdTestRPCSmartRouter.Flags().String("node", "", "node RPC endpoint")
	cmdTestRPCSmartRouter.Flags().String("from", "", "account key name")
	cmdTestRPCSmartRouter.Flags().Uint(chainproxy.ParallelConnectionsFlag, chainproxy.NumberOfParallelConnections, "parallel connections")
	cmdTestRPCSmartRouter.Flags().Bool(chainproxy.GRPCAllowInsecureConnection, false, "used to test grpc, to allow insecure (self signed cert).")
	cmdTestRPCSmartRouter.Flags().Bool(chainproxy.GRPCUseTls, true, "use tls configuration for grpc connections to your consumer")
	cmdTestRPCSmartRouter.Flags().StringArray(commonlib.UseStaticSpecFlag, nil, "load specs from file, directory, or remote URL — required in smart-router mode (same paths as rpcsmartrouter --use-static-spec)")
	return cmdTestRPCSmartRouter
}
