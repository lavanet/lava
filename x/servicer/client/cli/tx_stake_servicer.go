package cli

import (
	"strconv"

	"encoding/json"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdStakeServicer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-servicer [spec] [amount] [deadline]",
		Short: "Broadcast message stakeServicer",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSpec := new(types.SpecName)
			err = json.Unmarshal([]byte(args[0]), argSpec)
			if err != nil {
				return err
			}
			argAmount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			argDeadline := new(types.BlockNum)
			err = json.Unmarshal([]byte(args[2]), argDeadline)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeServicer(
				clientCtx.GetFromAddress().String(),
				argSpec,
				argAmount,
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
