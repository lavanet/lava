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

func CmdQueryIprpcProviderRewardEstimation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iprpc-provider-reward-estimation [provider]",
		Short: "Query for current estimation of IPRPC reward for a specific provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			provider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			params := &types.QueryIprpcProviderRewardEstimationRequest{
				Provider: provider,
			}

			res, err := queryClient.IprpcProviderRewardEstimation(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
