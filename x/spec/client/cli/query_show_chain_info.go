package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/spec/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdShowChainInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-chain-info [chain-name]",
		Short: "Query to show chain info by chain name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqChainName := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryShowChainInfoRequest{
				ChainName: reqChainName,
			}

			res, err := queryClient.ShowChainInfo(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
