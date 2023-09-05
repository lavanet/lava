package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/lavanet/lava/x/dualstaking/types"
)

const WithPendingDelegatorsFlagName = "with-pending"

func CmdQueryDelegatorProviders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegator-providers [delegator]",
		Short: "shows all the providers the delegator delegated to",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			delegator := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// check if the command includes --with-pending
			withPendingDelegatorsFlag := cmd.Flags().Lookup(WithPendingDelegatorsFlagName)
			if withPendingDelegatorsFlag == nil {
				return fmt.Errorf("%s flag wasn't found", WithPendingDelegatorsFlagName)
			}
			withPendingDelegators := withPendingDelegatorsFlag.Changed

			res, err := queryClient.DelegatorProviders(cmd.Context(), &types.QueryDelegatorProvidersRequest{
				Delegator:             delegator,
				WithPendingDelegators: withPendingDelegators,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().Bool(WithPendingDelegatorsFlagName, false, "output with pending delegators (delegated from next epoch)")

	return cmd
}
