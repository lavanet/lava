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

func CmdUserMaxCu() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user-entry [address] [chain-id] [block]",
		Short: "Query userEntry",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			reqAddress, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}
			reqChainID := args[1]
			reqBlock, err := cast.ToUint64E(args[2])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryUserEntryRequest{
				Address: reqAddress,
				ChainID: reqChainID,
				Block:   reqBlock,
			}

			res, err := queryClient.UserEntry(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
