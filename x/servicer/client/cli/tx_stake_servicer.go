package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdStakeServicer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-servicer [spec] [amount] [deadline] [endpoints]",
		Short: "Broadcast message stakeServicer",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSpec := &types.SpecName{Name: args[0]}
			argAmount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			num, err := cast.ToUint64E(args[2])
			if err != nil {
				return err
			}
			argDeadline := &types.BlockNum{Num: num}

			argOperatorAddresses, err := cast.ToStringSliceE(args[3])
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
				argOperatorAddresses,
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
