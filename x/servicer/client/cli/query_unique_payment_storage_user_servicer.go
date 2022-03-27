package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cobra"
)

func CmdListUniquePaymentStorageUserServicer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-unique-payment-storage-user-servicer",
		Short: "list all UniquePaymentStorageUserServicer",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllUniquePaymentStorageUserServicerRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.UniquePaymentStorageUserServicerAll(context.Background(), params)
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

func CmdShowUniquePaymentStorageUserServicer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-unique-payment-storage-user-servicer [index]",
		Short: "shows a UniquePaymentStorageUserServicer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetUniquePaymentStorageUserServicerRequest{
				Index: argIndex,
			}

			res, err := queryClient.UniquePaymentStorageUserServicer(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
