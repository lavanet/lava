package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/x/downtime/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	"github.com/spf13/cobra"
)

func NewQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: types.ModuleName + "query commands",
	}

	cmd.AddCommand(CmdQueryDowntime(), CmdQueryParams())
	return cmd
}

func CmdQueryParams() *cobra.Command {
	return &cobra.Command{
		Use:   "params",
		Short: "Query downtime module params",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			queryClient := v1.NewQueryClient(clientCtx)
			resp, err := queryClient.QueryParams(cmd.Context(), &v1.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}
}

func CmdQueryDowntime() *cobra.Command {
	return &cobra.Command{
		Use:   "downtime [epoch_start_block]",
		Short: "Query downtime",
		Long:  "Query downtime between blocks, if only start is provided then will query for downtime at the given block, if end is provided then it will query the full range",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			queryClient := v1.NewQueryClient(clientCtx)
			resp, err := queryClient.QueryDowntime(cmd.Context(), &v1.QueryDowntimeRequest{
				EpochStartBlock: start,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}
}
