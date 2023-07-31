package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/plans/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdGeolocations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "geolocations",
		Short: "Query to show all the available geolocations",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGeolocationsRequest{}

			res, err := queryClient.Geolocations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
