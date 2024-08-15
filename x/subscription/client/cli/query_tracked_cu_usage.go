package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdTrackedCuUsage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tracked-cu-usage [subscription]",
		Short: "shows the tracked cu for a given subscription",
		Long:  "Query to fetch all the CU usage of a specific subscription, the output of this query is a list of providers and the amount of CU they served for the subscription in the current month.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QuerySubscriptionTrackedUsageRequest{Subscription: args[0]}

			res, err := queryClient.TrackedUsage(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
