package lavavisor

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"

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
	Services []string `yaml:"services"`
}

func (lv *LavaVisor) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, lavavisorPath string, autoDownload bool, services []*processmanager.ServiceProcess) (err error) {
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

	versionMonitor := processmanager.NewVersionMonitor(version.ProviderMin, lavavisorPath, services, autoDownload)

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
	Short: "A command that will start service processes given with config.yml",
	Long: `A command that will start service processes given with config.yml and starts 
    lavavisor listening process. It reads config.yaml, checks the list of services, 
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
	txFactory, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
	if err != nil {
		utils.LavaFormatFatal("failed to create tx factory", err)
	}

	// auto-download
	autoDownload, err := cmd.Flags().GetBool("auto-download")
	if err != nil {
		return err
	}

	// Read config.yml
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

	// Iterate over the list of services and start them
	var processes []*processmanager.ServiceProcess
	for _, process := range config.Services {
		utils.LavaFormatInfo("Starting process", utils.Attribute{Key: "Process", Value: process})
		processes = processmanager.StartProcess(processes, process)
	}

	// Start lavavisor version monitor process
	lavavisor := LavaVisor{}
	err = lavavisor.Start(ctx, txFactory, clientCtx, lavavisorPath, autoDownload, processes)
	return err
}
