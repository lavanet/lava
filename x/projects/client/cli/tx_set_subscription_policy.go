package cli

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdSetSubscriptionPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-subscription-policy [project-index] [policy-file-path]",
		Short: "set subscription policy to a project",
		Long:  `The set-subscription-policy command allows the project's subscription consumer to set a new policy to its subscription which will affect some/all of the subscription's projects. The policy file is a YAML file (see cookbook/projects/example_policy.yml for reference). The new policy will be applied from the next epoch.`,
		Example: `required flags: --from <creator-address>
		lavad tx project set-subscription-policy [project-indices] [policy-file-path] --from <creator_address>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argProjects := strings.Split(args[0], listSeparator)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			subscriptionPolicyFilePath := args[1]

			policy, err := planstypes.ParsePolicyFromYaml(subscriptionPolicyFilePath)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetSubscriptionPolicy(
				clientCtx.GetFromAddress().String(),
				argProjects,
				*policy,
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
