package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/lavanet/lava/v2/x/projects/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdSetSubscriptionPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-subscription-policy [project-index] [policy-file-path]",
		Short: "set subscription policy to a project",
		Long:  `The set-subscription-policy command allows the project's subscription consumer to set a new policy to its subscription which will affect some/all of the subscription's projects. The policy file is a YAML file (see cookbook/projects/example_policy.yml for reference). The new policy will be applied from the next epoch. To define a geolocation in the policy file, use the available geolocations: ` + planstypes.PrintGeolocations(),
		Example: `required flags: --from <creator-address>
		lavad tx project set-subscription-policy [project-indices] [policy-file-path] --from <creator_address>
		lavad tx project set-subscription-policy [project-indices] --delete-policy --from <creator_address>`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argProjects := strings.Split(args[0], listSeparator)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// check if the command includes --delete-policy
			deletePolicyFlag := cmd.Flags().Lookup(DeletePolicyFlagName)
			if deletePolicyFlag == nil {
				return fmt.Errorf("%s flag wasn't found", DeletePolicyFlagName)
			}
			deletePolicy := deletePolicyFlag.Changed

			var policy *planstypes.Policy
			if !deletePolicy {
				if len(args) < 2 {
					return fmt.Errorf("not enough arguments")
				}
				subscriptionPolicyFilePath := args[1]
				policy, err = planstypes.ParsePolicyFromYamlPath(subscriptionPolicyFilePath)
				if err != nil {
					return err
				}
			}

			msg := types.NewMsgSetSubscriptionPolicy(
				clientCtx.GetFromAddress().String(),
				argProjects,
				policy,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().Bool(DeletePolicyFlagName, false, "deletes the policy")

	return cmd
}
