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

func CmdSpecTrackedInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec-tracked-info [chainid] {provider}",
		Short: "Query the total tracked cu serviced by providers for a specific spec",
		Long: `Query the total CU serviced to a spec throughout the current month for all providers.
		can specify a provider to get only his CU serviced.
		args: 
		[chain-id] Chain for which we want to view the tracked info.
		{provider} optional specific provider tracked info.
		`,
		Example: `
	lavad query rewards spec-tracked-info ETH1
	lavad query rewards spec-tracked-info ETH1 <provider's address> `,

		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqChainID := ""
			if len(args) == 2 {
				reqChainID = args[0]
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			reqProvider, err := utils.ParseCLIAddress(clientCtx, args[1])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QuerySpecTrackedInfoRequest{
				ChainId:  reqChainID,
				Provider: reqProvider,
			}

			res, err := queryClient.SpecTrackedInfo(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
