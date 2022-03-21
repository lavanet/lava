package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/user/types"
	"github.com/spf13/cobra"
)

func CmdListUnstakingUsersAllSpecs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-unstaking-users-all-specs",
		Short: "list all unstakingUsersAllSpecs",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllUnstakingUsersAllSpecsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.UnstakingUsersAllSpecsAll(context.Background(), params)
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

func CmdShowUnstakingUsersAllSpecs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-unstaking-users-all-specs [id]",
		Short: "shows a unstakingUsersAllSpecs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			params := &types.QueryGetUnstakingUsersAllSpecsRequest{
				Id: id,
			}

			res, err := queryClient.UnstakingUsersAllSpecs(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
