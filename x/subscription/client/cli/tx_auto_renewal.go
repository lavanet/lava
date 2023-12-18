package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	EnableAutoRenewalFlag  = "enable"
	DisableAutoRenewalFlag = "disable"
)

func CmdAutoRenewal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-renewal --[enable/disable] [plan-index] [optional: consumer]",
		Short: "Enable/Disable auto-renewal to a subscription",
		Long: `The auto-renewal command allows the subscription owner (consumer) to enable/disable
auto-renewal to a subscription. When a subscription with enabled auto-renewal expires, it's 
automatically extended by one month indefinitely, until the auto-renewal is disabled.`,
		Example: `Required flags: --from <subscription_consumer>
lavad tx subscription auto-renewal --enable explorer --from <subscription_consumer>
lavad tx subscription auto-renewal --enable explorer <subscription_consumer> --from <subscription_creator>
lavad tx subscription auto-renewal --disable --from <subscription_consumer>
lavad tx subscription auto-renewal --disable <subscription_consumer> --from <subscription_creator>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()
			enabled := false

			enableAutoRenewalFlag := cmd.Flags().Lookup(EnableAutoRenewalFlag)
			if enableAutoRenewalFlag == nil {
				return fmt.Errorf("%s flag wasn't found", EnableAutoRenewalFlag)
			}
			enableAutoRenewal := enableAutoRenewalFlag.Changed

			disableAutoRenewalFlag := cmd.Flags().Lookup(DisableAutoRenewalFlag)
			if disableAutoRenewalFlag == nil {
				return fmt.Errorf("%s flag wasn't found", DisableAutoRenewalFlag)
			}
			disableAutoRenewal := disableAutoRenewalFlag.Changed

			if enableAutoRenewal && !disableAutoRenewal {
				enabled = true
			} else if !enableAutoRenewal && disableAutoRenewal {
				enabled = false
			} else if enableAutoRenewal && disableAutoRenewal {
				return fmt.Errorf("can't use %s and %s together", EnableAutoRenewalFlag, DisableAutoRenewalFlag)
			} else {
				return fmt.Errorf("this command requires the %s or %s flag", EnableAutoRenewalFlag, DisableAutoRenewalFlag)
			}

			planIndex := types.AUTO_RENEWAL_PLAN_NONE
			consumer := creator
			switch len(args) {
			case 1:
				if enabled {
					planIndex = args[0]
				} else {
					consumer = args[0]
				}
			case 2:
				planIndex = args[0]
				consumer = args[1]
			}

			msg := types.NewMsgAutoRenewal(
				creator,
				consumer,
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
	cmd.Flags().Bool(EnableAutoRenewalFlag, false, "enable the auto renewal feature. If plan-index is not provided, the current plan is assumed")
	cmd.Flags().Bool(DisableAutoRenewalFlag, false, "disable the auto renewal feature")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
