package lavavisor

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	// "os/exec"
	// "os/signal"
	"path/filepath"

	// "strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	processmanager "github.com/lavanet/lava/ecosystem/lavavisor/pkg/monitor"
	lvstatetracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg/state"
	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"

	"github.com/lavanet/lava/protocol/statetracker"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type LavavisorStateTrackerInf interface {
	RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator statetracker.VersionValidationInf)
	GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error)
}

type LavaVisor struct {
	lavavisorStateTracker LavavisorStateTrackerInf
}

type Config struct {
	ProviderServices []string `yaml:"provider-services"`
}

var providers []*processmanager.ProviderProcess

// GetProviders returns the list of providers.
func (lv *LavaVisor) GetProviders() []*processmanager.ProviderProcess {
	return providers
}

func (lv *LavaVisor) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, lavavisorPath string, autoDownload bool) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	// spawn up LavaVisor
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	lavavisorStateTracker, err := lvstatetracker.NewLavaVisorStateTracker(ctx, txFactory, clientCtx, lavaChainFetcher)
	if err != nil {
		return err
	}
	lv.lavavisorStateTracker = lavavisorStateTracker

	// check version
	version, err := lavavisorStateTracker.GetProtocolVersion(ctx)
	if err != nil {
		utils.LavaFormatFatal("failed fetching protocol version from node", err)
	}
	lavavisorStateTracker.RegisterForVersionUpdates(ctx, version, &processmanager.VersionMonitor{}) // I will pass here a lavavisor type which implements VersionValidationInf interface

	// tearing down
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("Lavavisor ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("Lavavisor signalChan")
	}

	return nil
}

// spawn up LavavisorStateTracker

// // initialize state tracker
// lavavisorChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
// lavavisorStateTracker, err := lvstatetracker.NewLavaVisorStateTracker(ctx, txFactory, clientCtx, lavavisorChainFetcher)
// if err != nil {
// 	return err
// }
// lv.lavavisorStateTracker = lavavisorStateTracker
// //  register version updater
// protocolConsensusVersion, err := lv.lavavisorStateTracker.GetProtocolVersion(ctx)
// if err != nil {
// 	utils.LavaFormatFatal("failed fetching protocol version from node", err)
// }

// versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+protocolConsensusVersion.ProviderMin)
// if _, err := os.Stat(versionDir); os.IsNotExist(err) {
// 	utils.LavaFormatFatal("expected version directory does not exist, might be deleted!", err)
// }
// binaryPath := filepath.Join(versionDir, "lava-protocol")

// lv.lavavisorStateTracker.RegisterForVersionUpdates(ctx, protocolConsensusVersion, lavavisorPath, binaryPath, autoDownload, providers)

// }

var cmdLavavisorStart = &cobra.Command{
	Use:   "start",
	Short: "A command that will start provider processes given with config.yml",
	Long: `A command that will start provider processes given with config.yml and starts 
    lavavisor listening process. It reads config.yaml, checks the list of provider-services, 
    and starts them with the linked 'which lava-protocol' binary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("directory")

		// Build path to ./lavavisor
		lavavisorPath, err := lvutil.GetLavavisorPath(dir)
		if err != nil {
			return err
		}

		// Read config.yaml
		configPath := filepath.Join(lavavisorPath, "/config.yml")
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
			processmanager.StartProvider(&providers, provider)
		}

		// // Providers will run on their own go routine
		// // Now we'll create a go routine for LavaVisor & start monitoring for version changes constantly
		// // tracker initialization
		// ctx := context.Background()
		// clientCtx, err := client.GetClientQueryContext(cmd)
		// if err != nil {
		// 	return err
		// }
		// txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())

		// // auto-download
		// autoDownload, err := cmd.Flags().GetBool("auto-download")
		// if err != nil {
		// 	return err
		// }

		// // Start lavavisor version monitor process
		// lavavisor := LavaVisor{}
		// err = lavavisor.Start(ctx, txFactory, clientCtx, lavavisorPath, autoDownload, &lavavisor)
		return err
	},
}

func init() {
	cmdLavavisorStart.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorStart.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	rootCmd.AddCommand(cmdLavavisorStart)
}
