package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdGetPairing() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-pairing [chain-id] [client]",
		Short: "Query getPairing",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqChainID := args[0]
			reqClient := args[1]
			reqblockHeight := int64(-1)
			flagUsed := cmd.Flags().Lookup("height").Changed
			if flagUsed {
				reqblockHeight, err = cmd.Flags().GetInt64("height")
				if err != nil {
					return err
				}
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetPairingRequest{

				ChainID:     reqChainID,
				Client:      reqClient,
				BlockHeight: reqblockHeight,
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
