package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdCurrentSubscription() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-subscription [consumer]",
		Short: "Query the current subscription of a consumer to a service plan",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			reqConsumer := clientCtx.GetFromAddress().String()
			if len(args) == 1 {
				reqConsumer = args[0]
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryCurrentSubscriptionRequest{
				Consumer: reqConsumer,
			}

			res, err := queryClient.CurrentSubscription(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
