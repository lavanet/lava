package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v3/x/epochstorage/types"
	"github.com/spf13/cobra"
)

func CmdProviderMetadata() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider-metadata",
		Short:   "returns the metadata for the provider address, if empty returns the metadata of all providers",
		Example: "lavad provider-metadata lava@12345",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			provider := ""
			if len(args) == 1 {
				provider = args[0]
			}

			res, err := queryClient.ProviderMetaData(context.Background(), &types.QueryProviderMetaDataRequest{Provider: provider})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
