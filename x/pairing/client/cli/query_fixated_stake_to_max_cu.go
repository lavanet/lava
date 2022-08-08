package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdListFixatedStakeToMaxCu() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-fixated-stake-to-max-cu",
		Short: "list all FixatedStakeToMaxCu",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllFixatedStakeToMaxCuRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.FixatedStakeToMaxCuAll(context.Background(), params)
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

func CmdShowFixatedStakeToMaxCu() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-fixated-stake-to-max-cu [index]",
		Short: "shows a FixatedStakeToMaxCu",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetFixatedStakeToMaxCuRequest{
				Index: argIndex,
			}

			res, err := queryClient.FixatedStakeToMaxCu(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
