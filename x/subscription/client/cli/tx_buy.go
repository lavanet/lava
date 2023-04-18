package cli

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdBuy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy [plan-index] [optional: consumer] [optional: duration(months)]",
		Short: "buy a service plan",
		Long:  `The buy command allows a user to buy a subscription to a service plan for another user, effective next epoch. The consumer is the beneficiary user (default: the creator). The duration is stated in number of months (default: 1).`,
		Example: `required flags: --from <creator-address>
		lavad tx subscription buy [plan-index] --from <creator_address>
		lavad tx subscription buy [plan-index] --from <creator_address> <consumer_address> 12`,
		Args: cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()
			argIndex := args[0]

			argConsumer := creator
			if len(args) >= 2 {
				argConsumer = args[1]
			}

			argDuration := uint64(1)
			if len(args) == 3 {
				argDuration = cast.ToUint64(args[2])
			}
			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			salt := uint64(0)
			saltBytes := make([]byte, 32)
			binary.LittleEndian.PutUint64(saltBytes, salt)
			_, vrfpk, err := utils.GenerateVRFKey(clientCtx, txFactory, saltBytes)
			if err != nil {
				return err
			}
			vrfpk_str, err := vrfpk.EncodeBech32()
			if err != nil {
				return err
			}
			msg := types.NewMsgBuy(
				creator,
				argConsumer,
				argIndex,
				argDuration,
				vrfpk_str,
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
