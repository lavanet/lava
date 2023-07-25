package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/spf13/cobra"
)

func CmdDelKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-keys [project-id] [optional: project-keys-file-path]",
		Short: "Delete developer/admin keys to an existing project",
		Long: `The del-keys command allows the project admin to delete project keys (admin/developer) from the project.
		To delete the keys you can optionally provide a YAML file of the project keys to delete (see example in cookbook/project/example_project_keys.yml).
		Another way to delete keys is with the --admin-key and --developer-key flags.`,
		Example: `required flags: --from <admin-key> (the project's subscription address is also considered admin)

		lavad tx project del-keys [project-id] [project-keys-file-path] --from <admin-key>
		lavad tx project del-keys [project-id] --admin-key <other-admin-key> --admin-key <another-admin-key> --developer-key <developer-key> --from <admin-key>`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			projectID := args[0]
			var projectKeys []types.ProjectKey

			if len(args) > 1 {
				projectKeysFilePath := args[1]
				_, err = commontypes.ReadYaml(projectKeysFilePath, "Project-Keys", &projectKeys, nil, false)
				if err != nil {
					return err
				}
			} else {
				developerFlagsValue, err := cmd.Flags().GetStringSlice("developer-key")
				if err != nil {
					return err
				}
				for _, developerFlagValue := range developerFlagsValue {
					projectKeys = append(projectKeys, types.ProjectDeveloperKey(developerFlagValue))
				}

				adminAddresses, err := cmd.Flags().GetStringSlice("admin-key")
				if err != nil {
					return err
				}
				for _, adminAddress := range adminAddresses {
					projectKeys = append(projectKeys, types.ProjectAdminKey(adminAddress))
				}
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDelKeys(
				clientCtx.GetFromAddress().String(),
				projectID,
				projectKeys,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("developer-key", []string{}, "Developer keys to add")
	cmd.Flags().StringSlice("admin-key", []string{}, "Admin keys to add")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
