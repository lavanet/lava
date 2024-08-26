package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDelProject() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-project [project-name]",
		Short: "Delete a project from a subscription",
		Long: `The del-project command allows the subscription owner to delete an existing project from its
		subscription. If successful, the project will be deleted at the end of the current epoch.`,
		Example: `required flags: --from <subscription_consumer>
		lavad tx subscription del-project <project-name> --from <subscription_consumer>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			projectName := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()

			msg := types.NewMsgDelProject(
				creator,
				projectName,
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
