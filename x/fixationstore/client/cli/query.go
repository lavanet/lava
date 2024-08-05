package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/v2/x/fixationstore/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group pairing queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.MODULE_NAME,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.MODULE_NAME),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdAllIndices())
	cmd.AddCommand(CmdStoreKeys())
	cmd.AddCommand(CmdVersions())
	cmd.AddCommand(CmdEntry())

	// this line is used by starport scaffolding # 1

	return cmd
}
