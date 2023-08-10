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
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/app"
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

	fmt.Println("start- clientCtx: ", clientCtx)
	fmt.Println("start- txFactory: ", txFactory)
	fmt.Println("start- lavaChainFetcher: ", lavaChainFetcher)

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

	versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+version.ProviderMin)
	binaryPath := filepath.Join(versionDir, "lava-protocol")

	versionMonitor := processmanager.NewVersionMonitor(binaryPath)

	lavavisorStateTracker.RegisterForVersionUpdates(ctx, version, versionMonitor)

	// start a goroutine that checks for process manager's trigger flag!
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
    and starts them with the linked 'which lava-protocol' binary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("directory")
		fmt.Println("start- dir: ", dir)

		// Build path to ./lavavisor
		lavavisorPath, err := lvutil.GetLavavisorPath(dir)
		if err != nil {
			return err
		}
		fmt.Println("start- lavavisorPath: ", lavavisorPath)

		// tracker initialization
		ctx := context.Background()
		fmt.Println("start- ctx: ", ctx)
		clientCtx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			fmt.Println("start- clientCtxerr: ", err)
			return err
		}
		txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())

		fmt.Println("startcmd- clientCtx: ", clientCtx)
		fmt.Println("startcmd- txFactory: ", txFactory)

		// auto-download
		autoDownload, err := cmd.Flags().GetBool("auto-download")
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

		// Start lavavisor version monitor process
		lavavisor := LavaVisor{}
		err = lavavisor.Start(ctx, txFactory, clientCtx, lavavisorPath, autoDownload)
		return err
	},
}

func init() {
	flags.AddQueryFlagsToCmd(cmdLavavisorStart)
	cmdLavavisorStart.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorStart.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorStart.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	rootCmd.AddCommand(cmdLavavisorStart)
}
