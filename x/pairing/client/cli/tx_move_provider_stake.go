package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdMoveProviderStake() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move-provider-stake src-chain dst-chain amount",
		Short: "moves providers stak from one chain to another",
		Long:  `moves a provider stake amount from source chain to destination chain that the provider is taked in`,
		Example: `required flags: --from alice.
		lavad tx pairing move-provider-stake chain0 chain1 100ulava"`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			srcChain := args[0]
			dstChain := args[1]
			amount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return fmt.Errorf("failed to parse amount: %w", err)
			}

			msg := types.NewMsgMoveProviderStake(
				clientCtx.GetFromAddress().String(),
				srcChain,
				dstChain,
				amount,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
