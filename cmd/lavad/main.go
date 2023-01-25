package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/protocol/rpcconsumer"
	"github.com/lavanet/lava/relayer"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DefaultRPCConsumerFileName = "rpcconsumer.yml"
)

func main() {
	rootCmd, _ := cosmoscmd.NewRootCmd(
		app.Name,
		app.AccountAddressPrefix,
		app.DefaultNodeHome,
		app.Name,
		app.ModuleBasics,
		app.New,
		// this line is used by starport scaffolding # root/arguments
	)

	cmdServer := &cobra.Command{
		Use:   "server [listen-ip] [listen-port] [node-url] [node-chain-id] [api-interface]",
		Short: "server",
		Long:  `server`,
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("Provider process started", &map[string]string{"args": strings.Join(args, ",")})
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			//
			// TODO: there has to be a better way to send txs
			// (cosmosclient was a fail)

			clientCtx.SkipConfirm = true
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithChainID(networkChainId)

			port, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			chainID := args[3]

			apiInterface := args[4]

			listenAddr := fmt.Sprintf("%s:%d", args[0], port)
			ctx := context.Background()
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err, nil)
			}
			utils.LoggingLevel(logLevel)
			relayer.Server(ctx, clientCtx, txFactory, listenAddr, args[2], chainID, apiInterface, cmd.Flags())

			return nil
		},
	}

	cmdPortalServer := &cobra.Command{
		Use:   "portal_server [listen-ip] [listen-port] [relayer-chain-id] [api-interface]",
		Short: "portal server",
		Long:  `portal server`,
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("Gateway Proxy process started", &map[string]string{"args": strings.Join(args, ",")})
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			port, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			chainID := args[2]
			apiInterface := args[3]

			listenAddr := fmt.Sprintf("%s:%d", args[0], port)
			ctx := context.Background()
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err, nil)
			}
			utils.LoggingLevel(logLevel)

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

			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithChainID(networkChainId)

			relayer.PortalServer(ctx, clientCtx, txFactory, listenAddr, chainID, apiInterface, cmd.Flags())

			return nil
		},
	}

	cmdTestClient := &cobra.Command{
		Use:   "test_client [chain-id] [api-interface] [duration-seconds]",
		Short: "test client",
		Long:  `test client`,
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("Test consumer process started", &map[string]string{"args": strings.Join(args, ",")})
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			chainID := args[0]

			apiInterface := args[1]

			// if duration is not set, set duration value to 1 so tests runs atleast once
			duration := int64(1)
			if len(args) == 3 {
				duration, err = strconv.ParseInt(args[2], 10, 64)
				if err != nil {
					return err
				}
			}
			ctx := context.Background()
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err, nil)
			}
			utils.LoggingLevel(logLevel)

			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithChainID(networkChainId)

			relayer.TestClient(ctx, txFactory, clientCtx, chainID, apiInterface, duration, cmd.Flags())

			return nil
		},
	}

	cmdRPCConsumer := &cobra.Command{
		Use:   "rpcconsumer [config-file] | { {listen-ip:listen-port spec-chain-id api-interface} ... }",
		Short: `rpcconsumer sets up a server to perform api requests and sends them through the lava protocol to data providers`,
		Long: `rpcconsumer sets up a server to perform api requests and sends them through the lava protocol to data providers
		all configs should be located in` + app.DefaultNodeHome + "/config or the local running directory" + ` 
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
			if len(args)%rpcconsumer.NumFieldsInConfig != 0 {
				return fmt.Errorf("invalid number of arguments, either its a single config file or repeated groups of 3 IP:PORT chain-id api-interface")
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
			var rpcEndpoints []*lavasession.RPCEndpoint
			var endpoints_strings []string
			var viper_endpoints *viper.Viper
			if len(args) > 1 {
				viper_endpoints, err = rpcconsumer.ParseEndpointArgs(args)
				if err != nil {
					return utils.LavaFormatError("invalid endpoints arguments", err, &map[string]string{"endpoint_strings": strings.Join(args, "")})
				}
				viper.MergeConfigMap(viper_endpoints.AllSettings())
				err := viper.SafeWriteConfigAs(DefaultRPCConsumerFileName)
				if err != nil {
					utils.LavaFormatInfo("did not create new config file, if it's desired remove the config file", &map[string]string{"file_name": viper.ConfigFileUsed()})
				} else {
					utils.LavaFormatInfo("created new config file", &map[string]string{"file_name": DefaultRPCConsumerFileName + ".yml"})
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
			rpcEndpoints, err = rpcconsumer.ParseEndpoints(viper.GetViper(), geolocation)
			if err != nil || len(rpcEndpoints) == 0 {
				return utils.LavaFormatError("invalid endpoints definition", err, &map[string]string{"endpoint_strings": strings.Join(endpoints_strings, "")})
			}
			// handle flags, pass necessary fields
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithChainID(networkChainId)
			rpcConsumer := rpcconsumer.RPCConsumer{}
			requiredResponses := 1 // TODO: handle secure flag, for a majority between providers
			utils.LavaFormatInfo("lavad Binary Version: "+version.Version, nil)
			rand.Seed(time.Now().UnixNano())
			vrf_sk, _, err := utils.GetOrCreateVRFKey(clientCtx)
			if err != nil {
				utils.LavaFormatFatal("failed getting or creating a VRF key", err, nil)
			}
			rpcConsumer.Start(ctx, txFactory, clientCtx, rpcEndpoints, requiredResponses, vrf_sk)
			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmdServer)
	cmdServer.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmdPortalServer)
	cmdPortalServer.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmdTestClient)

	cmdPortalServer.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdTestClient.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdServer.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdPortalServer.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdTestClient.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdServer.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdTestClient.MarkFlagRequired(sentry.GeolocationFlag)
	cmdServer.MarkFlagRequired(sentry.GeolocationFlag)
	cmdPortalServer.MarkFlagRequired(sentry.GeolocationFlag)
	cmdTestClient.MarkFlagRequired(flags.FlagFrom)
	cmdTestClient.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdPortalServer.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdPortalServer.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdPortalServer.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdServer.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdServer.Flags().Uint(chainproxy.ParallelConnectionsFlag, chainproxy.DefaultNumberOfParallelConnections, "parallel connections")

	rootCmd.AddCommand(cmdServer)
	rootCmd.AddCommand(cmdPortalServer)
	rootCmd.AddCommand(cmdTestClient)

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdRPCConsumer)
	cmdRPCConsumer.MarkFlagRequired(flags.FlagFrom)
	cmdRPCConsumer.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdRPCConsumer.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCConsumer.MarkFlagRequired(sentry.GeolocationFlag)
	cmdRPCConsumer.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdRPCConsumer.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCConsumer.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	// rootCmd.AddCommand(cmdRPCConsumer) // TODO: DISABLE COMMAND SO IT'S NOT EXPOSED ON MAIN YET

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
