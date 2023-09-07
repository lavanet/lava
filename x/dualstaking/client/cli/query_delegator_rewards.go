package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/lavanet/lava/x/dualstaking/types"
)

func CmdQueryDelegatorRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegator-rewards [delegator] [optional: chain_id]",
		Short: "shows all the rewards that can be claimed for a specific delegator",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			delegator := args[0]
			chainID := ""
			if len(args) > 1 {
				chainID = args[1]
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.DelegatorRewards(cmd.Context(), &types.QueryDelegatorRewardsRequest{
				Delegator: delegator,
				ChainId:   chainID,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
