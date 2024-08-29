package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/spec/types"
	"github.com/spf13/cobra"
)

const FlagRaw = "raw"

func CmdListSpec() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-spec",
		Short: "list all Spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllSpecRequest{
				Pagination: pageReq,
			}

			raw, _ := cmd.Flags().GetBool(FlagRaw)
			var res *types.QueryAllSpecResponse

			if raw {
				res, err = queryClient.SpecAllRaw(context.Background(), params)
			} else {
				res, err = queryClient.SpecAll(context.Background(), params)
			}

			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(FlagRaw, false, "Show the Spec in raw format (before imports)")
	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowSpec() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-spec [index]",
		Short: "shows a Spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetSpecRequest{
				ChainID: argIndex,
			}

			raw, _ := cmd.Flags().GetBool(FlagRaw)
			var res *types.QueryGetSpecResponse

			if raw {
				res, err = queryClient.SpecRaw(context.Background(), params)
			} else {
				res, err = queryClient.Spec(context.Background(), params)
			}

			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(FlagRaw, false, "Show the Spec in raw format (before imports)")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
