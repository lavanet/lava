package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdCurrent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current [consumer]",
		Short: "Query the current subscription of a consumer to a service plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			reqConsumer := args[0]

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryCurrentRequest{
				Consumer: reqConsumer,
			}

			res, err := queryClient.Current(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
