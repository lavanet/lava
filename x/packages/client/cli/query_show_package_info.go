package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/packages/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdShowPackageInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-package-info [package-index]",
		Short: "Query to show package info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqPackageIndex := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryShowPackageInfoRequest{
				PackageIndex: reqPackageIndex,
			}

			res, err := queryClient.ShowPackageInfo(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
