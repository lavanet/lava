package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdQueryProviderReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-reward [provider] {chain-id}",
		Short: "Query the total rewards for a given provider and chain",
		Long: `args: 
	[provider] Provider address for which we want to query the rewards
	{chain-id} Chain for which we want to view the rewards of the provider
		`,
		Example: `
	lavad query rewards provider-reward <provider's address>
	lavad query rewards provider-reward <provider's address> ETH1`,

		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqChainID := ""
			if len(args) == 2 {
				reqChainID = args[1]
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			reqProvider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryProviderRewardRequest{
				ChainId:  reqChainID,
				Provider: reqProvider,
			}

			res, err := queryClient.ProviderReward(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
