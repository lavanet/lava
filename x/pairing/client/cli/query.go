package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/x/pairing/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group pairing queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdProviders())

	cmd.AddCommand(CmdClients())

	cmd.AddCommand(CmdGetPairing())

	cmd.AddCommand(CmdVerifyPairing())

	cmd.AddCommand(CmdListUniquePaymentStorageClientProvider())
	cmd.AddCommand(CmdShowUniquePaymentStorageClientProvider())
	cmd.AddCommand(CmdListClientPaymentStorage())
	cmd.AddCommand(CmdShowClientPaymentStorage())
	cmd.AddCommand(CmdListEpochPayments())
	cmd.AddCommand(CmdShowEpochPayments())
	cmd.AddCommand(CmdUserMaxCu())

	cmd.AddCommand(CmdListFixatedServicersToPair())
	cmd.AddCommand(CmdShowFixatedServicersToPair())
	cmd.AddCommand(CmdListFixatedStakeToMaxCu())
	cmd.AddCommand(CmdShowFixatedStakeToMaxCu())
	cmd.AddCommand(CmdListFixatedEpochBlocksOverlap())
	cmd.AddCommand(CmdShowFixatedEpochBlocksOverlap())
	// this line is used by starport scaffolding # 1

	return cmd
}
