package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdEstimatedValidatorsRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimated--validator-rewards [validator] {optional: amount/delegator}",
		Short: "calculates the rewards estimation for a validator delegation",
		Long: `Query to estimate the rewards a delegator will get for 1 month from the validator, if used without optional args the calculations will be for the validator itself.
		optional args can be amount for new delegation or address for an existing one.
		The estimation takes into account subscription rewards, block rewards and iprpc rewards, as well as commisions.
		args: 
		[validator] validator address.
		[amount/delegator] optional: delegation amount for the estimation Or delegator address for existing delegation. if not used the rewards will be calculated for the validator.
		`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryEstimatedValidatorRewardsRequest{}
			req.Validator = args[0]

			if len(args) == 2 {
				address, err := utils.ParseCLIAddress(clientCtx, args[1])
				if err != nil {
					req.AmountDelegator = args[1]
				} else {
					req.AmountDelegator = address
				}
			}

			res, err := queryClient.EstimatedValidatorRewards(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
