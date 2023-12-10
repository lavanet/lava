package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/conflict/types"
	"github.com/spf13/cobra"
)

func CmdConsumerConflicts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consumer-conflicts <consumer>",
		Short: "Gets a consumer's active conflict list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryConsumerConflictsRequest{
				Consumer: args[0],
			}

			res, err := queryClient.ConsumerConflicts(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
