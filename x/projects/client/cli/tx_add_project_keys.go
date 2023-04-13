package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAddProjectKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-project-keys [project-id] [project-keys-file-path]",
		Short: "Add developer/admin keys to an existing project",
		Long: `The add-project-keys command allows the project admin to add new project keys (admin/developer) to the project.
		To add the keys you need to provide a YAML file of the new project keys (see example in cookbook/project/example_project_keys.yml).
		Note that in project keys, to define the key type, you should follow the enum described in the top of example_project_keys.yml.`,
		Example: `required flags: --from <admin-key> (the project's subscription address is also considered admin)
				  
		lavad tx subscription add-project-keys [project-id] [project-keys-file-path] --from <admin-key>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			projectID := args[0]
			projectKeysFilePath := args[1]

			var projectKeys []types.ProjectKey
			err = commontypes.ReadYaml(projectKeysFilePath, "Project-Keys", &projectKeys)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddProjectKeys(
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

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
