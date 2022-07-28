package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/conflict/types"
	"github.com/spf13/cobra"
)

func CmdListConflictVote() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-conflict-vote",
		Short: "list all ConflictVote",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllConflictVoteRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.ConflictVoteAll(context.Background(), params)
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

func CmdShowConflictVote() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-conflict-vote [index]",
		Short: "shows a ConflictVote",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetConflictVoteRequest{
				Index: argIndex,
			}

			res, err := queryClient.ConflictVote(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
