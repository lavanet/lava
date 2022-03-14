package cli

import (
	"strconv"

	"encoding/json"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnstakeServicer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake-servicer [spec] [deadline]",
		Short: "Broadcast message unstakeServicer",
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
