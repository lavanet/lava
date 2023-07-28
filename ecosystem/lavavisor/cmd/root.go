package lavavisor

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.1"

var rootCmd = &cobra.Command{
	Use:   "lavavisor",
	Short: `LavaVisor is a protocol upgrade manager for Lava protocol binaries.`,
	Long: `LavaVisor is a protocol upgrade manager designed to orchestrate and automate
		the process of protocol version upgrades.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "There was an error while running LavaVisor CLI '%s'", err)
		os.Exit(1)
	}
}
