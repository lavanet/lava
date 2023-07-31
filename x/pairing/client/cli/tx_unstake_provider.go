package cli

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnstakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake-provider [chain-id,chain-id,chain-id...]",
		Short: "unstake a provider staked on a specific specification on the lava blockchain initiating an un-stake period, funds are returned at the end of the period",
		Long: `args:
		[chain-id,chain-id] is the specs the provider wishes to stop supporting separated by a ','`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argChainIDs := args[0]
			chainIDs := strings.Split(argChainIDs, ",")
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msgs := []sdk.Msg{}
			for _, chainID := range chainIDs {
				if chainID == "" {
					continue
				}
				msg := types.NewMsgUnstakeProvider(
					clientCtx.GetFromAddress().String(),
					chainID,
				)
				if err := msg.ValidateBasic(); err != nil {
					return err
				}
				msgs = append(msgs, msg)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
