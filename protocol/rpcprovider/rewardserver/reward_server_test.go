package rewardserver_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/store"
	"github.com/lavanet/lava/testutil/common"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"
	"golang.org/x/net/context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
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
			UnresponsiveProviders: []byte{},
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
	var eventAttrs []terderminttypes.EventAttribute
	for _, attributes := range attributesList {
		for key, val := range attributes {
			eventAttrs = append(eventAttrs, terderminttypes.EventAttribute{Key: []byte(key), Value: []byte(val)})
		}
	}
	event := terderminttypes.Event{Type: utils.EventPrefix + pairingtypes.RelayPaymentEventName, Attributes: eventAttrs}
	paymentRequests, err := rewardserver.BuildPaymentFromRelayPaymentEvent(event, 500)
	require.Nil(t, err)
	require.Equal(t, internalRelays, len(paymentRequests))
}

func TestSendNewProof(t *testing.T) {
	proofStore := rewardserver.NewProofStore()
	testCases := []struct {
		Proofs                   []*pairingtypes.RelaySession
		ExpectedExistingCu       uint64
		ExpectedUpdatedWithProof bool
	}{
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(1), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(2), uint64(0), "spec", nil),
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(2), uint64(0), "newSpec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(3), uint64(0), "spec", nil),
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(4), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(5), uint64(0), "spec", nil),
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(5), uint64(42), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(6), uint64(42), "spec", nil),
				common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(6), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(42),
			ExpectedUpdatedWithProof: false,
		},
	}

	for _, testCase := range testCases {
		var existingCU, updatedWithProf = uint64(0), false
		for _, proof := range testCase.Proofs {
			rws := rewardserver.NewRewardServerWithStorage(&rewardsTxSenderDouble{}, nil, proofStore)
			existingCU, updatedWithProf = rws.SendNewProof(context.TODO(), proof, uint64(proof.Epoch), "consumerAddress", "apiInterface")
		}
		require.Equal(t, testCase.ExpectedExistingCu, existingCU)
		require.Equal(t, testCase.ExpectedUpdatedWithProof, updatedWithProf)
	}
}

func TestUpdateEpoch(t *testing.T) {
	t.Run("gathers rewards for all epochs", func(t *testing.T) {
		stubRewardsTxSender := rewardsTxSenderDouble{}
		proofStore := rewardserver.NewProofStore()
		rws := rewardserver.NewRewardServerWithStorage(&stubRewardsTxSender, nil, proofStore)

		for _, sessionId := range []uint64{1, 2, 3, 4, 5} {
			epoch := sessionId%2 + 1
			proof := common.BuildRelaySession(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, sessionId, uint64(0), "spec", nil)
			proof.Epoch = int64(epoch)

			_, _ = rws.SendNewProof(context.Background(), proof, epoch, "consumerAddress", "apiInterface")
		}

		rws.UpdateEpoch(1)

		require.Len(t, stubRewardsTxSender.sentPayments, 2)

		rws.UpdateEpoch(2)

		require.Len(t, stubRewardsTxSender.sentPayments, 5)
	})
}

func newSdkContext() sdk.Context {
	key := sdk.NewKVStoreKey("storekey")
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(key, sdk.StoreTypeIAVL, nil)
	_ = ms.LoadLatestVersion()

	return sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, log.NewNopLogger())
}

type rewardsTxSenderDouble struct {
	sentPayments []*pairingtypes.RelaySession
}

func (rts *rewardsTxSenderDouble) TxRelayPayment(_ context.Context, payments []*pairingtypes.RelaySession, _ string) error {
	rts.sentPayments = append(rts.sentPayments, payments...)

	return nil
}

func (rts *rewardsTxSenderDouble) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(_ context.Context) (uint64, error) {
	return 0, nil
}

func (rts *rewardsTxSenderDouble) EarliestBlockInMemory(_ context.Context) (uint64, error) {
	return 1, nil
}
