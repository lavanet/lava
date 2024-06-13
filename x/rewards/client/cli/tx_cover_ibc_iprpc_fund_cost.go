package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdCoverIbcIprpcFundCost() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cover-ibc-iprpc-fund-cost [index] --from <creator>",
		Short: "Apply a pending IBC IPRPC fund by paying its mandatory minimum cost of funding the IPRPC pool",
		Long: `Apply a pending IBC IPRPC fund by paying its mandatory minimum cost of funding the IPRPC pool. Find your desired 
		fund's index and cost by using the query pending-ibc-iprpc-funds. By sending this message, the full cost of the fund will be 
		paid automatically. Then, the pending fund's coins will be sent to the IPRPC pool.`,
		Example: `lavad tx rewards cover-ibc-iprpc-fund-cost 4 --from alice`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			index, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			msg := types.NewMsgCoverIbcIprpcFundCost(
				clientCtx.GetFromAddress().String(),
				index,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}
