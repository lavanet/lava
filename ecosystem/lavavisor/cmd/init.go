package lavavisor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	lvstatetracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg"
	"github.com/spf13/cobra"
)

var cmdLavavisorInit = &cobra.Command{
	Use:   "init",
	Short: "initializes the environment for LavaVisor",
	Long: `Prepares the local environment for the operation of LavaVisor.
	config.yml should be located in the ./lavavisor/ directory.`,
	Args: cobra.ExactArgs(0),
	Example: `optional flags: --directory | --auto-download 
		lavavisor init <flags>
		lavavisor init --directory ./custom/lavavisor/path 
		lavavisor init --directory ./custom/lavavisor/path --auto-download true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("directory")
		dir, err := expandTilde(dir)
		if err != nil {
			return fmt.Errorf("unable to expand directory path: %w", err)
		}
		fmt.Println("dir: ", dir)
		// Build path to ./lavavisor
		lavavisorPath := filepath.Join(dir, "./.lavavisor")

		fmt.Println("lavavisorPath: ", lavavisorPath)

		// Check if ./lavavisor directory exists
		if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
			// ToDo: handle case where user didn't set up the file
			return fmt.Errorf("lavavisor directory does not exist at path: %s", lavavisorPath)
		}

		// 1- check lava-protocol version from consensus
		// ...GetProtocolVersion()
		// handle flags, pass necessary fields
		ctx := context.Background()

		clientCtx, err := client.GetClientTxContext(cmd)
		if err != nil {
			return err
		}
		clientCtx = clientCtx.WithChainID("LAV1")

		sq := lvstatetracker.NewStateQuery(clientCtx)
		protoVer, err := sq.GetProtocolVersion(ctx)
		if err != nil {
			return fmt.Errorf("protcol version cannot be fetched from consensus")
		}
		fmt.Println("protoVer: ", protoVer)

		// 2- search extracted directory inside ./lavad/upgrades/<fetched_version>

		// 3- if found: create a link from that binary to $(which lava-protocol)

		// 4- if not found: check-auto download flag

		// 5- if auto-download flag is true: initiate fetchFromGithub()
		//		if false: alert user that binary is not exist, monitor directory constantly!,

		return nil
	},
}

func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get user home directory: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}

func init() {
	// Use the log directory in flags
	cmdLavavisorInit.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorInit.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	rootCmd.AddCommand(cmdLavavisorInit)
}
