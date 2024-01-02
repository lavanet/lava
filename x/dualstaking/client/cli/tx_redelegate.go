package cli

import (
	"strconv"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/dualstaking/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdRedelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redelegate [from-provider] [from-chain-id] [to-provider] [to-chain-id] [amount]",
		Short: "redelegate from one provider to another provider",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argFromProvider := args[0]
			argFromChainID := args[1]
			argToProvider := args[2]
			argToChainID := args[3]
			argAmount, err := sdk.ParseCoinNormalized(args[4])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRedelegate(
				clientCtx.GetFromAddress().String(),
				argFromProvider,
				argFromChainID,
				argToProvider,
				argToChainID,
				argAmount,
			)

			if msg.Amount.Denom != commontypes.TokenDenom {
				return sdkerrors.Wrapf(types.ErrWrongDenom, "Coin denomanator is not ulava")
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
