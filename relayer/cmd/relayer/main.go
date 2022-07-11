package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/relayer"
	"github.com/spf13/cobra"
)

//
// TODO: https://docs.cosmos.network/master/architecture/adr-027-deterministic-protobuf-serialization.html

func main() {
	fmt.Println(fmt.Sprintf(" ::: Relayer Started :::"))
	rootCmd, _ := cosmoscmd.NewRootCmd(
		"relayer",
		app.AccountAddressPrefix,
		app.DefaultNodeHome,
		app.Name,
		app.ModuleBasics,
		app.New,
	)

	// If envs arent set well (inexisting hosts for example) you cant ctrl+c out of this process
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println(sig)
		os.Exit(3)
	}()

	var cmdServer = &cobra.Command{
		Use:   "server [listen-ip] [listen-port] [node-url] [node-chain-id] [api-interface]",
		Short: "server",
		Long:  `server`,
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			//
			// TODO: there has to be a better way to send txs
			// (cosmosclient was a fail)
			clientCtx.SkipConfirm = true
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithChainID("lava")

			port, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			chainID := args[3]

			apiInterface := args[4]

			listenAddr := fmt.Sprintf("%s:%d", args[0], port)
			ctx := context.Background()
			relayer.Server(ctx, clientCtx, txFactory, listenAddr, args[2], chainID, apiInterface)

			return nil
		},
	}

	var cmdPortalServer = &cobra.Command{
		Use:   "portal_server [listen-ip] [listen-port] [relayer-chain-id] [api-interface]",
		Short: "portal server",
		Long:  `portal server`,
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			relayer.PortalServer(ctx, clientCtx, listenAddr, chainID, apiInterface, cmd.Flags())

			return nil
		},
	}

	var cmdTestClient = &cobra.Command{
		Use:   "test_client [chain-id] [api-interface]",
		Short: "test client",
		Long:  `test client`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			chainID := args[0]

			apiInterface := args[1]

			ctx := context.Background()

			relayer.TestClient(ctx, clientCtx, chainID, apiInterface, cmd.Flags())

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmdServer)
	cmdServer.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmdPortalServer)
	cmdPortalServer.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmdTestClient)
	cmdTestClient.MarkFlagRequired(flags.FlagFrom)
	rootCmd.AddCommand(cmdServer)
	rootCmd.AddCommand(cmdPortalServer)
	rootCmd.AddCommand(cmdTestClient)
	cmdTestClient.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdPortalServer.Flags().Bool("secure", false, "secure sends reliability on every message")
	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
