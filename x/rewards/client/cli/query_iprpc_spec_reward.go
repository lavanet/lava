package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdQueryIprpcSpecReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iprpc-spec-reward [chain-id]",
		Short: "Query for IPRPC rewards for a specific spec. If no spec is given, all IPRPC rewards will be shown",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var spec string
			if len(args) > 0 {
				spec = args[0]
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryIprpcSpecRewardRequest{
				Spec: spec,
			}

			res, err := queryClient.IprpcSpecReward(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
