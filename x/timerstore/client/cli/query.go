package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/v2/x/timerstore/types"
	"github.com/spf13/cobra"
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

	cmd.AddCommand(CmdAllTimers())
	cmd.AddCommand(CmdNext())
	cmd.AddCommand(CmdStoreKeys())

	return cmd
}
