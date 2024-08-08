package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/timerstore/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAllTimers() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "all-timers [store-key] [prefix]",
		Short:   "Query all timers of a specific timer store",
		Example: "lavad q timerstore all-timers [store_key] [prefix]",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			storeKey := args[0]
			prefix := args[1]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllTimersRequest{
				StoreKey: storeKey,
				Prefix:   prefix,
			}

			res, err := queryClient.AllTimers(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
