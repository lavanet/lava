package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/decoder"
	"github.com/lavanet/lava/v2/x/projects/types"
	"github.com/spf13/cobra"
)

func CmdAddKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-keys [project-id] [optional: project-keys-file-path]",
		Short: "Add developer/admin keys to an existing project",
		Long: `The add-keys command allows the project admin to add new project keys (admin/developer) to the project.
		To add the keys you can optionally provide a YAML file of the new project keys (see example in cookbook/project/example_project_keys.yml).
		Another way to add keys is with the --admin-key and --developer-key flags.`,
		Example: `required flags: --from <admin-key> (the project's subscription address is also considered admin)
				  
		lavad tx project add-keys [project-id] [project-keys-file-path] --from <admin-key>
		lavad tx project add-keys [project-id] --admin-key <other-admin-key> --admin-key <another-admin-key> --developer-key <developer-key> --from <admin-key>`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			projectID := args[0]
			var projectKeys []types.ProjectKey

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if len(args) > 1 {
				projectKeysFilePath := args[1]
				err = decoder.DecodeFile(projectKeysFilePath, "Project-Keys", &projectKeys, nil, nil, nil)
				if err != nil {
					return err
				}
			} else {
				developerFlagsValue, err := cmd.Flags().GetStringSlice("developer-key")
				if err != nil {
					return err
				}
				var developerKeys []types.ProjectKey
				for _, developerFlagValue := range developerFlagsValue {
					developer, err := utils.ParseCLIAddress(clientCtx, developerFlagValue)
					if err != nil {
						return err
					}
					developerKeys = append(developerKeys, types.ProjectDeveloperKey(developer))
				}

				adminAddresses, err := cmd.Flags().GetStringSlice("admin-key")
				if err != nil {
					return err
				}
				var adminKeys []types.ProjectKey
				for _, adminAddress := range adminAddresses {
					admin, err := utils.ParseCLIAddress(clientCtx, adminAddress)
					if err != nil {
						return err
					}
					adminKeys = append(adminKeys, types.ProjectAdminKey(admin))
				}

				// doing this redundant line to avoid lint issues
				developerKeys = append(developerKeys, adminKeys...)
				projectKeys = developerKeys
			}

			msg := types.NewMsgAddKeys(
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
