package processmanager

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	protocolVersion "github.com/lavanet/lava/protocol/upgrade"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

func ReloadDaemon() error {
	cmd := exec.Command("sudo", "systemctl", "daemon-reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return utils.LavaFormatError("Failed to run command", err, utils.Attribute{Key: "Output", Value: output})
	}
	return nil
}

func StartProcess(process string) error {
	// Extract the chain id from the process string
	utils.LavaFormatInfo("Starting Process", utils.Attribute{Key: "process", Value: process})

	// Create command list
	cmds := []*exec.Cmd{
		exec.Command("sudo", "systemctl", "enable", process),
		exec.Command("sudo", "systemctl", "restart", process),
		exec.Command("sudo", "systemctl", "status", process),
	}

	// Run the commands and capture their output
	for _, cmd := range cmds {
		utils.LavaFormatInfo("Running", utils.Attribute{Key: "command", Value: strings.Join(cmd.Args, " ")})
		output, err := cmd.CombinedOutput()
		if err != nil {
			return utils.LavaFormatError("Failed to run command", err, utils.Attribute{Key: "Output", Value: output})
		}
		utils.LavaFormatInfo("Successfully run command", utils.Attribute{Key: "cmd", Value: cmd})
		if len(output) != 0 {
			utils.LavaFormatInfo("Command Output", utils.Attribute{Key: "out", Value: output})
		}
	}
	return nil
}

func GetBinaryVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", utils.LavaFormatWarning("GetBinaryVersion failed to execute command, lavavisor will try to fetch version from github", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func ValidateMismatch(incoming *protocoltypes.Version, current string) bool {
	return (protocolVersion.HasVersionMismatch(incoming.ConsumerMin, current) ||
		protocolVersion.HasVersionMismatch(incoming.ProviderMin, current) ||
		protocolVersion.HasVersionMismatch(incoming.ConsumerTarget, current) ||
		protocolVersion.HasVersionMismatch(incoming.ProviderTarget, current))
}

func GetHomePath() (string, error) {
	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		return homeDir, nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return "", utils.LavaFormatError("Unable to get current user", err)
	}
	return currentUser.HomeDir, nil
}

func AddGoPathToDollarPath(path string) error {
	// Get the current PATH
	currentPath := os.Getenv("PATH")
	// Check if the default Go bin path is already in the PATH
	if strings.Contains(currentPath, path) {
		utils.LavaFormatInfo("Validation completed successfully - Default Go bin path already exists in PATH")
		return nil
	}
	utils.LavaFormatInfo("Adding Path to $PATH", utils.Attribute{Key: "path", Value: path})
	// Append the default Go bin path to the existing PATH
	newPath := fmt.Sprintf("%s:%s", currentPath, path)
	// Set the updated PATH
	err := os.Setenv("PATH", newPath)
	return err
}
