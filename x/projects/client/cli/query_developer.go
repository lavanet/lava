package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDeveloper() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "developer [developer-addr]",
		Short: "Query to show the project for a developer key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryDeveloperRequest{Developer: args[0]}

			res, err := queryClient.Developer(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
