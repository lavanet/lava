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
	"github.com/lavanet/lava/relayer"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
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

	cmdRPCClient := &cobra.Command{
		Use:     "rpcclient [listen-ip]:[listen-port],[spec-chain-id],[api-interface] repeat...",
		Short:   "rpcconsumer sets up a server to perform api requests and sends them through the lava protocol to data providers",
		Long:    `rpcconsumer sets up a server to perform api requests and sends them through the lava protocol to data providers`,
		Example: `rpcclient 127.0.0.1:3333,COS3,tendermintrpc 127.0.0.1:3334,COS3,rest`,
		Args:    cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("Gateway Proxy process started", &map[string]string{"args": strings.Join(args, ",")})
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// decide on cli input method
			// write good documentation for usage
			// parse requested inputs into RPCEndpoint list
			// handle flags, pass necessary fields
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithChainID(networkChainId)
			rpcConsumer := rpcconsumer.RPCConsumer{}
			rpcEndpoints := []*rpcconsumer.RPCEndpoint{}

			rpcConsumer.Start(ctx, txFactory, clientCtx, rpcEndpoints)
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
	rootCmd.AddCommand(cmdServer)
	rootCmd.AddCommand(cmdPortalServer)
	rootCmd.AddCommand(cmdTestClient)

	// RPCConsumer command flags
	flags.AddTxFlagsToCmd(cmdRPCClient)
	cmdRPCClient.MarkFlagRequired(flags.FlagFrom)
	cmdRPCClient.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdRPCClient.Flags().Uint64(sentry.GeolocationFlag, 0, "geolocation to run from")
	cmdRPCClient.MarkFlagRequired(sentry.GeolocationFlag)
	cmdRPCClient.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdRPCClient.Flags().String(performance.PprofAddressFlagName, "", "pprof server address, used for code profiling")
	cmdRPCClient.Flags().String(performance.CacheFlagName, "", "address for a cache server to improve performance")
	// rootCmd.AddCommand(cmdRPCClient) // Remove this when ready

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
