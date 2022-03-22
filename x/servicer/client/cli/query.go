package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/x/servicer/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group servicer queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListStakeMap())
	cmd.AddCommand(CmdShowStakeMap())
	cmd.AddCommand(CmdListSpecStakeStorage())
	cmd.AddCommand(CmdShowSpecStakeStorage())
	cmd.AddCommand(CmdStakedServicers())

	cmd.AddCommand(CmdShowBlockDeadlineForCallback())
	cmd.AddCommand(CmdListUnstakingServicersAllSpecs())
	cmd.AddCommand(CmdShowUnstakingServicersAllSpecs())
	cmd.AddCommand(CmdGetPairing())

	// this line is used by starport scaffolding # 1

	return cmd
}
