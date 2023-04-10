package cli

import (
	"path/filepath"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var _ = strconv.Itoa(0)

func CmdSetAdminPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-project-policy [project-index] [policy-file-path]",
		Short: "set policy to a project",
		Long:  `The set-project-policy command allows a project admin to set a new policy to its project. The policy file is a YAML file (see cookbook/project-policies/example.yml for reference). The new policy will be applied from the next epoch.`,
		Example: `required flags: --from <creator-address>
		lavad tx project set-project-policy [project-index] [policy-file-path] --from <creator_address>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			projectId := args[0]
			configPath, configName := filepath.Split(args[1])
			if configPath == "" {
				configPath = "."
			}
			viper.SetConfigName(configName)
			viper.SetConfigType("yml")
			viper.AddConfigPath(configPath)

			err = viper.ReadInConfig()
			if err != nil {
				return err
			}

			var policy types.Policy

			err = viper.GetViper().UnmarshalKey("Policy", &policy, func(dc *mapstructure.DecoderConfig) {
				dc.ErrorUnset = true
				dc.ErrorUnused = true
			})
			if err != nil {
				return err
			}

			msg := types.NewMsgSetAdminPolicy(
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
