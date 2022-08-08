package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdListFixatedServicersToPair() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-fixated-servicers-to-pair",
		Short: "list all FixatedServicersToPair",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllFixatedServicersToPairRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.FixatedServicersToPairAll(context.Background(), params)
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

func CmdShowFixatedServicersToPair() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-fixated-servicers-to-pair [index]",
		Short: "shows a FixatedServicersToPair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetFixatedServicersToPairRequest{
				Index: argIndex,
			}

			res, err := queryClient.FixatedServicersToPair(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
