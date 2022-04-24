package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdListUniquePaymentStorageClientProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-unique-payment-storage-client-provider",
		Short: "list all UniquePaymentStorageClientProvider",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllUniquePaymentStorageClientProviderRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.UniquePaymentStorageClientProviderAll(context.Background(), params)
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

func CmdShowUniquePaymentStorageClientProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-unique-payment-storage-client-provider [index]",
		Short: "shows a UniquePaymentStorageClientProvider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetUniquePaymentStorageClientProviderRequest{
				Index: argIndex,
			}

			res, err := queryClient.UniquePaymentStorageClientProvider(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
