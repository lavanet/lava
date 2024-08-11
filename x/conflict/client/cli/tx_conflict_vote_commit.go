package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdConflictVoteCommit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conflict-vote-commit [vote-id] [hash]",
		Short: "Broadcast message ConflictVoteCommit",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argVoteID := args[0]

			argHash := []byte(args[1])

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgConflictVoteCommit(
				clientCtx.GetFromAddress().String(),
				argVoteID,
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
