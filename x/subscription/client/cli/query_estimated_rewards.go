package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdEstimatedRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimated-rewards [provider] [chainid] [amount]",
		Short: "calculates the rewards estimation for a provider delegation",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryEstimatedRewardsRequest{}
			req.Provider = args[0]
			req.ChainId = args[1]
			req.Amount = args[2]

			res, err := queryClient.EstimatedRewards(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
