package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/timerstore/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdNext() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "next [store-key] [prefix]",
		Short:   "Query next timeout of a specific timer store",
		Example: "lavad q timerstore next [store_key] [prefix]",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			storeKey := args[0]
			prefix := args[1]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryNextRequest{
				StoreKey: storeKey,
				Prefix:   prefix,
			}

			res, err := queryClient.Next(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
