package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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
			// Extract clientCtx
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

			// QoS
			qosValues, err := cmd.Flags().GetStringSlice(QoSValuesFlag)
			if err != nil {
				return err
			}
			qosReport, err := extractQoSFlag(qosValues)
			if err != nil {
				return err
			}

			// Epoch
			queryRoute := "epochstorage/EpochDetails" // Adjust this to your actual route and method name
			res, _, err := clientCtx.QueryWithData(queryRoute, nil)
			if err != nil {
				return err
			}
			var epochDetailsResponse epochstoragetypes.QueryGetEpochDetailsResponse
			if err := clientCtx.Codec.UnmarshalJSON(res, &epochDetailsResponse); err != nil {
				return err
			}
			epochDetails := epochDetailsResponse.EpochDetails
			epoch := epochDetails.GetStartBlock()

			// Create RelaySession
			relaySession := &types.RelaySession{
				SpecId:    specId,
				SessionId: sessionId,
				CuSum:     cuAmount,
				Provider:  providerAddr,
				RelayNum:  relayNum,
				QosReport: qosReport,
				Epoch:     int64(epoch),
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

func extractQoSFlag(qosValues []string) (qosReport *types.QualityOfServiceReport, err error) {
	if err != nil {
		return nil, err
	}
	// Check if we have exactly 3 values
	if len(qosValues) != 3 {
		return nil, utils.LavaFormatError("expected 3 values for QoSValuesFlag", nil, utils.Attribute{Key: "QoSValues", Value: qosValues})
	}
	// Convert string values to Dec
	latency, err := sdk.NewDecFromStr(qosValues[0])
	if err != nil {
		return nil, utils.LavaFormatError("invalid latency value", err)
	}

	availability, err := sdk.NewDecFromStr(qosValues[1])
	if err != nil {
		return nil, utils.LavaFormatError("invalid availability value", err)
	}

	syncScore, err := sdk.NewDecFromStr(qosValues[2])
	if err != nil {
		return nil, utils.LavaFormatError("invalid sync score value", err)
	}

	qosReport = &types.QualityOfServiceReport{
		Latency:      latency,
		Availability: availability,
		Sync:         syncScore,
	}

	return qosReport, nil
}
