package rewardserver_test

import (
	"fmt"
	"strconv"
	"testing"

	terderminttypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func stubPaymentEvents(num int) (tos []map[string]string) {
	generateOne := func(uniqueIdentifier uint64) (to map[string]string) {
		// stub data
		relay := pairingtypes.RelaySession{
			SpecId:      "stub-spec",
			ContentHash: []byte{},
			SessionId:   123,
			CuSum:       100,
			Provider:    "lava@test0",
			RelayNum:    5,
			QosReport: &pairingtypes.QualityOfServiceReport{
				Latency:      sdk.OneDec(),
				Availability: sdk.OneDec(),
				Sync:         sdk.OneDec(),
			},
			Epoch:                 20,
			UnresponsiveProviders: nil,
			LavaChainId:           "stub",
			Sig:                   []byte{},
			Badge:                 nil,
		}
		clientAddr := sdk.AccAddress{1, 2, 3, 4}
		providerAddr := sdk.AccAddress{100, 1, 2, 3, 4}
		QoS, _ := relay.QosReport.ComputeQoS()
		rewardCoins, _ := sdk.ParseCoinNormalized("50ulava")
		burnAmount, _ := sdk.ParseCoinNormalized("40ulava")
		from := map[string]string{"chainID": fmt.Sprintf(relay.SpecId), "client": clientAddr.String(), "provider": providerAddr.String(), "CU": strconv.FormatUint(relay.CuSum, 10), "BasePay": rewardCoins.String(), "totalCUInEpoch": strconv.FormatUint(700, 10), "uniqueIdentifier": strconv.FormatUint(relay.SessionId, 10), "descriptionString": "banana"}
		from["QoSReport"] = "Latency: " + relay.QosReport.Latency.String() + ", Availability: " + relay.QosReport.Availability.String() + ", Sync: " + relay.QosReport.Sync.String()
		from["QoSLatency"] = relay.QosReport.Latency.String()
		from["QoSAvailability"] = relay.QosReport.Availability.String()
		from["QoSSync"] = relay.QosReport.Sync.String()
		from["QoSScore"] = QoS.String()
		from["clientFee"] = burnAmount.String()
		from["reliabilityPay"] = "false"
		from["Mint"] = from["BasePay"]
		from["relayNumber"] = strconv.FormatUint(relay.RelayNum, 10)
		to = map[string]string{}
		sessionIDStr := strconv.FormatUint(uniqueIdentifier, 10)
		for key, value := range from {
			to[key+"."+sessionIDStr] = value
		}
		return to
	}
	for idx := 0; idx < num; idx++ {
		generateOne(uint64(idx))
		tos = append(tos, generateOne(uint64(idx)))
	}
	return tos
}

func TestPayments(t *testing.T) {
	internalRelays := 5
	attributesList := stubPaymentEvents(internalRelays)
	eventAttrs := []terderminttypes.EventAttribute{}
	for _, attributes := range attributesList {
		for key, val := range attributes {
			eventAttrs = append(eventAttrs, terderminttypes.EventAttribute{Key: key, Value: val})
		}
	}
	event := terderminttypes.Event{Type: utils.EventPrefix + pairingtypes.RelayPaymentEventName, Attributes: eventAttrs}
	paymentRequests, err := rewardserver.BuildPaymentFromRelayPaymentEvent(event, 500)
	require.Nil(t, err)
	require.Equal(t, internalRelays, len(paymentRequests))
}
