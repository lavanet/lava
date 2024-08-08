package lavavisor

import (
	"context"
	"os"
	"os/signal"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v2/app"
	processmanager "github.com/lavanet/lava/v2/ecosystem/lavavisor/pkg/process"
	lvstatetracker "github.com/lavanet/lava/v2/ecosystem/lavavisor/pkg/state"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/spf13/cobra"
)

func CreateLavaVisorPodCobraCommand() *cobra.Command {
	cmdLavavisorPod := &cobra.Command{
		Use:   "pod",
		Short: "A command that will wrap a single lavap process, this is usually used in k8s environments inside pods",
		Long: `A command that will start service processes given with a yml file (rpcprovider / rpcconsumer) and starts 
		lavavisor version monitor process`,
		Example: `required flags: --cmd
consumer example:
	lavavisor pod --cmd "lavap rpcconsumer ./config/.../rpcconsumer_config.yml --geolocation 1 --from alice --log-level debug" --directory <path-to-persistency>
provider example: 
	lavavisor pod --cmd "lavap rpcprovider ./config/.../rpcprovider_config.yml --geolocation 1 --from alice --log-level debug" --directory <path-to-persistency>
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return LavavisorPod(cmd)
		},
	}
	flags.AddQueryFlagsToCmd(cmdLavavisorPod)
	cmdLavavisorPod.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	// cmdLavavisorPod.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorPod.Flags().Bool(KeyRingPasswordFlag, false, "If you are using keyring OS you will need to enter the keyring password for it.")
	cmdLavavisorPod.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdLavavisorPod.Flags().String("cmd", "", "the command to execute")
	cmdLavavisorPod.MarkFlagRequired("cmd")
	return cmdLavavisorPod
}

func LavavisorPod(cmd *cobra.Command) error {
	keyRingPassword := getKeyringPassword(cmd)
	dir, err := cmd.Flags().GetString("directory")
	if err != nil {
		return err
	}
	utils.LavaFormatInfo("[Lavavisor] Start Wrap command")
	runCommand, err := cmd.Flags().GetString("cmd")
	if err != nil {
		return err
	}
	utils.LavaFormatInfo("[Lavavisor] Running", utils.Attribute{Key: "command", Value: runCommand})

	ctx := context.Background()
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}
	txFactory, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
	if err != nil {
		utils.LavaFormatFatal("failed to create tx factory", err)
	}

	lavavisor := LavaVisor{}
	err = lavavisor.PodStart(ctx, txFactory, clientCtx, runCommand, dir, keyRingPassword)
	return err
}

func (lv *LavaVisor) PodStart(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, runCommand string, lavavisorDir string, keyRingPassword *processmanager.KeyRingPassword) (err error) {
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

	binaryFetcher := processmanager.ProtocolBinaryFetcherWithoutBuild{}
	// Build path to ./lavavisor
	lavavisorPath, err := binaryFetcher.ValidateLavavisorDir(lavavisorDir)
	if err != nil {
		return err
	}
	binaryFetcher.FetchProtocolBinary(version.Version)
	// Select most recent version set by init command (in the range of min-target version)
	selectedVersion, _ := SelectMostRecentVersionFromDir(lavavisorPath, version.Version)
	if err != nil {
		utils.LavaFormatWarning("[Lavavisor] failed getting most recent version from .lavavisor dir", err)
	} else {
		utils.LavaFormatInfo("[Lavavisor] Version check OK in '.lavavisor' directory.", utils.Attribute{Key: "Selected Version", Value: selectedVersion})
	}

	// Initialize version monitor with selected most recent version
	versionMonitor := processmanager.NewVersionMonitorProcessPodFlow(selectedVersion, lavavisorPath, runCommand)

	lavavisorStateTracker.RegisterForVersionUpdates(ctx, version.Version, versionMonitor)

	defer func() {
		if r := recover(); r != nil {
			utils.LavaFormatError("[Lavavisor] Panic occurred: ", nil)
			versionMonitor.StopSubprocess()
		}
	}()

	versionMonitor.StartProcess(keyRingPassword)

	// tear down
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("[Lavavisor] Lavavisor ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("[Lavavisor] Lavavisor signalChan")
	}

	return nil
}
