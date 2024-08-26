package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdAutoRenewal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-renewal [true/false] [optional: plan-index] [optional: consumer]",
		Short: "Enable/Disable auto-renewal to a subscription",
		Long: `The auto-renewal command allows the subscription owner (consumer) to enable/disable
auto-renewal to a subscription. When a subscription with enabled auto-renewal expires, it's 
automatically extended by one month indefinitely, until the auto-renewal is disabled.
If 'true' is provided, and plan-index is not provided, the current plan is assumed.`,
		Example: `Required flags: --from <subscription_consumer>
lavad tx subscription auto-renewal true --from <subscription_consumer>
lavad tx subscription auto-renewal true explorer --from <subscription_consumer>
lavad tx subscription auto-renewal true explorer <subscription_consumer> --from <subscription_creator>
lavad tx subscription auto-renewal false --from <subscription_consumer>
lavad tx subscription auto-renewal false <subscription_consumer> --from <subscription_creator>`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}

			if args[0] != "true" && args[0] != "false" {
				return fmt.Errorf("the first argument must be 'true' or 'false'")
			}

			maxArgCount := 2          // true/false & consumer
			if cast.ToBool(args[0]) { // If enabled, expect the plan-index as well
				maxArgCount = 3
			}

			return cobra.MaximumNArgs(maxArgCount)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()
			enabled := cast.ToBool(args[0])
			consumer := creator
			planIndex := ""

			switch len(args) {
			case 2:
				if enabled {
					planIndex = args[1]
				} else {
					consumer = args[1]
				}
			case 3:
				planIndex = args[1]
				consumer = args[2]
			}

			parsedConsumer, err := utils.ParseCLIAddress(clientCtx, consumer)
			if err != nil {
				return err
			}

			msg := types.NewMsgAutoRenewal(
				creator,
				parsedConsumer,
				planIndex,
				enabled,
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
