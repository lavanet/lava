package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdVerifyPairing() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-pairing [chain-id] [client] [provider] [block]",
		Short: "Query verifyPairing",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			reqChainID := args[0]
			reqClient, err := utils.ParseCLIAddress(clientCtx, args[1])
			if err != nil {
				return err
			}

			reqProvider, err := utils.ParseCLIAddress(clientCtx, args[2])
			if err != nil {
				return err
			}
			reqBlock, err := cast.ToUint64E(args[3])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryVerifyPairingRequest{
				ChainID:  reqChainID,
				Client:   reqClient,
				Provider: reqProvider,
				Block:    reqBlock,
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
