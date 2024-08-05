package cli

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdFreeze() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze [chain-ids]",
		Short: "Freezes a provider",
		Long:  `The freeze command allows a provider to freeze its service, effective next epoch. This allows providers to pause their services without the impact of bad QoS rating. While frozen, the provider won't be paired with consumers. To unfreeze, the provider must use the unfreeze transaction. Example use case: a provider wishes to halt its services during maintenance.`,
		Example: `required flags: --from alice. optional flags: --reason
		lavad tx pairing freeze [chain-ids] --from <provider_address>
		lavad tx pairing freeze [chain-ids] --from <provider_address> --reason <freeze_reason>
		lavad tx pairing freeze ETH1,OSMOSIS --from alice --reason "maintenance"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argChainIds := strings.Split(args[0], listSeparator)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// get the freeze reason (default value: "")
			reason, err := cmd.Flags().GetString(types.ReasonFlagName)
			if err != nil {
				utils.LavaFormatFatal("failed to read freeze reason flag", err)
			}

			msg := types.NewMsgFreeze(
				clientCtx.GetFromAddress().String(),
				argChainIds,
				reason,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)
	cmd.Flags().String(types.ReasonFlagName, "", "reason for freeze")

	return cmd
}
