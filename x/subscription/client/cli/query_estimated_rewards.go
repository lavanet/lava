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
		Long: `Query to estimate the rewards a delegator will get for 1 month from a provider with a given delegation amount.
		The estimation takes into account subscription rewards, bonus rewards and iprpc rewards, as well as the provider delegation limit, spec contribution and tax.
		The query does not take into account the likes of addons, geolocation and the quality of the provider services.
		args: 
		[provider] provider address.
		[chain-id] provider chain id.
		[amount] delegation amount for the estimation.
		`,
		Args: cobra.ExactArgs(3),
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
