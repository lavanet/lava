package lavavisor

import (
	"context"
	"os"
	"os/signal"

	// "os/exec"
	// "os/signal"
	"path/filepath"

	// "strings"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/app"
	processmanager "github.com/lavanet/lava/ecosystem/lavavisor/pkg/process"
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

func (lv *LavaVisor) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, lavavisorPath string, autoDownload bool, providers []*processmanager.ProviderProcess) (err error) {
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

	versionMonitor := processmanager.NewVersionMonitor(version.ProviderMin, lavavisorPath, providers, autoDownload)

	lavavisorStateTracker.RegisterForVersionUpdates(ctx, version, versionMonitor)

	// A goroutine that checks for process manager's trigger flag!
	versionMonitor.MonitorVersionUpdates(ctx)

	// tearing down
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("Lavavisor ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("Lavavisor signalChan")
	}

	return nil
}

var cmdLavavisorStart = &cobra.Command{
	Use:   "start",
	Short: "A command that will start provider processes given with config.yml",
	Long: `A command that will start provider processes given with config.yml and starts 
    lavavisor listening process. It reads config.yaml, checks the list of provider-services, 
    and starts them with the linked binary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return LavavisorStart(cmd)
	},
}

func init() {
	flags.AddQueryFlagsToCmd(cmdLavavisorStart)
	cmdLavavisorStart.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorStart.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorStart.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	rootCmd.AddCommand(cmdLavavisorStart)
}

func LavavisorStart(cmd *cobra.Command) error {
	dir, _ := cmd.Flags().GetString("directory")
	// Build path to ./lavavisor
	lavavisorPath, err := lvutil.GetLavavisorPath(dir)
	if err != nil {
		return err
	}
	// initialize lavavisor state tracker
	ctx := context.Background()
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}
	txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())

	// auto-download
	autoDownload, err := cmd.Flags().GetBool("auto-download")
	if err != nil {
		return err
	}

	// Read config.yaml
	configPath := filepath.Join(lavavisorPath, "/config.yml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return utils.LavaFormatError("failed to read config.yaml: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return utils.LavaFormatError("failed to unmarshal config.yaml: %v", err)
	}

	// Iterate over the list of provider services and start them
	var providers []*processmanager.ProviderProcess
	for _, provider := range config.ProviderServices {
		utils.LavaFormatInfo("Starting provider: %s\n", utils.Attribute{Key: "Provider", Value: provider})
		providers = processmanager.StartProvider(providers, provider)
	}

	// Start lavavisor version monitor process
	lavavisor := LavaVisor{}
	err = lavavisor.Start(ctx, txFactory, clientCtx, lavavisorPath, autoDownload, providers)
	return err
}
