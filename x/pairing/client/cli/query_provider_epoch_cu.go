package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdProviderEpochCu() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-epoch-cu",
		Short: "Query to show the amount of CU serviced by all provider in a specific epoch",
		Example: `
		lavad q pairing provider-epoch-cu`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryProviderEpochCuRequest{}

			res, err := queryClient.ProviderEpochCu(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
