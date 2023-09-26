package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	QoSValuesFlag = "qos-values"
	CuAmountFlag  = "cu-amount"
	EpochFlag     = "epoch"
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
			consumerInput := args[0]

			keyName, err := sigs.GetKeyName(clientCtx.WithFrom(consumerInput))
			if err != nil {
				return err
			}
			fmt.Println("SIM keyName: ", keyName)
			privKey, err := sigs.GetPrivKey(clientCtx, keyName)
			if err != nil {
				return err
			}
			fmt.Println("SIM privKey: ", privKey)

			// SpecID
			specId := args[1]
			fmt.Println("SIM specId: ", specId)

			// CU
			cuAmount, err := cmd.Flags().GetUint64("cu-amount")
			if err != nil {
				return err
			}
			fmt.Println("SIM cuAmount: ", cuAmount)

			// Session ID
			sessionId := uint64(0)

			// Provider
			providerAddr, _ := cmd.Flags().GetString(flags.FlagFrom)
			fmt.Println("SIM providerAddr: ", providerAddr)

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
			fmt.Println("SIM qosReport Latency: ", qosReport.Latency.String())

			// Epoch
			epochValue, err := cmd.Flags().GetUint64(EpochFlag)
			if err != nil {
				return err
			}
			fmt.Println("clientCtx.Height: ", clientCtx.Height)
			epoch, err := extractEpoch(clientCtx, epochValue)
			if err != nil {
				return err
			}
			fmt.Println("SIM epoch: ", epoch)

			// Create RelaySession
			relaySession := newRelaySession(specId, sessionId, cuAmount, providerAddr, relayNum, qosReport, 20, clientCtx.ChainID)
			fmt.Println("SIM relaySession before sig: ", relaySession)

			sig, err := sigs.Sign(privKey, *relaySession)
			if err != nil {
				return err
			}
			// Set Sig
			relaySession.Sig = sig
			fmt.Println("SIM relaySession after sig: ", relaySession)

			msg := types.NewMsgRelayPayment(
				clientCtx.GetFromAddress().String(),
				[]*types.RelaySession{relaySession},
				"",
			)
			fmt.Println("SIM msg: ", msg)

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

func extractEpoch(clientCtx client.Context, epochValue uint64) (epoch int64, err error) {
	// If epochValue is not the default value (0 in this case), use it as epoch
	if epochValue != 0 {
		return int64(epochValue), nil
	}
	status, err := clientCtx.Client.Status(context.Background())
	if err != nil {
		return 0, err
	}
	latestBlockHeight := status.SyncInfo.LatestBlockHeight
	fmt.Println("latestBlockHeight: ", latestBlockHeight)

	epoch = calculateEpochFromBlockHeight(latestBlockHeight)
	return epoch, nil
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

func calculateEpochFromBlockHeight(blockHeight int64) int64 {
	epochSize := int64(20)
	remainder := blockHeight % epochSize
	return blockHeight - remainder
}
