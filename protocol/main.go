package main

import (
	_ "net/http/pprof"
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/cmd/lavad/cmd"
	"github.com/lavanet/lava/protocol/badgegenerator"
	"github.com/lavanet/lava/protocol/rpcconsumer"
	"github.com/lavanet/lava/protocol/rpcprovider"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/spf13/cobra"
)

const (
	DefaultRPCConsumerFileName = "rpcconsumer.yml"
	DefaultRPCProviderFileName = "rpcprovider.yml"
)

func main() {
	rootCmd := cmd.NewLavaProtocolRootCmd()

	// rpc consumer cobra command
	cmdRPCConsumer := rpcconsumer.CreateRPCConsumerCobraCommand()
	// rpc provider cobra command
	cmdRPCProvider := rpcprovider.CreateRPCProviderCobraCommand()
	// badge generator cobra command
	badgeGenerator := badgegenerator.CreateBadgeGeneratorCobraCommand()

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
	testCmd.AddCommand(rpcprovider.CreateTestRPCProviderCACertificateCobraCommand())
	testCmd.AddCommand(statetracker.CreateEventsCobraCommand())
	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}

}
