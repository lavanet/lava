package lavavisor

// TODO: Parallel service restart
import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/app"
	processmanager "github.com/lavanet/lava/ecosystem/lavavisor/pkg/process"
	lvstatetracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg/state"
	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils/rand"

	"github.com/lavanet/lava/protocol/statetracker"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type LavavisorStateTrackerInf interface {
	RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator statetracker.VersionValidationInf)
	GetProtocolVersion(ctx context.Context) (*statetracker.ProtocolVersionResponse, error)
}

type LavaVisor struct {
	lavavisorStateTracker LavavisorStateTrackerInf
}

type Config struct {
	Services []string `yaml:"services"`
}

func (lv *LavaVisor) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, lavavisorPath string, autoDownload bool, services []string) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	rand.InitRandomSeed()
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

	// Select most recent version set by init command (in the range of min-target version)
	selectedVersion, err := SelectMostRecentVersionFromDir(lavavisorPath, version.Version)
	if err != nil {
		utils.LavaFormatFatal("failed getting most recent version from .lavavisor dir", err)
	}
	utils.LavaFormatInfo("Version check OK in '.lavavisor' directory.", utils.Attribute{Key: "Selected Version", Value: selectedVersion})

	// Initialize version monitor with selected most recent version
	versionMonitor := processmanager.NewVersionMonitor(selectedVersion, lavavisorPath, services, autoDownload)

	lavavisorStateTracker.RegisterForVersionUpdates(ctx, version.Version, versionMonitor)

	// tear down
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("Lavavisor ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("Lavavisor signalChan")
	}

	return nil
}

func CreateLavaVisorStartCobraCommand() *cobra.Command {
	cmdLavavisorStart := &cobra.Command{
		Use:   "start",
		Short: "A command that will start service processes given with config.yml",
		Long: `A command that will start service processes given with config.yml and starts 
		lavavisor version monitor process. It reads config.yaml, checks the list of services, 
		and starts them with the linked binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return LavavisorStart(cmd)
		},
	}
	flags.AddQueryFlagsToCmd(cmdLavavisorStart)
	cmdLavavisorStart.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorStart.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorStart.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	return cmdLavavisorStart
}

func LavavisorStart(cmd *cobra.Command) error {
	dir, _ := cmd.Flags().GetString("directory")
	binaryFetcher := processmanager.ProtocolBinaryFetcher{}
	// Build path to ./lavavisor
	lavavisorPath, err := binaryFetcher.ValidateLavavisorDir(dir)
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
	lavavisorServicesDir := lavavisorPath + "/services/"
	if _, err := os.Stat(lavavisorServicesDir); os.IsNotExist(err) {
		return utils.LavaFormatError("directory does not exist", nil, utils.Attribute{Key: "lavavisorServicesDir", Value: lavavisorServicesDir})
	}
	for _, process := range config.Services {
		utils.LavaFormatInfo("Starting process", utils.Attribute{Key: "Process", Value: process})
		err := processmanager.StartProcess(process)
		if err != nil {
			utils.LavaFormatError("Failed starting process", err, utils.Attribute{Key: "Process", Value: process})
		}
	}

	// Start lavavisor version monitor process
	lavavisor := LavaVisor{}
	err = lavavisor.Start(ctx, txFactory, clientCtx, lavavisorPath, autoDownload, config.Services)
	return err
}

func SelectMostRecentVersionFromDir(lavavisorPath string, version *protocoltypes.Version) (selectedVersion string, err error) {
	upgradesDir := filepath.Join(lavavisorPath, "upgrades")
	// List all directories under lavavisor/upgrades
	dirs, err := os.ReadDir(upgradesDir)
	if err != nil {
		return "", err
	}
	// Filter out directories that match the version naming pattern
	var versions []string
	for _, dir := range dirs {
		if dir.IsDir() && strings.HasPrefix(dir.Name(), "v") {
			versions = append(versions, dir.Name())
		}
	}
	// Sort versions in descending order based on semantic versioning
	sort.Slice(versions, func(i, j int) bool {
		v1 := lvutil.ParseToSemanticVersion(strings.TrimPrefix(versions[i], "v"))
		v2 := lvutil.ParseToSemanticVersion(strings.TrimPrefix(versions[j], "v"))
		return lvutil.IsVersionGreaterThan(v1, v2)
	})
	// Define the version range
	minVersion := lvutil.ParseToSemanticVersion(version.ProviderMin)
	targetVersion := lvutil.ParseToSemanticVersion(version.ProviderTarget)
	// Iterate and check for the most recent valid version within the range
	selectedVersion = ""
	for _, ver := range versions {
		parsedVer := lvutil.ParseToSemanticVersion(strings.TrimPrefix(ver, "v"))
		if lvutil.IsVersionLessThan(parsedVer, minVersion) || lvutil.IsVersionGreaterThan(parsedVer, targetVersion) {
			continue
		}
		versionDir := filepath.Join(upgradesDir, ver)
		binaryPath := filepath.Join(versionDir, "lavap")
		binaryVersion, err := processmanager.GetBinaryVersion(binaryPath)
		if err != nil || binaryVersion == "" {
			continue
		}
		binaryVersionSemantic := lvutil.ParseToSemanticVersion(strings.TrimPrefix(ver, "v"))
		// second check to see if returned protocol binary version is actually in the allowed range
		if lvutil.IsVersionLessThan(binaryVersionSemantic, minVersion) || lvutil.IsVersionGreaterThan(binaryVersionSemantic, targetVersion) {
			continue
		}
		if binaryVersion == strings.TrimPrefix(ver, "v") {
			selectedVersion = binaryVersion
			break
		}
	}

	if selectedVersion == "" {
		return "", utils.LavaFormatError("No valid version found in the range", nil)
	}

	return selectedVersion, nil
}
