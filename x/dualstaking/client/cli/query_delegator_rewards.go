package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

const (
	providerFlagName = "provider"
	chainIDFlagName  = "chain-id"
)

func CmdQueryDelegatorRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delegator-rewards [delegator]",
		Short: `shows all the rewards that can be claimed for a specific delegator. 
		Can be more specific using the optional --provider and --chain-id flags`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var provider, chainID string

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

			// check if the command includes --chain-id
			chainIDFlag := cmd.Flags().Lookup(chainIDFlagName)
			if chainIDFlag == nil {
				return fmt.Errorf("%s flag wasn't found", chainIDFlagName)
			}
			if cmd.Flags().Changed(chainIDFlagName) {
				chainID = chainIDFlag.Value.String()
			}

			res, err := queryClient.DelegatorRewards(cmd.Context(), &types.QueryDelegatorRewardsRequest{
				Delegator: delegator,
				Provider:  provider,
				ChainId:   chainID,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().String(providerFlagName, "", "output rewards from a specific provider")
	cmd.Flags().String(chainIDFlagName, "", "output rewards for a specific chain")

	return cmd
}
