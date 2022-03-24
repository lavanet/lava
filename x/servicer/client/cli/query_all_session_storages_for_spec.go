package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAllSessionStoragesForSpec() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-session-storages-for-spec [spec-name]",
		Short: "Query allSessionStoragesForSpec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqSpecName := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllSessionStoragesForSpecRequest{

				SpecName: reqSpecName,
			}

			res, err := queryClient.AllSessionStoragesForSpec(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
