package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/packagemanager/types"
	"github.com/spf13/cobra"
)

func CmdListPackageVersionsStorage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-package-versions-storage",
		Short: "list all packageVersionsStorage",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllPackageVersionsStorageRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.PackageVersionsStorageAll(context.Background(), params)
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

func CmdShowPackageVersionsStorage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-package-versions-storage [package-index]",
		Short: "shows a packageVersionsStorage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argPackageIndex := args[0]

			params := &types.QueryGetPackageVersionsStorageRequest{
				PackageIndex: argPackageIndex,
			}

			res, err := queryClient.PackageVersionsStorage(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
