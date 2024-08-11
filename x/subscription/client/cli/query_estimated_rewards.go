package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdEstimatedRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimated-rewards [provider] [chainid] {optional: amount/delegator}",
		Short: "calculates the rewards estimation for a provider delegation",
		Long: `Query to estimate the rewards a delegator will get for 1 month from the provider, if used without optional args the calculations will be for the provider.
		optional args can be amount for new delegation or address for an existing one.
		The estimation takes into account subscription rewards, bonus rewards and iprpc rewards, as well as the provider delegation limit, spec contribution and tax.
		The query does not take into account the likes of addons, geolocation and the quality of the provider services.
		args: 
		[provider] provider address.
		[chain-id] provider chain id.
		[amount/delegator] optional: delegation amount for the estimation Or delegator address for existing delegation. if not used the rewards will be calculated for the provider.
		`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryEstimatedRewardsRequest{}
			req.Provider = args[0]
			req.ChainId = args[1]

			if len(args) == 3 {
				address, err := utils.ParseCLIAddress(clientCtx, args[2])
				if err != nil {
					req.Amount = args[2]
				} else {
					req.Delegator = address
				}
			}

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
