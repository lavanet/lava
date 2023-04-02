package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/spf13/cobra"
	"strings"
)

var _ = strconv.Itoa(0)

func CmdSetSubscriptionPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-subscription-policy [subscription] [projects] [policy]",
		Short: "Broadcast message set-subscription-policy",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSubscription := args[0]
			argProjects := strings.Split(args[1], listSeparator)
			argPolicy := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetSubscriptionPolicy(
				clientCtx.GetFromAddress().String(),
				argSubscription,
				argProjects,
				argPolicy,
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
