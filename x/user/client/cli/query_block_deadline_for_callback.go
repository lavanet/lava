package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/user/types"
	"github.com/spf13/cobra"
)

func CmdShowBlockDeadlineForCallback() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-block-deadline-for-callback",
		Short: "shows BlockDeadlineForCallback",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetBlockDeadlineForCallbackRequest{}

			res, err := queryClient.BlockDeadlineForCallback(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
