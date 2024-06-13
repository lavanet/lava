package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdQueryPendingIbcIprpcFunds() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pending-ibc-iprpc-funds [filter: index/creator/spec]",
		Short: "Query for pending IBC IPRPC funds",
		Long: `Query for pending IBC IPRPC funds. Use the optional filter argument to get a specific fund (using its index),
		funds of a specific creator or funds for a specific spec. Each fund has its own cost which is derived from the minimum IPRPC
		cost for funding the IPRPC pool, multiplied by the fund's duration. To cover the cost and apply the pending fund, use 
		the "lavad tx rewards cover-ibc-iprpc-cost" TX`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}

			params := &types.QueryPendingIbcIprpcFundsRequest{
				Filter: filter,
			}

			res, err := queryClient.PendingIbcIprpcFunds(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
