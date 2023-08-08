package lvutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lavanet/lava/utils"
)

type ProviderProcess struct {
	Name      string
	ChainID   string
	IsRunning bool
}

func ExpandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", utils.LavaFormatError("cannot get user home directory", err)
	}
	return filepath.Join(home, path[1:]), nil
}

func GetLavavisorPath(dir string) (lavavisorPath string, err error) {
	dir, err = ExpandTilde(dir)
	if err != nil {
		return "", utils.LavaFormatError("unable to expand directory path", err)
	}
	// Build path to ./lavavisor
	lavavisorPath = filepath.Join(dir, "./.lavavisor")

	// Check if ./lavavisor directory exists
	if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
		// ToDo: handle case where user didn't set up the file
		return "", utils.LavaFormatError("lavavisor directory does not exist at path", err, utils.Attribute{Key: "lavavisorPath", Value: lavavisorPath})
	}

	return lavavisorPath, nil
}

func Copy(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return utils.LavaFormatError("couldn't read source file", err)
	}

	err = os.WriteFile(dest, input, 0o755)
	if err != nil {
		return utils.LavaFormatError("couldn't write destination file", err)
	}
	return nil
}

func StartProvider(providers *[]*ProviderProcess, provider string) {
	// Extract the chain id from the provider string
	chainID := strings.Split(provider, "-")[1]

	// Create command list
	cmds := []*exec.Cmd{
		exec.Command("sudo", "systemctl", "daemon-reload"),
		exec.Command("sudo", "systemctl", "enable", provider+".service"),
		exec.Command("sudo", "systemctl", "restart", provider+".service"),
		exec.Command("sudo", "systemctl", "status", provider+".service"),
	}

	// Run the commands and capture their output
	for _, cmd := range cmds {
		fmt.Printf("Running command: %s\n", strings.Join(cmd.Args, " "))
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to run command: %s, Error: %s\n", cmd, err)
			fmt.Printf("Command Output: \n%s\n", output)
			return
		} else {
			fmt.Printf("Successfully run command: %s\n", cmd)
			fmt.Printf("Command Output: \n%s\n", output)
		}
	}

	// Add to the list of providers
	*providers = append(*providers, &ProviderProcess{
		Name:      provider,
		ChainID:   chainID,
		IsRunning: true,
	})
}
