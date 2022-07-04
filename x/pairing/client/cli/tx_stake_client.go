package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdStakeClient() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-client [chain-id] [amount] [geolocation]",
		Short: "Broadcast message stakeClient",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argChainID := args[0]
			argAmount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			argGeolocation, err := cast.ToUint64E(args[2])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			_, vrfpk, err := utils.GetOrCreateVRFKey(clientCtx)
			if err != nil {
				return err
			}
			vrfpkStr, err := vrfpk.EncodeBech32()
			if err != nil {
				return err
			}
			msg := types.NewMsgStakeClient(
				clientCtx.GetFromAddress().String(),
				argChainID,
				argAmount,
				argGeolocation,
				vrfpkStr,
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
