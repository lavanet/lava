package lavavisor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ProviderServices []string `yaml:"provider-services"`
}

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
		configData, err := ioutil.ReadFile(configPath)
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

		}

		return nil
	},
}

func init() {
	cmdLavavisorStart.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	rootCmd.AddCommand(cmdLavavisorStart)
}
