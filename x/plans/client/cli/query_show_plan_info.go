package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/plans/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdShowPlanInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-plan-info [plan-index]",
		Short: "Query to show plan info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqPlanIndex := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryShowPlanInfoRequest{
				PlanIndex: reqPlanIndex,
			}

			res, err := queryClient.ShowPlanInfo(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
