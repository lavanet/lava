package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/v2/x/pairing/types"
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
	cmd.AddCommand(CmdProvider())
	cmd.AddCommand(CmdGetPairing())
	cmd.AddCommand(CmdProviderPairingChance())
	cmd.AddCommand(CmdVerifyPairing())
	cmd.AddCommand(CmdUserMaxCu())

	cmd.AddCommand(CmdStaticProvidersList())
	cmd.AddCommand(CmdAccountInfo())
	cmd.AddCommand(CmdEffectivePolicy())

	cmd.AddCommand(CmdSdkPairing())
	cmd.AddCommand(CmdProviderMonthlyPayout())
	cmd.AddCommand(CmdSubscriptionMonthlyPayout())

	cmd.AddCommand(CmdProvidersEpochCu())

	cmd.AddCommand(CmdDebugQuery())

	// this line is used by starport scaffolding # 1

	return cmd
}
