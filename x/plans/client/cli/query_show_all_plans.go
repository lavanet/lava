package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/plans/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdShowAllPlans() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-all-plans",
		Short: "Query to show all available plan",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryShowAllPlansRequest{}

			res, err := queryClient.ShowAllPlans(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
