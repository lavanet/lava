package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/spf13/cobra"
)

const (
	FlagGeolocation = "geolocation"
	FlagCluster     = "cluster"
)

func CmdProviderPairingChance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-pairing-chance [provider] [chain_id] --geolocation <geolocation> (optional) --cluster <cluster> (optional)",
		Short: "Query to show the chance of a specific provider to be picked in the pairing process to serve consumers on a specific chain. You can optionally specify geolocation and consumer cluster (else, the query will calculate the average among all geolocations/clusters).",
		Example: `
		lavad q pairing provider-pairing-chance alice ETH1
		lavad q pairing provider-pairing-chance alice ETH1 --geolocation "1"
		lavad q pairing provider-pairing-chance alice ETH1 --geolocation "AS" --cluster "free"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			provider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}
			chainID := args[1]

			geo := planstypes.Geolocation_value["GL"]
			if cmd.Flags().Lookup(FlagGeolocation).Changed {
				geoStr, err := cmd.Flags().GetString(FlagGeolocation)
				if err != nil {
					return err
				}
				geo, err = planstypes.ParseGeoEnum(geoStr)
				if err != nil {
					return err
				}
			}

			cluster := ""
			if cmd.Flags().Lookup(FlagCluster).Changed {
				cluster, err = cmd.Flags().GetString(FlagCluster)
				if err != nil {
					return err
				}
			}

			params := &types.QueryProviderPairingChanceRequest{
				Provider:    provider,
				ChainID:     chainID,
				Geolocation: geo,
				Cluster:     cluster,
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.ProviderPairingChance(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().String(flags.FlagFrom, "", "wallet name or address of the developer to query for, to be used instead of [consumer/project]")
	cmd.Flags().String(FlagGeolocation, "GL", "geolocation to check pairing chance (default: global)")
	cmd.Flags().String(FlagCluster, "", "cluster to check pairing chance (default: empty)")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
