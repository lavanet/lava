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
	"golang.org/x/term"
)

const KeyRingPasswordFlag = "enter-keyring-password"

func CreateLavaVisorWrapCobraCommand() *cobra.Command {
	cmdLavavisorWrap := &cobra.Command{
		Use:   "wrap",
		Short: "A command that will wrap a single lavap process, this is usually used in dockerized / kubernetes environments",
		Long: `A command that will start service processes given with a yml file (rpcprovider / rpcconsumer) and starts 
		lavavisor version monitor process.
		and starts them with the linked binary.`,
		Example: `required flags: --cmd
consumer example:
	lavavisor wrap --cmd "lavap rpcconsumer ./config/.../rpcconsumer_config.yml --geolocation 1 --from alice --log-level debug" --auto-download
provider example: 
	lavavisor wrap --cmd "lavap rpcprovider ./config/.../rpcprovider_config.yml --geolocation 1 --from alice --log-level debug" --auto-download
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return LavavisorWrap(cmd)
		},
	}
	flags.AddQueryFlagsToCmd(cmdLavavisorWrap)
	cmdLavavisorWrap.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorWrap.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorWrap.Flags().Bool(KeyRingPasswordFlag, false, "If you are using keyring OS you will need to enter the keyring password for it.")
	cmdLavavisorWrap.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdLavavisorWrap.Flags().String("cmd", "", "the command to execute")
	cmdLavavisorWrap.MarkFlagRequired("cmd")
	return cmdLavavisorWrap
}

func getKeyringPassword(cmd *cobra.Command) *processmanager.KeyRingPassword {
	password, err := cmd.Flags().GetBool(KeyRingPasswordFlag)
	if err != nil {
		utils.LavaFormatFatal("failed fetching flag", err)
	}
	var passphrase string
	if password {
		utils.LavaFormatInfo("[Lavavisor] Please enter the keyring password:")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			utils.LavaFormatFatal("failed reading password from user", err)
		}
		passphrase = string(passwordBytes)
		utils.LavaFormatInfo("[Lavavisor] Password received")
	}
	return &processmanager.KeyRingPassword{Password: password, Passphrase: passphrase}
}

func LavavisorWrap(cmd *cobra.Command) error {
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

	binaryFetcher := processmanager.ProtocolBinaryFetcher{}
	// Validate we have go in Path if we dont we add it to the $PATH and if directory is missing we will download go.
	binaryFetcher.VerifyGoInstallation()
	// Build path to ./lavavisor
	lavavisorPath, err := binaryFetcher.ValidateLavavisorDir(dir)
	if err != nil {
		return err
	}
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

	lavavisor := LavaVisor{}
	err = lavavisor.Wrap(ctx, txFactory, clientCtx, lavavisorPath, autoDownload, runCommand, keyRingPassword)
	return err
}

func (lv *LavaVisor) Wrap(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, lavavisorPath string, autoDownload bool, runCommand string, keyringPassword *processmanager.KeyRingPassword) (err error) {
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
	selectedVersion, _ := SelectMostRecentVersionFromDir(lavavisorPath, version.Version)
	if err != nil {
		utils.LavaFormatWarning("[Lavavisor] failed getting most recent version from .lavavisor dir", err)
	} else {
		utils.LavaFormatInfo("[Lavavisor] Version check OK in '.lavavisor' directory.", utils.Attribute{Key: "Selected Version", Value: selectedVersion})
	}

	// Initialize version monitor with selected most recent version
	versionMonitor := processmanager.NewVersionMonitorProcessWrapFlow(selectedVersion, lavavisorPath, autoDownload, runCommand)

	lavavisorStateTracker.RegisterForVersionUpdates(ctx, version.Version, versionMonitor)

	defer func() {
		if r := recover(); r != nil {
			utils.LavaFormatError("[Lavavisor] Panic occurred: ", nil)
			versionMonitor.StopSubprocess()
		}
	}()

	versionMonitor.StartProcess(keyringPassword)

	// tear down
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("[Lavavisor] Lavavisor ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("[Lavavisor] Lavavisor signalChan")
	}

	return nil
}
