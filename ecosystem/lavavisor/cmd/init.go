package lavavisor

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	lvstatetracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
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

		clientCtx, err := client.GetClientQueryContext(cmd)
		fmt.Println("cmd: ", cmd)
		fmt.Println("clientCtx-1: ", clientCtx)
		if err != nil {
			return err
		}
		clientCtx = clientCtx.WithChainID("lava")
		fmt.Println("clientCtx-2: ", clientCtx)

		sq := lvstatetracker.NewStateQuery(clientCtx)
		protoVer, err := sq.GetProtocolVersion(ctx)
		if err != nil {
			return utils.LavaFormatError("protcol version cannot be fetched from consensus", err)
		}
		version := protocoltypes.Version{
			ConsumerTarget: protoVer.ConsumerTarget,
			ConsumerMin:    protoVer.ConsumerMin,
			ProviderTarget: protoVer.ProviderTarget,
			ProviderMin:    protoVer.ProviderMin,
		}
		fmt.Println("version.ConsumerTarget: ", version.ConsumerTarget)
		fmt.Println("version.ConsumerMin: ", version.ConsumerMin)
		fmt.Println("version.ProviderTarget: ", version.ProviderTarget)
		fmt.Println("version.ProviderMin: ", version.ProviderMin)

		// 2- search extracted directory inside ./lavad/upgrades/<fetched_version>
		// first check target version, then check min version
		versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+version.ConsumerMin)
		// check if version directory exists
		if _, err := os.Stat(versionDir); os.IsNotExist(err) {
			fmt.Printf("Error: Version directory does not exist: %s\n", versionDir)
			os.Exit(1)
		}

		binaryPath := filepath.Join(versionDir, "lava-protocol")

		err = checkLavaProtocolVersion(version.ConsumerMin, binaryPath)
		if err != nil {
			utils.LavaFormatError("Wrong protocol version found in the version directory\n ", err)
			os.Exit(1)
		}

		fmt.Println("Version check passed")

		// 3- if found: create a link from that binary to $(which lava-protocol)
		out, err := exec.Command("which", "lava-protocol").Output()
		if err != nil {
			// if "which" command fails, copy binary to system path
			gobin, err := exec.Command("go", "env", "GOPATH").Output()
			if err != nil {
				utils.LavaFormatFatal("couldn't determine Go binary path", err)
			}

			goBinPath := strings.TrimSpace(string(gobin)) + "/bin/"
			fmt.Println("goBinPath: ", goBinPath)
			err = Copy(binaryPath, goBinPath+"lava-protocol")
			if err != nil {
				utils.LavaFormatFatal("couldn't copy binary to system path", err)
			}

			// try "which" command again
			out, err = exec.Command("which", "lava-protocol").Output()
			if err != nil {
				utils.LavaFormatFatal("couldn't extract binary at the system path", err)
			}
		}
		fmt.Println("string(out): ", string(out))
		dest := strings.TrimSpace(string(out))
		fmt.Println("dest: ", dest)

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

		fmt.Println("Symbolic link created successfully")

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
	flags.AddQueryFlagsToCmd(cmdLavavisorInit)
	cmdLavavisorInit.MarkFlagRequired(flags.FlagFrom)
	cmdLavavisorInit.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorInit.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorInit.Flags().String("from", "", "Name or address of private key with which to sign")
	rootCmd.AddCommand(cmdLavavisorInit)
}

func getBinaryVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}

	// Assume the output format is "lava-protocol version x.x.x"
	version := strings.Split(string(output), " ")[2]
	fmt.Println("getBinaryVersion - version: ", version)
	return version, nil
}

func checkLavaProtocolVersion(targetVersion, binaryPath string) error {
	binaryVersion, err := getBinaryVersion(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to get binary version: %w", err)
	}

	if strings.TrimSpace(binaryVersion) != strings.TrimSpace(targetVersion) {
		return fmt.Errorf("version mismatch, expected %s but got %s", targetVersion, binaryVersion)
	}

	return nil
}

func Copy(src, dest string) error {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return fmt.Errorf("couldn't read source file: %v", err)
	}

	err = ioutil.WriteFile(dest, input, 0755)
	if err != nil {
		return fmt.Errorf("couldn't write destination file: %v", err)
	}
	return nil
}
