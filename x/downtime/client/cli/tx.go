package cli

import (
	"fmt"

	"github.com/lavanet/lava/v2/x/downtime/types"
	"github.com/spf13/cobra"
)

func NewTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("the downtime module does not have any tx commands")
		},
	}
}
