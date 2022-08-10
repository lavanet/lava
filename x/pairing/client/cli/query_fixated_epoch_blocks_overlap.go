package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdListFixatedEpochBlocksOverlap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-fixated-epoch-blocks-overlap",
		Short: "list all FixatedEpochBlocksOverlap",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllFixatedEpochBlocksOverlapRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.FixatedEpochBlocksOverlapAll(context.Background(), params)
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

func CmdShowFixatedEpochBlocksOverlap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-fixated-epoch-blocks-overlap [index]",
		Short: "shows a FixatedEpochBlocksOverlap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetFixatedEpochBlocksOverlapRequest{
				Index: argIndex,
			}

			res, err := queryClient.FixatedEpochBlocksOverlap(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
