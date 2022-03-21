package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/user/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnstakeUser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake-user [spec] [deadline]",
		Short: "Broadcast message unstakeUser",
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

			msg := types.NewMsgUnstakeUser(
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
