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
		Use:   "add-project [project-name] [enable] [optional: consumer]",
		Short: "Add a new project to a subcsciption",
		Long:  `The add-project command allows the subscription owner (the TX's "creator") to create a new project and associate it with its subscription. The consumer is the beneficiary user, i.e. the project admin (default: the creator)`,
		Example: `required flags: --from <creator-address>
		lavad tx subscription add-project [project-name] [enable] --from <creator_address>
		lavad tx subscription add-project [project-name] [enable] [consumer] --from <creator_address>`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argProjectName := args[0]
			argEnable, err := cast.ToBoolE(args[1])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()

			argConsumer := creator
			if len(args) == 3 {
				argConsumer = args[2]
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
				argEnable,
				argConsumer,
				vrfpk_str,
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
