package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/lavanet/lava/v5/ecosystem/cache"
	"github.com/lavanet/lava/v5/protocol/performance/connection"
	"github.com/lavanet/lava/v5/protocol/rpcsmartrouter"
	protocoltypes "github.com/lavanet/lava/v5/types/protocol"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "lavap",
		Short: "Lava Protocol Smart Router",
	}

	cmdVersion := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(protocoltypes.DefaultVersion.ConsumerTarget)
		},
	}

	cmdRPCSmartRouter := rpcsmartrouter.CreateRPCSmartRouterCobraCommand()

	rootCmd.AddCommand(cmdVersion)
	rootCmd.AddCommand(cmdRPCSmartRouter)
	rootCmd.AddCommand(cache.CreateCacheCobraCommand())

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test commands for protocol network",
	}
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(rpcsmartrouter.CreateTestRPCSmartRouterCobraCommand())
	testCmd.AddCommand(connection.CreateTestConnectionServerCobraCommand())
	testCmd.AddCommand(connection.CreateTestConnectionProbeCobraCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
