package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/conflict/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDetection() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detection [finalization-conflict] [response-conflict]",
		Short: "Broadcast message detection",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argFinalizationConflict, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}
			argResponseConflict, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDetection(
				clientCtx.GetFromAddress().String(),
				&argFinalizationConflict,
				&argResponseConflict,
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
