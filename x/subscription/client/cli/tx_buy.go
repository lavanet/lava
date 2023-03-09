package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdBuy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy [index] [consumer] [duration]",
		Short: "buy to a service plan",
		Args:  cobra.RangeArgs(1, 3),
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

			msg := types.NewMsgBuy(
				creator,
				argConsumer,
				argIndex,
				argDuration,
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
