package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/dualstaking/types"
)

const (
	providerFlagName = "provider"
)

func CmdQueryDelegatorRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delegator-rewards [delegator]",
		Short: `shows all the rewards that can be claimed for a specific delegator. 
		show rewards from a specific provider using the optional --provider flag`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var provider string

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			delegator, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			// check if the command includes --provider
			providerFlag := cmd.Flags().Lookup(providerFlagName)
			if providerFlag == nil {
				return fmt.Errorf("%s flag wasn't found", providerFlagName)
			}
			provider, err = utils.ParseCLIAddress(clientCtx, providerFlag.Value.String())
			if err != nil {
				return err
			}

			res, err := queryClient.DelegatorRewards(cmd.Context(), &types.QueryDelegatorRewardsRequest{
				Delegator: delegator,
				Provider:  provider,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().String(providerFlagName, "", "output rewards from a specific provider")

	return cmd
}
