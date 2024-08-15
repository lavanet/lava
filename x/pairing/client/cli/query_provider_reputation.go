package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdProviderReputation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-reputation [address] [chain-id] [cluster]",
		Short: "Query for a provider's reputation. Use \"*\" for specify all for chain/cluster.",
		Args:  cobra.ExactArgs(3),
		Example: `
		Reputation of alice for chain ETH1 and the cluster "free":
		lavad q pairing provider-reputation alice ETH1 free
		
		Reputation of alice for all chains and the cluster "free":
		lavad q pairing provider-reputation alice * free
		
		Reputation of alice for ETH1 and for all clusters:
		lavad q pairing provider-reputation alice ETH1 *
		
		Reputation of alice for all chains and for all clusters:
		lavad q pairing provider-reputation alice * *`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			address, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}
			chainID := args[1]
			cluster := args[2]

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryProviderReputationRequest{
				Address: address,
				ChainID: chainID,
				Cluster: cluster,
			}

			res, err := queryClient.ProviderReputation(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
