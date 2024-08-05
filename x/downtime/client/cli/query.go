package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v2/x/downtime/types"
	v1 "github.com/lavanet/lava/v2/x/downtime/v1"
	"github.com/spf13/cobra"
)

func NewQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryDowntime(), CmdQueryParams())
	return cmd
}

func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query downtime module params",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := v1.NewQueryClient(clientCtx)
			resp, err := queryClient.QueryParams(cmd.Context(), &v1.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryDowntime() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "downtime [block]",
		Short: "Query downtime",
		Long:  "Query downtime at the given epoch (the epoch of the block)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
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

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
