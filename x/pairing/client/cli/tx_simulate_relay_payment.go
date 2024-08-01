package cli

import (
	"context"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	QoSValuesFlag = "qos-values"
	CuAmountFlag  = "cu-amount"
	EpochFlag     = "epoch"
	RelayNumFlag  = "relay-num"
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
			consumerAddress, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			keyName, err := sigs.GetKeyName(clientCtx.WithFrom(consumerAddress))
			if err != nil {
				return err
			}

			privKey, err := sigs.GetPrivKey(clientCtx, keyName)
			if err != nil {
				return err
			}

			// SpecID
			specId := args[1]

			// CU
			cuAmount, err := cmd.Flags().GetUint64("cu-amount")
			if err != nil {
				return err
			}

			// Provider
			providerAddr := clientCtx.GetFromAddress().String()

			// Relay Num
			relayNum, err := cmd.Flags().GetUint64(RelayNumFlag)
			if err != nil {
				return err
			}

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
			epochValue, err := cmd.Flags().GetUint64(EpochFlag)
			if err != nil {
				return err
			}
			epoch, err := extractEpoch(clientCtx, epochValue)
			if err != nil {
				return err
			}

			relaySessions := []*types.RelaySession{}
			for relayIdx := uint64(1); relayIdx <= relayNum; relayIdx++ {
				// Session ID
				// We need to randomize it, otherwise it will give unique ID error
				sessionId := uint64(time.Now().UnixNano())

				// Create RelaySession
				relaySession := newRelaySession(specId, sessionId, cuAmount, providerAddr, relayIdx, qosReport, epoch, clientCtx.ChainID)

				sig, err := sigs.Sign(privKey, *relaySession)
				if err != nil {
					return err
				}
				// Set Sig
				relaySession.Sig = sig

				relaySessions = append(relaySessions, relaySession)
			}

			msg := types.NewMsgRelayPayment(
				clientCtx.GetFromAddress().String(),
				relaySessions,
				"Simulation for RelayPayment",
				nil,
			)
			utils.LavaFormatInfo("Message Data ", utils.Attribute{Key: "msg", Value: msg})

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)
	cmd.Flags().Uint64(CuAmountFlag, 1, "CU serviced by the provider")
	cmd.Flags().Uint64(RelayNumFlag, 1, "Number of relays")
	cmd.Flags().StringSlice(QoSValuesFlag, []string{"1", "1", "1"}, "QoS values: latency, availability, sync scores")
	cmd.Flags().Uint64(EpochFlag, 0, "Epoch value to be used. If not set, it will be queried from the chain.")

	return cmd
}

func newRelaySession(
	specId string,
	sessionId uint64,
	cuAmount uint64,
	providerAddr string,
	relayNum uint64,
	qosReport *types.QualityOfServiceReport,
	epoch int64,
	chainID string,
) *types.RelaySession {
	relaySession := &types.RelaySession{
		SpecId:      specId,
		SessionId:   sessionId,
		CuSum:       cuAmount,
		Provider:    providerAddr,
		RelayNum:    relayNum,
		QosReport:   qosReport,
		Epoch:       epoch,
		LavaChainId: chainID,
	}
	return relaySession
}

func extractQoSFlag(qosValues []string) (qosReport *types.QualityOfServiceReport, err error) {
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

func extractEpoch(clientCtx client.Context, epochValue uint64) (epoch int64, err error) {
	// If epochValue is not the default value (0 in this case), use it as epoch
	if epochValue != 0 {
		return int64(epochValue), nil
	}
	epochStorageQuerier := epochstoragetypes.NewQueryClient(clientCtx)
	params := &epochstoragetypes.QueryGetEpochDetailsRequest{}
	epochDetails, err := epochStorageQuerier.EpochDetails(context.Background(), params)
	if err != nil {
		return int64(0), err
	}
	return int64(epochDetails.EpochDetails.StartBlock), nil
}
