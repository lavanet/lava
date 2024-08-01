package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [address] [chain-id]",
		Short: "Query for a provider's stake entry on a specific chain. If a chain ID is not specified, all of the provider's stake entries will be printed",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			address, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}
			chainID := ""
			if len(args) > 1 {
				chainID = args[1]
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryProviderRequest{
				Address: address,
				ChainID: chainID,
			}

			res, err := queryClient.Provider(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
