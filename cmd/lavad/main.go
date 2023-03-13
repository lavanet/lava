package main

import (
	"context"
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
