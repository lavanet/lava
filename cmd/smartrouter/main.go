package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/lavanet/lava/v5/protocol/performance/connection"
	"github.com/lavanet/lava/v5/protocol/rpcsmartrouter"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "smart-router",
		Short: "Lava Smart Router — centralized RPC routing engine",
	}

	cmdVersion := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(protocoltypes.DefaultVersion.ConsumerTarget)
		},
	}

	// The rpcsmartrouter command is the primary operational command.
	cmdRPCSmartRouter := rpcsmartrouter.CreateRPCSmartRouterCobraCommand()

	rootCmd.AddCommand(cmdVersion)
	rootCmd.AddCommand(cmdRPCSmartRouter)

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test commands for the smart router",
	}
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(rpcsmartrouter.CreateTestRPCSmartRouterCobraCommand())
	testCmd.AddCommand(connection.CreateTestConnectionServerCobraCommand())
	testCmd.AddCommand(connection.CreateTestConnectionProbeCobraCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
