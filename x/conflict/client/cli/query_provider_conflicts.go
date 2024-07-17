package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"github.com/spf13/cobra"
)

func CmdProviderConflicts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-conflicts <provider>",
		Short: "Queries a provider's conflict list (ones that the provider was reported in and ones that the provider needs to vote)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			provider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}
			params := &types.QueryProviderConflictsRequest{
				Provider: provider,
			}

			res, err := queryClient.ProviderConflicts(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
