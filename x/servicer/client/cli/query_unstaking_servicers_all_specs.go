package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cobra"
)

func CmdListUnstakingServicersAllSpecs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-unstaking-servicers-all-specs",
		Short: "list all unstakingServicersAllSpecs",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllUnstakingServicersAllSpecsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.UnstakingServicersAllSpecsAll(context.Background(), params)
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

func CmdShowUnstakingServicersAllSpecs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-unstaking-servicers-all-specs [id]",
		Short: "shows a unstakingServicersAllSpecs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			params := &types.QueryGetUnstakingServicersAllSpecsRequest{
				Id: id,
			}

			res, err := queryClient.UnstakingServicersAllSpecs(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
