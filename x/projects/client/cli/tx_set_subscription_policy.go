package cli

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdSetSubscriptionPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-subscription-policy [projects] [policy]",
		Short: "Broadcast message set-subscription-policy",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argProjects := strings.Split(args[1], listSeparator)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// TODO: get policy from YAML
			msg := types.NewMsgSetSubscriptionPolicy(
				clientCtx.GetFromAddress().String(),
				argProjects,
				types.Policy{},
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
