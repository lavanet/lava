package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

func CmdQueryProviderDelegators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-delegators [provider]",
		Short: "shows all the delegators of a specific provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			provider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// check if the command includes --with-pending
			withPendingDelegationsFlag := cmd.Flags().Lookup(WithPendingDelegatorsFlagName)
			if withPendingDelegationsFlag == nil {
				return fmt.Errorf("%s flag wasn't found", WithPendingDelegatorsFlagName)
			}
			withPendingDelegations := withPendingDelegationsFlag.Changed

			res, err := queryClient.ProviderDelegators(cmd.Context(), &types.QueryProviderDelegatorsRequest{
				Provider:    provider,
				WithPending: withPendingDelegations,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().Bool(WithPendingDelegatorsFlagName, false, "output with pending delegations (applied from next epoch)")

	return cmd
}
