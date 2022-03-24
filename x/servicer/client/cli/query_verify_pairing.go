package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdVerifyPairing() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-pairing [spec] [user-addr] [servicer-addr] [block-num]",
		Short: "Query verifyPairing",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqSpec, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}
			reqUserAddr := args[1]
			reqServicerAddr := args[2]
			reqBlockNum, err := cast.ToUint64E(args[3])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryVerifyPairingRequest{

				Spec:         reqSpec,
				UserAddr:     reqUserAddr,
				ServicerAddr: reqServicerAddr,
				BlockNum:     reqBlockNum,
			}

			res, err := queryClient.VerifyPairing(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
