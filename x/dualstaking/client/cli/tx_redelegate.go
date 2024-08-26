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

func CmdRedelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redelegate [from-provider] [from-chain-id] [to-provider] [to-chain-id] [amount]",
		Short: "redelegate from one provider to another provider",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			argFromProvider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}
			argFromChainID := args[1]
			argToProvider, err := utils.ParseCLIAddress(clientCtx, args[2])
			if err != nil {
				return err
			}
			argToChainID := args[3]
			argAmount, err := sdk.ParseCoinNormalized(args[4])
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

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
