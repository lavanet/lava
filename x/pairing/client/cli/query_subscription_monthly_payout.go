package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdSubscriptionMonthlyPayout() *cobra.Command {
	cmd := &cobra.Command{
		Use: "subscription-monthly-payout [consumer]",
		Short: `Query to show the current monthly payout for a specific consumer. It shows the total reward that
		is going to be paid to provider and its components (the amount of funds for each provider, ordered by chain ID)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			consumer := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QuerySubscriptionMonthlyPayoutRequest{
				Consumer: consumer,
			}

			res, err := queryClient.SubscriptionMonthlyPayout(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
