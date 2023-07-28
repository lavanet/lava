package lavavisor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
