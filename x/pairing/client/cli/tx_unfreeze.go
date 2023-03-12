package cli

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnfreeze() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfreeze [chain-ids]",
		Short: "Unfreezes a provider",
		Long: `The unfreeze command allows a provider to unfreeze its activity (starting from the next epoch). This 
		allows the provider to provide service to consumers after freezing.`,
		Example: `required flags: --from alice
		lavad tx pairing unfreeze [chain-ids] --from <provider_address>
		lavad tx pairing unfreeze ETH1,COS3 --from alice`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argChainIds := strings.Split(args[0], listSeparator)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUnfreeze(
				clientCtx.GetFromAddress().String(),
				argChainIds,
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
