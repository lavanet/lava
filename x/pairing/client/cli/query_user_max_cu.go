package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUserMaxCu() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user-max-cu [address] [chain-id]",
		Short: "Query userMaxCu",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqAddress := args[0]
			reqChainID := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryUserMaxCuRequest{

				Address: reqAddress,
				ChainID: reqChainID,
			}

			res, err := queryClient.UserMaxCu(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
