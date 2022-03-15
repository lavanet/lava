package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/lavanet/lava/relayer"
	"github.com/spf13/cobra"
)

func main() {
	var cmdServer = &cobra.Command{
		Use:   "server [listen-ip] [listen-port]",
		Short: "server",
		Long:  `server`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			port, err := strconv.Atoi(args[1])
			if err != nil {
				os.Exit(1)
			}

			listenAddr := fmt.Sprintf("%s:%d", args[0], port)
			ctx := context.Background()
			relayer.Server(ctx, listenAddr)
		},
	}

	var cmdTestClient = &cobra.Command{
		Use:   "test_client [listen-ip] [listen-port]",
		Short: "test client",
		Long:  `test client`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			port, err := strconv.Atoi(args[1])
			if err != nil {
				os.Exit(1)
			}

			listenAddr := fmt.Sprintf("%s:%d", args[0], port)
			ctx := context.Background()
			relayer.TestClient(ctx, listenAddr)
		},
	}

	var rootCmd = &cobra.Command{Use: "relayer"}
	rootCmd.AddCommand(cmdServer)
	rootCmd.AddCommand(cmdTestClient)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
