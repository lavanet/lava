package cli

import (
	"github.com/lavanet/lava/x/downtime/types"
	"github.com/spf13/cobra"
)

func NewQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use: types.ModuleName + "query commands",
	}
}
