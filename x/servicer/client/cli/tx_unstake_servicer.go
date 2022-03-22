package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnstakeServicer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake-servicer [spec] [deadline]",
		Short: "Broadcast message unstakeServicer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSpec := &types.SpecName{Name: args[0]}

			num, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}
			argDeadline := &types.BlockNum{Num: num}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUnstakeServicer(
				clientCtx.GetFromAddress().String(),
				argSpec,
				argDeadline,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
