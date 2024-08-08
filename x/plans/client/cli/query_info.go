package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/plans/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [plan-index]",
		Short: "Query to show a plan info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqPlanIndex := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryInfoRequest{
				PlanIndex: reqPlanIndex,
			}

			res, err := queryClient.Info(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
