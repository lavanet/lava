package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	QoSValuesFlag = "qos-values"
	CuAmountFlag  = "cu-amount"
)

func CmdSimulateRelayPayment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulate-relay-payment [consumer-key] [spec-id]",
		Short: "Create & broadcast message relayPayment for simulation",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Extract arguments
			// consumerKey := args[0]
			specId := args[1]

			// CU
			cuAmount, err := cmd.Flags().GetUint64("cu-amount")
			if err != nil {
				return err
			}

			// Session ID
			sessionId := uint64(0)

			// Provider
			providerAddr, _ := cmd.Flags().GetString(flags.FlagFrom)

			// Relay Num
			relayNum := uint64(0)

			// Create RelaySession
			relaySession := &types.RelaySession{
				SpecId:    specId,
				SessionId: sessionId,
				CuSum:     cuAmount,
				Provider:  providerAddr,
				RelayNum:  relayNum,
			}

			msg := types.NewMsgRelayPayment(
				clientCtx.GetFromAddress().String(),
				[]*types.RelaySession{relaySession},
				"",
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)
	cmd.Flags().StringSlice(QoSValuesFlag, []string{"1", "1", "1"}, "QoS values: latency, availability, sync scores")
	cmd.Flags().Uint64(CuAmountFlag, 0, "CU serviced by the provider")

	return cmd
}
