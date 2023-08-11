package lavavisor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/app"
	processmanager "github.com/lavanet/lava/ecosystem/lavavisor/pkg/process"
	lvstatetracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg/state"
	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/utils"
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

		// Build path to ./lavavisor
		lavavisorPath, err := lvutil.GetLavavisorPath(dir)
		if err != nil {
			return err
		}

		// tracker initialization
		ctx := context.Background()
		clientCtx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}
		txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())

		// initialize lavavisor state tracker
		lavavisorChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
		lavavisorStateTracker, err := lvstatetracker.NewLavaVisorStateTracker(ctx, txFactory, clientCtx, lavavisorChainFetcher)
		if err != nil {
			return err
		}

		// 1- check lava-protocol version from consensus
		protocolConsensusVersion, err := lavavisorStateTracker.GetProtocolVersion(ctx)
		if err != nil {
			return utils.LavaFormatError("protcol version cannot be fetched from consensus", err)
		}

		utils.LavaFormatInfo("Initializing the environment", utils.Attribute{Key: "Version", Value: protocolConsensusVersion.ProviderMin})

		// 2- search extracted directory inside ./lavad/upgrades/<fetched_version>
		// first check target version, then check min version
		versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+protocolConsensusVersion.ProviderMin)
		binaryPath := filepath.Join(versionDir, "lava-protocol")

		// check auto-download flag
		autoDownload, err := cmd.Flags().GetBool("auto-download")
		if err != nil {
			return err
		}

		// check if version directory exists
		if _, err := os.Stat(versionDir); os.IsNotExist(err) {
			if autoDownload {
				utils.LavaFormatInfo("Version directory does not exist, but auto-download is enabled. Attempting to download binary from GitHub...")
				os.MkdirAll(versionDir, os.ModePerm) // before downloading, ensure version directory exists
				err = processmanager.FetchAndBuildFromGithub(protocolConsensusVersion.ProviderMin, versionDir)
				if err != nil {
					utils.LavaFormatError("Failed to auto-download binary from GitHub\n ", err)
					os.Exit(1)
				}
			} else {
				utils.LavaFormatError("Sub-directory for version not found in lavavisor", nil, utils.Attribute{Key: "version", Value: protocolConsensusVersion.ProviderMin})
				os.Exit(1)
			}
			// ToDo: add checkLavaProtocolVersion after version flag is added to release
			//
		} else {
			vm := processmanager.VersionMonitor{
				BinaryPath:    binaryPath,
				LavavisorPath: lavavisorPath,
			}
			err = vm.ValidateProtocolVersion(protocolConsensusVersion)
			if err != nil {
				if autoDownload {
					utils.LavaFormatInfo("Version mismatch or binary not found, but auto-download is enabled. Attempting to download binary from GitHub...")
					err = processmanager.FetchAndBuildFromGithub(protocolConsensusVersion.ProviderMin, versionDir)
					if err != nil {
						utils.LavaFormatError("Failed to auto-download binary from GitHub\n ", err)
						os.Exit(1)
					}
				} else {
					utils.LavaFormatError("Protocol version mismatch or binary not found in lavavisor directory\n ", err)
					os.Exit(1)
				}
				// ToDo: add checkLavaProtocolVersion after version flag is added to release
				//
			}
		}
		utils.LavaFormatInfo("Protocol binary with target version has been successfully set!")

		// 3- if found: create a link from that binary to $(which lava-protocol)
		out, err := exec.LookPath("lava-protocol")
		if err != nil {
			// if "which" command fails, copy binary to system path
			gobin, err := exec.Command("go", "env", "GOPATH").Output()
			if err != nil {
				utils.LavaFormatFatal("couldn't determine Go binary path", err)
			}

			goBinPath := strings.TrimSpace(string(gobin)) + "/bin/"

			// Check if the fetched binary is executable
			// ToDo: change flag to "--version" once relased binaries support the flag
			_, err = exec.Command(binaryPath, "--help").Output()
			if err != nil {
				utils.LavaFormatFatal("binary is not a valid executable: ", err)
			}

			// Check if the link already exists and remove it
			lavaLinkPath := goBinPath + "lava-protocol"
			if _, err := os.Lstat(lavaLinkPath); err == nil {
				utils.LavaFormatInfo("Discovered an existing link. Attempting to refresh.")
				err = os.Remove(lavaLinkPath)
				if err != nil {
					utils.LavaFormatFatal("couldn't remove existing link", err)
				}
			} else if !os.IsNotExist(err) {
				// other error
				utils.LavaFormatFatal("unexpected error when checking for existing link", err)
			}
			utils.LavaFormatInfo("Old binary link successfully removed. Attempting to create the updated link.")

			err = lvutil.Copy(binaryPath, goBinPath+"lava-protocol")
			if err != nil {
				utils.LavaFormatFatal("couldn't copy binary to system path", err)
			}

			// try "which" command again
			out, err = exec.LookPath("lava-protocol")
			if err != nil {
				utils.LavaFormatFatal("couldn't find the binary in the system path", err)
			} else {
				utils.LavaFormatInfo("Found binary at:", utils.Attribute{Key: "Path", Value: out})
			}
		}
		dest := strings.TrimSpace(string(out))

		if _, err := os.Lstat(dest); err == nil {
			// if destination file exists, remove it
			err = os.Remove(dest)
			if err != nil {
				utils.LavaFormatFatal("couldn't remove existing link", err)
			}
		}

		err = os.Symlink(binaryPath, dest)
		if err != nil {
			utils.LavaFormatFatal("couldn't create symbolic link", err)
		}

		// check that the link has been established
		link, err := os.Readlink(dest)
		if err != nil || link != binaryPath {
			utils.LavaFormatFatal("failed to verify symbolic link", err)
		}

		utils.LavaFormatInfo("Symbolic link created successfully.", utils.Attribute{Key: "Linked binary path", Value: dest})

		// ToDo: if autodownload false: alert user that binary is not exist, monitor directory constantly!,

		return nil
	},
}

func init() {
	// Use the log directory in flags
	flags.AddQueryFlagsToCmd(cmdLavavisorInit)
	cmdLavavisorInit.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorInit.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorInit.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	rootCmd.AddCommand(cmdLavavisorInit)
}
