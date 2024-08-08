package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdConflictVoteReveal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conflict-vote-reveal [vote-id] [nonce] [hash]",
		Short: "Broadcast message ConflictVoteReveal",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argVoteID := args[0]
			if err != nil {
				return err
			}
			argNonce, err := cast.ToInt64E(args[1])
			if err != nil {
				return err
			}

			argHash := []byte(args[2])

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgConflictVoteReveal(
				clientCtx.GetFromAddress().String(),
				argVoteID,
				argNonce,
				argHash,
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
