package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/relayer"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/starport/starport/pkg/cosmoscmd"
)

//
// TODO: https://docs.cosmos.network/master/architecture/adr-027-deterministic-protobuf-serialization.html

func main() {
	rootCmd, _ := cosmoscmd.NewRootCmd(
		"relayer",
		app.AccountAddressPrefix,
		app.DefaultNodeHome,
		app.Name,
		app.ModuleBasics,
		app.New,
	)

	var cmdServer = &cobra.Command{
		Use:   "server [listen-ip] [listen-port] [node-url] [node-spec]",
		Short: "server",
		Long:  `server`,
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			port, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			specId, err := strconv.Atoi(args[3])
			if err != nil {
				return err
			}

			listenAddr := fmt.Sprintf("%s:%d", args[0], port)
			ctx := context.Background()
			relayer.Server(ctx, clientCtx, queryClient, listenAddr, args[2], specId)

			return nil
		},
	}

	var cmdTestClient = &cobra.Command{
		Use:   "test_client [listen-ip] [listen-port]",
		Short: "test client",
		Long:  `test client`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			port, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			listenAddr := fmt.Sprintf("%s:%d", args[0], port)
			ctx := context.Background()
			relayer.TestClient(ctx, clientCtx, queryClient, listenAddr)

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmdServer)
	cmdServer.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmdTestClient)
	cmdTestClient.MarkFlagRequired(flags.FlagFrom)

	rootCmd.AddCommand(cmdServer)
	rootCmd.AddCommand(cmdTestClient)

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
