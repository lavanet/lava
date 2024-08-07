package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnbond() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbond [validator] [provider] [chain-id] [amount]",
		Short: "unbond from a provider",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			argValidator := args[0]
			argProvider, err := utils.ParseCLIAddress(clientCtx, args[1])
			if err != nil {
				return err
			}
			argChainID := args[2]
			argAmount, err := sdk.ParseCoinNormalized(args[3])
			if err != nil {
				return err
			}

			msg := types.NewMsgUnbond(
				clientCtx.GetFromAddress().String(),
				argValidator,
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
