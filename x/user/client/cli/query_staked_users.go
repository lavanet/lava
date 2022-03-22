package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/user/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdStakedUsers() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staked-users [spec-name]",
		Short: "Query stakedUsers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqSpecName := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryStakedUsersRequest{

				SpecName: reqSpecName,
			}

			res, err := queryClient.StakedUsers(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
