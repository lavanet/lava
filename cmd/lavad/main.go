package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "net/http/pprof"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/protocol/rpcconsumer"
	"github.com/lavanet/lava/protocol/rpcprovider"
	"github.com/lavanet/lava/relayer"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
)

const (
	DefaultRPCConsumerFileName = "rpcconsumer.yml"
	DefaultRPCProviderFileName = "rpcprovider.yml"
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

	// rpc consumer cobra command
	cmdRPCConsumer := rpcconsumer.CreateRPCConsumerCobraCommand()
	// rpc provider cobra command
	cmdRPCProvider := rpcprovider.CreateRPCProviderCobraCommand()

	// Server command flags
	flags.AddTxFlagsToCmd(cmdServer)
	cmdServer.MarkFlagRequired(flags.FlagFrom)
	cmdServer.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdServer.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdServer.MarkFlagRequired(sentry.GeolocationFlag)
	cmdServer.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	cmdServer.Flags().Uint(chainproxy.ParallelConnectionsFlag, chainproxy.NumberOfParallelConnections, "parallel connections")
	cmdServer.Flags().String(chainproxy.TendermintProviderHttpEndpoint, "", "The http endpoint when starting a Tendermint Provider process, otherwise leave empty")
	rootCmd.AddCommand(cmdServer)

	// Portal Server command flags
	flags.AddTxFlagsToCmd(cmdPortalServer)
	cmdPortalServer.MarkFlagRequired(flags.FlagFrom)
	cmdPortalServer.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdPortalServer.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdPortalServer.MarkFlagRequired(sentry.GeolocationFlag)
	cmdPortalServer.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdPortalServer.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdPortalServer.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	rootCmd.AddCommand(cmdPortalServer)

	// Test Client command flags
	flags.AddTxFlagsToCmd(cmdTestClient)
	cmdTestClient.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdTestClient.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdTestClient.MarkFlagRequired(sentry.GeolocationFlag)
	cmdTestClient.MarkFlagRequired(flags.FlagFrom)
	cmdTestClient.Flags().Bool("secure", false, "secure sends reliability on every message")
	rootCmd.AddCommand(cmdTestClient)

	// Add RPC Consumer Command
	rootCmd.AddCommand(cmdRPCConsumer)
	// Add RPC Provider Command
	rootCmd.AddCommand(cmdRPCProvider)

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
