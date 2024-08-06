package main

import (
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/lavanet/lava/v2/app"
	cmdcommon "github.com/lavanet/lava/v2/cmd/common"
	"github.com/lavanet/lava/v2/cmd/lavad/cmd"
	"github.com/lavanet/lava/v2/protocol/badgegenerator"
	"github.com/lavanet/lava/v2/protocol/rpcconsumer"
	"github.com/lavanet/lava/v2/protocol/rpcprovider"
	"github.com/lavanet/lava/v2/protocol/statetracker"
	utilscli "github.com/lavanet/lava/v2/utils/cli"
	"github.com/spf13/cobra"
)

const (
	DefaultRPCConsumerFileName = "rpcconsumer.yml"
	DefaultRPCProviderFileName = "rpcprovider.yml"
)

func main() {
	rootCmd, _ := cmd.NewRootCmd()

	// rpc consumer cobra command
	cmdRPCConsumer := rpcconsumer.CreateRPCConsumerCobraCommand()
	// rpc provider cobra command
	cmdRPCProvider := rpcprovider.CreateRPCProviderCobraCommand()
	// badge generator cobra command
	badgeGenerator := badgegenerator.CreateBadgeGeneratorCobraCommand()

	deprecationWarningMessage := "This command will become deprecated on this binary, in the future. Please switch to the 'lavap' binary for this command."
	// Add RPC Consumer Command
	rootCmd.AddCommand(cmdcommon.CreateWarningLogCommandWrapper(cmdRPCConsumer, deprecationWarningMessage))
	// Add RPC Provider Command
	rootCmd.AddCommand(cmdcommon.CreateWarningLogCommandWrapper(cmdRPCProvider, deprecationWarningMessage))
	// Add Badge Generator Command
	rootCmd.AddCommand(cmdcommon.CreateWarningLogCommandWrapper(badgeGenerator, deprecationWarningMessage))

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test commands for protocol network",
	}
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(rpcconsumer.CreateTestRPCConsumerCobraCommand())
	testCmd.AddCommand(rpcprovider.CreateTestRPCProviderCobraCommand())
	testCmd.AddCommand(statetracker.CreateEventsCobraCommand())
	testCmd.AddCommand(utilscli.NewMultiSendTxCmd())
	testCmd.AddCommand(utilscli.NewQueryTotalGasCmd())

	cmd.OverwriteFlagDefaults(rootCmd, map[string]string{
		flags.FlagChainID:       strings.ReplaceAll(app.Name, "-", ""),
		flags.FlagGasAdjustment: statetracker.DefaultGasAdjustment,
	})

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
