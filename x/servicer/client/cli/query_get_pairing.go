package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdGetPairing() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-pairing [spec-name] [user-addr]",
		Short: "Query getPairing",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqSpecName := args[0]
			reqUserAddr := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetPairingRequest{

				SpecName: reqSpecName,
				UserAddr: reqUserAddr,
			}

			res, err := queryClient.GetPairing(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
