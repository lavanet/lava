package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdEstimatedProviderRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimated-provider-rewards [provider] {optional: amount/delegator}",
		Short: "Calculates estimated rewards for a provider delegation.",
		Long: `Estimates the rewards a delegator will earn from a provider over one month. If no optional arguments are provided, the calculation is for the provider itself.
		The estimation considers subscription rewards, bonus rewards, IPRPC rewards, provider delegation limits, spec contribution, and tax.
		It does not factor in addons, geolocation, or provider service quality.
		
		The optional argument can either be: 
			- amount: The delegation amount for a new delegation
			- delegator: The address of an existing delegator.
		`,
		Example: ` The query can be used in 3 ways:
		1. estimated-provider-rewards <provider_address>: estimates the monthly reward of a provider.
		2. estimated-provider-rewards <provider_address> <delegator_address>: estimates the monthly reward of a delegator from a specific provider.
		3. estimated-provider-rewards <provider_address> <delegation_amount>: estimates the monthly reward of a delegator that has a specific amount of delegation to a specific provider.
		`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryEstimatedProviderRewardsRequest{}
			req.Provider, err = utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			if len(args) == 2 {
				address, err := utils.ParseCLIAddress(clientCtx, args[1])
				if err != nil {
					req.AmountDelegator = args[1]
				} else {
					req.AmountDelegator = address
				}
			}

			res, err := queryClient.EstimatedProviderRewards(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdPoolRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimated-pools-rewards",
		Short: "Calculates estimated rewards for all pools",
		Long:  `estimate the total rewards a pool will give to all providers if the month ends now`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryEstimatedPoolsRewardsRequest{}

			res, err := queryClient.EstimatedPoolsRewards(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
