package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/packages/types"
	"github.com/spf13/cobra"
)

func CmdListPackageEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-package-versions-storage",
		Short: "list all packageEntry",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllPackageEntryRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.PackageEntryAll(context.Background(), params)
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

func CmdShowPackageEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-package-versions-storage [package-index]",
		Short: "shows a packageEntry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argPackageIndex := args[0]

			params := &types.QueryGetPackageEntryRequest{
				PackageIndex: argPackageIndex,
			}

			res, err := queryClient.PackageEntry(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
