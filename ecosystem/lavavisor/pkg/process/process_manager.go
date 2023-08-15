package processmanager

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lavanet/lava/utils"
)

type ServiceProcess struct {
	Name    string
	ChainID string
}

func StartProcess(processes []*ServiceProcess, process string) []*ServiceProcess {
	// Extract the chain id from the process string
	chainID := strings.Split(process, "-")[1]

	// Create command list
	cmds := []*exec.Cmd{
		exec.Command("sudo", "systemctl", "daemon-reload"),
		exec.Command("sudo", "systemctl", "enable", process+".service"),
		exec.Command("sudo", "systemctl", "restart", process+".service"),
		exec.Command("sudo", "systemctl", "status", process+".service"),
	}

	// Run the commands and capture their output
	for _, cmd := range cmds {
		utils.LavaFormatInfo("Running", utils.Attribute{Key: "command", Value: strings.Join(cmd.Args, " ")})
		output, err := cmd.CombinedOutput()
		if err != nil {
			utils.LavaFormatError("Failed to run command", err, utils.Attribute{Key: "Output", Value: output})
			return nil
		}
		fmt.Printf("Successfully run command: %s\n", cmd)
		if len(output) != 0 {
			fmt.Printf("Command Output: \n%s\n", output)
		}
	}
	// Add to the list of services
	processes = append(processes, &ServiceProcess{
		Name:    process,
		ChainID: chainID,
	})
	return processes
}

func getBinaryVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", utils.LavaFormatError("failed to execute command", err)
	}
	return strings.TrimSpace(string(output)), nil
}
