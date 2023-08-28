package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/dualstaking/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate [provider] [chain-id] [amount]",
		Short: "delegate to a provider",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argProvider := args[0]
			argChainID := args[1]
			argAmount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDelegate(
				clientCtx.GetFromAddress().String(),
				argProvider,
				argChainID,
				argAmount,
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
