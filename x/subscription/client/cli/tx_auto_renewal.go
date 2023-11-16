package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAutoRenewal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-renewal [enable: true/false]",
		Short: "Enable/Disable auto-renewal to a subscription",
		Long: `The auto-renewal command allows the subscription owner (consumer) to enable/disable
		auto-renewal to a subscription. When a subscription with enabled auto-renewal expires, it's 
		automatically extended by one month indefinitely, until the auto-renewal is disabled`,
		Example: `required flags: --from <subscription_consumer>

		lavad tx subscription auto-renewal <true/false> --from <subscription_consumer>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			enableStr := args[0]
			enable, err := strconv.ParseBool(enableStr)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()

			msg := types.NewMsgAutoRenewal(
				creator,
				enable,
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
