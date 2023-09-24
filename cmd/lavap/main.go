package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/cmd/lavad/cmd"
	"github.com/lavanet/lava/ecosystem/cache"
	"github.com/lavanet/lava/protocol/badgegenerator"
	"github.com/lavanet/lava/protocol/rpcconsumer"
	"github.com/lavanet/lava/protocol/rpcprovider"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/protocol/upgrade"
	"github.com/spf13/cobra"
)

const (
	DefaultRPCConsumerFileName = "rpcconsumer.yml"
	DefaultRPCProviderFileName = "rpcprovider.yml"
)

func main() {
	rootCmd := cmd.NewLavaProtocolRootCmd()

	// version cobra command
	cmdVersion := versionCommand()
	// rpc consumer cobra command
	cmdRPCConsumer := rpcconsumer.CreateRPCConsumerCobraCommand()
	// rpc provider cobra command
	cmdRPCProvider := rpcprovider.CreateRPCProviderCobraCommand()
	// badge generator cobra command
	badgeGenerator := badgegenerator.CreateBadgeGeneratorCobraCommand()

	// Add Version Command
	rootCmd.AddCommand(cmdVersion)
	// Add RPC Consumer Command
	rootCmd.AddCommand(cmdRPCConsumer)
	// Add RPC Provider Command
	rootCmd.AddCommand(cmdRPCProvider)
	// Add Badge Generator Command
	rootCmd.AddCommand(badgeGenerator)

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test commands for protocol network",
	}
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(rpcconsumer.CreateTestRPCConsumerCobraCommand())
	testCmd.AddCommand(rpcprovider.CreateTestRPCProviderCobraCommand())
	testCmd.AddCommand(statetracker.CreateEventsCobraCommand())
	rootCmd.AddCommand(cache.CreateCacheCobraCommand())
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			// Print the lavap version
			version := upgrade.GetCurrentVersion()
			fmt.Println(version.ProviderVersion) // currently we have only one version.
		},
	}
}
