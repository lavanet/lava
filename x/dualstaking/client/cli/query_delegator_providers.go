package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/lavanet/lava/x/dualstaking/types"
)

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

			res, err := queryClient.DelegatorProviders(cmd.Context(), &types.QueryDelegatorProvidersRequest{Delegator: delegator})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
