package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/relayer"
	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
	"github.com/lavanet/lava/spec"
	"github.com/lavanet/lava/utils"
	specpb "github.com/lavanet/lava/x/spec/types"
	"github.com/pkg/errors"
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

	var cmdServer = &cobra.Command{
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
			relayer.PortalServer(ctx, clientCtx, listenAddr, chainID, apiInterface, cmd.Flags())

			return nil
		},
	}

	var cmdTestClient = &cobra.Command{
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

			relayer.TestClient(ctx, clientCtx, chainID, apiInterface, duration, cmd.Flags())

			return nil
		},
	}

	var cmdSpec = &cobra.Command{
		Use:     "spec [grpc-addr] [path]",
		Short:   "Fetch spec and save it to the path locally",
		Example: "spec 127.0.0.1:9000 /home/.lavad/spec/spec.json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx := context.Background()
			conn := grpcutil.MustDial(ctx, args[0])
			specManager := spec.NewSpecManager(conn)

			fetchCtx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			var pbSpec *specpb.Spec
			if pbSpec, err = specManager.Fetch(fetchCtx); err != nil {
				return errors.Wrap(err, "specManager.Fetch()")
			}

			if err = specManager.Save(pbSpec, args[1]); err != nil {
				return errors.Wrap(err, "specManager.Save()")
			}

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmdServer)
	cmdServer.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmdPortalServer)
	cmdPortalServer.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmdTestClient)
	cmdTestClient.MarkFlagRequired(flags.FlagFrom)
	cmdTestClient.Flags().Bool("secure", false, "secure sends reliability on every message")
	cmdPortalServer.Flags().Bool("secure", false, "secure sends reliability on every message")
	rootCmd.AddCommand(cmdServer)
	rootCmd.AddCommand(cmdPortalServer)
	rootCmd.AddCommand(cmdTestClient)
	rootCmd.AddCommand(cmdSpec)

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
