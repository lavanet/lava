package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/fixationstore/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdVersions() *cobra.Command {
	cmd := &cobra.Command{
		Use: "versions [store-key] [prefix] [key]",
		Short: `Query all versions of a specific fixation store entry. A new version 
		is defined as an entry that was appended in a different block`,
		Example: "lavad q fixationstore versions [store_key] [prefix] [key]",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			storeKey := args[0]
			prefix := args[1]
			key := args[2]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryVersionsRequest{
				StoreKey: storeKey,
				Prefix:   prefix,
				Key:      key,
			}

			res, err := queryClient.Versions(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
