package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	commontypes "github.com/lavanet/lava/common/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdSetPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-policy [project-index] [policy-file-path]",
		Short: "set policy to a project",
		Long:  `The set-policy command allows a project admin to set a new policy to its project. The policy file is a YAML file (see cookbook/projects/example_policy.yml for reference). The new policy will be applied from the next epoch.`,
		Example: `required flags: --from <creator-address>
		lavad tx project set-policy [project-index] [policy-file-path] --from <creator_address>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			projectId := args[0]
			adminPolicyFilePath := args[1]

			var policy planstypes.Policy
			enumHooks := []commontypes.EnumDecodeHookFuncType{
				commontypes.EnumDecodeHook(uint64(0), planstypes.ParsePolicyEnumValue), // for geolocation
				commontypes.EnumDecodeHook(planstypes.SELECTED_PROVIDERS_MODE(0), planstypes.ParsePolicyEnumValue),
				// Add more enum hook functions for other enum types as needed
			}
			err = commontypes.ReadYaml(adminPolicyFilePath, "Policy", &policy, enumHooks)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetPolicy(
				clientCtx.GetFromAddress().String(),
				projectId,
				policy,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}
