package main

import (
	"fmt"
	"os"

	"github.com/lavanet/lava/relayer"
	"github.com/spf13/cobra"
)

func main() {
	var cmdServer = &cobra.Command{
		Use:   "server",
		Short: "server",
		Long:  `server`,
		Run: func(cmd *cobra.Command, args []string) {
			relayer.Server()
		},
	}

	var cmdTestClient = &cobra.Command{
		Use:   "test_client",
		Short: "test client",
		Long:  `test client`,
		Run: func(cmd *cobra.Command, args []string) {
			relayer.TestClient()
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
