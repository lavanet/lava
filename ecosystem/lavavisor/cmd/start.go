package lavavisor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ProviderServices []string `yaml:"provider-services"`
}
type ProviderProcess struct {
	Name      string
	ChainID   string
	IsRunning bool
}

var providers []*ProviderProcess

var cmdLavavisorStart = &cobra.Command{
	Use:   "start",
	Short: "A command that will start provider processes given with config.yml",
	Long: `A command that will start provider processes given with config.yml and starts 
    lavavisor listening process. It reads config.yaml, checks the list of provider-services, 
    and starts them with the linked 'which lava-protocol' binary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("directory")
		dir, err := lvutil.ExpandTilde(dir)
		if err != nil {
			return utils.LavaFormatError("unable to expand directory path", err)
		}
		// Build path to ./lavavisor
		configPath := filepath.Join(dir+"/.lavavisor", "/config.yml")

		// Read config.yaml
		configData, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config.yaml: %v", err)
		}

		var config Config
		err = yaml.Unmarshal(configData, &config)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config.yaml: %v", err)
		}

		// Iterate over the list of provider services and start them
		for _, provider := range config.ProviderServices {
			fmt.Printf("Starting provider: %s\n", provider)
			// TODO: Implement the actual starting of the providers
			startProvider(provider)
		}
		return nil
	},
}

func init() {
	cmdLavavisorStart.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	rootCmd.AddCommand(cmdLavavisorStart)
}
func startProvider(provider string) {
	// Extract the chain id from the provider string
	chainID := strings.Split(provider, "-")[1]
	fmt.Println("chainId: ", chainID)

	// Create command list
	cmds := []*exec.Cmd{
		exec.Command("sudo", "systemctl", "daemon-reload"),
		exec.Command("sudo", "systemctl", "enable", provider+".service"),
		exec.Command("sudo", "systemctl", "restart", provider+".service"),
		exec.Command("sudo", "systemctl", "status", provider+".service"),
	}

	// Start the command in a separate goroutine so that it doesn't block the main thread
	go func() {
		fmt.Printf("Starting provider: %s\n", provider)

		// Run the commands and capture their output
		for _, cmd := range cmds {
			if output, err := cmd.CombinedOutput(); err != nil {
				fmt.Printf("Failed to run command: %s, Error: %s\n", cmd, err)
				fmt.Printf("Command Output: \n%s\n", output)
			} else {
				fmt.Printf("Successfully run command: %s\n", cmd)
				fmt.Printf("Command Output: \n%s\n", output)
			}
		}

	}()

	// Add to the list of providers
	providers = append(providers, &ProviderProcess{
		Name:      provider,
		ChainID:   chainID,
		IsRunning: true,
	})
}
