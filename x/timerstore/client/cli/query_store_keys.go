package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/timerstore/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdStoreKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store-keys",
		Short: "Query all timer store keys and prefixes",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryStoreKeysRequest{}

			res, err := queryClient.StoreKeys(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
