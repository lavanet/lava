package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/x/packages/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group packages queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListPackageEntry())
	cmd.AddCommand(CmdShowPackageEntry())
	cmd.AddCommand(CmdShowAllPackages())

	cmd.AddCommand(CmdShowPackageInfo())

	cmd.AddCommand(CmdListPackageUniqueIndex())
	cmd.AddCommand(CmdShowPackageUniqueIndex())
	// this line is used by starport scaffolding # 1

	return cmd
}
