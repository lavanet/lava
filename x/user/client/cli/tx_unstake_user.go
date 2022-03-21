package cli

import (
	"strconv"

	"encoding/json"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/user/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnstakeUser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake-user [spec] [deadline]",
		Short: "Broadcast message unstakeUser",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSpec := new(types.SpecName)
			err = json.Unmarshal([]byte(args[0]), argSpec)
			if err != nil {
				return err
			}
			argDeadline := new(types.BlockNum)
			err = json.Unmarshal([]byte(args[1]), argDeadline)
			if err != nil {
				return err
			}

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
