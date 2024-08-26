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

const (
	BuyEnableAutoRenewalFlag = "enable-auto-renewal"
	AdvancedPurchaseFlag     = "advance-purchase"
)

func CmdBuy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy [plan-index] [optional: consumer] [optional: duration(months)]",
		Short: "Buy a service plan",
		Long: `The buy command allows a user to buy or upgrade a subscription to a service plan for another user, effective next epoch. 
The consumer is the beneficiary user (default: the creator). 
The duration is stated in number of months (default: 1).
If the plan index is different than the consumer's current plan, it will upgrade to that plan index.
If the plan index is the same as the consumer's current plan, it will extend the current subscription by [duration].`,
		Example: `required flags: --from <creator-address>, optional flags: --enable-auto-renewal
		lavad tx subscription buy [plan-index] --from <creator_address>
		lavad tx subscription buy [plan-index] --from <creator_address> <consumer_address> 12
		lavad tx subscription buy [plan-index] --from <creator_address> <consumer_address> 12 --enable-auto-renewal
		lavad tx subscription buy [plan-index] --from <creator_address> <consumer_address> 12 --advance-purchase`,
		Args: cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()
			argIndex := args[0]

			argConsumer := creator
			if len(args) >= 2 {
				argConsumer, err = utils.ParseCLIAddress(clientCtx, args[1])
				if err != nil {
					return err
				}
			}

			argDuration := uint64(1)
			if len(args) == 3 {
				argDuration = cast.ToUint64(args[2])
			}

			// check if the command includes --enable-auto-renewal
			enableAutoRenewalFlag := cmd.Flags().Lookup(BuyEnableAutoRenewalFlag)
			if enableAutoRenewalFlag == nil {
				return fmt.Errorf("%s flag wasn't found", BuyEnableAutoRenewalFlag)
			}
			autoRenewal := enableAutoRenewalFlag.Changed

			// check if the command includes --enable-auto-renewal
			advancedPurchasedFlag := cmd.Flags().Lookup(AdvancedPurchaseFlag)
			if advancedPurchasedFlag == nil {
				return fmt.Errorf("%s flag wasn't found", AdvancedPurchaseFlag)
			}
			advancedPurchase := advancedPurchasedFlag.Changed

			msg := types.NewMsgBuy(
				creator,
				argConsumer,
				argIndex,
				argDuration,
				autoRenewal,
				advancedPurchase,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().Bool(BuyEnableAutoRenewalFlag, false, "enables auto-renewal upon expiration")
	cmd.Flags().Bool(AdvancedPurchaseFlag, false, "make an advanced purchase that will be activated once the current subscription ends")

	return cmd
}
