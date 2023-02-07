package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/packages/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdShowAllPackages() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-all-packages",
		Short: "Query to show all available packages",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryShowAllPackagesRequest{}

			res, err := queryClient.ShowAllPackages(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
