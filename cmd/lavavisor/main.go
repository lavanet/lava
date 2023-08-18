package main

import (
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/cmd/lavad/cmd"
	lvcmd "github.com/lavanet/lava/ecosystem/lavavisor/cmd"
)

func main() {
	rootCmd := cmd.NewLavaVisorRootCmd()
	// lavavisor init cobra command
	cmdLavavisorInit := lvcmd.CreateLavaVisorInitCobraCommand()
	// lavavisor start cobra command
	cmdLavavisorStart := lvcmd.CreateLavaVisorStartCobraCommand()
	// lavavisor start cobra command
	cmdLavavisorCreateService := lvcmd.CreateLavaVisorCreateServiceCobraCommand()

	// Add Lavavisor Init
	rootCmd.AddCommand(cmdLavavisorInit)
	// Add Lavavisor Start
	rootCmd.AddCommand(cmdLavavisorStart)
	// Add Lavavisor Create Service
	rootCmd.AddCommand(cmdLavavisorCreateService)

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
