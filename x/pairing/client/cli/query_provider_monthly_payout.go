package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdProviderMonthlyPayout() *cobra.Command {
	cmd := &cobra.Command{
		Use: "provider-monthly-payout [provider]",
		Short: `Query to show the current monthly payout for a specific provider. It shows the total reward and its 
		components (the amount of funds from each subscription + chain ID)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			provider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryProviderMonthlyPayoutRequest{
				Provider: provider,
			}

			res, err := queryClient.ProviderMonthlyPayout(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
