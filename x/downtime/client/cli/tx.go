package cli

import (
	"fmt"

	"github.com/lavanet/lava/x/downtime/types"
	"github.com/spf13/cobra"
)

func NewTxCmd() *cobra.Command {
	return &cobra.Command{
		Use: types.ModuleName + "tx commands",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("the downtime module does not have any tx commands")
		},
	}
}
