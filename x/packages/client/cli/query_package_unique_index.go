package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/packages/types"
	"github.com/spf13/cobra"
)

func CmdListPackageUniqueIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-package-unique-index",
		Short: "list all packageUniqueIndex",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllPackageUniqueIndexRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.PackageUniqueIndexAll(context.Background(), params)
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

func CmdShowPackageUniqueIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-package-unique-index [id]",
		Short: "shows a packageUniqueIndex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			params := &types.QueryGetPackageUniqueIndexRequest{
				Id: id,
			}

			res, err := queryClient.PackageUniqueIndex(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
