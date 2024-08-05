package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	ShowFrozenProvidersFlagName = "show-frozen-providers"
)

func CmdProviders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers [chain-id]",
		Short: "Query providers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqChainID := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// check if the command includes --show-frozen-providers
			showFrozenProvidersFlag := cmd.Flags().Lookup(ShowFrozenProvidersFlagName)
			if showFrozenProvidersFlag == nil {
				return fmt.Errorf("%s flag wasn't found", ShowFrozenProvidersFlagName)
			}
			showFrozenProviders := showFrozenProvidersFlag.Changed

			params := &types.QueryProvidersRequest{
				ChainID:    reqChainID,
				ShowFrozen: showFrozenProviders,
			}

			res, err := queryClient.Providers(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().Bool(ShowFrozenProvidersFlagName, false, "shows frozen providers")

	return cmd
}
