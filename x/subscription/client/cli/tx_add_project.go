package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAddProject() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-project [project-name] [enabled] [geolocation] [optional: project-admin] [optional: project-description]",
		Short: "Add a new project to a subcsciption",
		Long:  `The add-project command allows the subscription owner to create a new project and associate it with its subscription. The project-admin can optionally be a different account than the subscription owner (note, the owner is an admin by default)`,
		Example: `required flags: --from <sub-owner-address>
		lavad tx subscription add-project [project-name] [enable] [geolocation] --from <sub-owner-address>
		lavad tx subscription add-project [project-name] [enable] [geolocation] [project-admin] --from <sub-owner-address>
		lavad tx subscription add-project [project-name] [enable] [geolocation] [project-admin] [project-description] --from <sub-owner-address>`,
		Args: cobra.RangeArgs(3, 5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argProjectName := args[0]
			argEnabled, err := cast.ToBoolE(args[1])
			if err != nil {
				return err
			}

			argGeolocation, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()

			argConsumer := creator
			if len(args) > 3 {
				argConsumer = args[3]
			}

			argProjectDescription := ""
			if len(args) > 4 {
				argProjectDescription = args[4]
			}

			_, vrfpk, err := utils.GetOrCreateVRFKey(clientCtx)
			if err != nil {
				return err
			}
			vrfpk_str, err := vrfpk.EncodeBech32()
			if err != nil {
				return err
			}

			msg := types.NewMsgAddProject(
				creator,
				argProjectName,
				argEnabled,
				argConsumer,
				vrfpk_str,
				argProjectDescription,
				argGeolocation,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.MarkFlagRequired(flags.FlagFrom)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
