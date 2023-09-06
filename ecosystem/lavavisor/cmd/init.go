package lavavisor

import (
	"context"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/app"
	processmanager "github.com/lavanet/lava/ecosystem/lavavisor/pkg/process"
	lvstatetracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg/state"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
)

func CreateLavaVisorInitCobraCommand() *cobra.Command {
	cmdLavavisorInit := &cobra.Command{
		Use:   "init",
		Short: "initializes the environment for LavaVisor",
		Long: `Prepares the local environment for the operation of LavaVisor.
		config.yml should be located in the ./lavavisor/ directory.`,
		Args: cobra.ExactArgs(0),
		Example: `optional flags: --directory | --auto-download | --auto-start 
			lavavisor init <flags>
			lavavisor init --directory ./custom/lavavisor/path 
			lavavisor init --directory ./custom/lavavisor/path --auto-download
			lavavisor init --auto-start --auto-download`,
		RunE: func(cmd *cobra.Command, args []string) error {
			autoStart, err := cmd.Flags().GetBool("auto-start")
			if err != nil {
				return err
			}
			if err := LavavisorInit(cmd); err != nil {
				return err
			}
			if autoStart {
				return LavavisorStart(cmd)
			}
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmdLavavisorInit)
	cmdLavavisorInit.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorInit.Flags().Bool("auto-download", false, "Automatically download missing binaries")
	cmdLavavisorInit.Flags().Bool("auto-start", false, "Executes start cmd automatically after init is completed")
	cmdLavavisorInit.Flags().String(flags.FlagChainID, app.Name, "network chain id")

	return cmdLavavisorInit
}

func LavavisorInit(cmd *cobra.Command) error {
	dir, _ := cmd.Flags().GetString("directory")
	// Build path to ./lavavisor
	lavavisorPath, err := processmanager.SetupLavavisorDir(dir)
	if err != nil {
		return err
	}
	// check auto-download flag
	autoDownload, err := cmd.Flags().GetBool("auto-download")
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

	lavavisorChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	lavavisorStateTracker, err := lvstatetracker.NewLavaVisorStateTracker(ctx, txFactory, clientCtx, lavavisorChainFetcher)
	if err != nil {
		return err
	}
	// fetch lavap version from consensus
	protocolConsensusVersion, err := lavavisorStateTracker.GetProtocolVersion(ctx)
	if err != nil {
		return utils.LavaFormatError("protcol version cannot be fetched from consensus", err)
	}
	utils.LavaFormatInfo("Initializing the environment", utils.Attribute{Key: "Version", Value: protocolConsensusVersion.ProviderMin})

	// fetcher returns binaryPath (according to selected min or target version)
	binaryPath, err := processmanager.FetchProtocolBinary(lavavisorPath, autoDownload, protocolConsensusVersion)
	if err != nil {
		return utils.LavaFormatError("Protocol binary couldn't be fetched", nil)
	}
	// linker
	processmanager.CreateLink(binaryPath)

	return nil
}
