package rewardserver_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	"golang.org/x/net/context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
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
	ts := setup(t)
	db := rewardserver.NewMemoryDB()
	rewardStore := rewardserver.NewRewardStore(db)

	testCases := []struct {
		Proofs                   []*pairingtypes.RelaySession
		ExpectedExistingCu       uint64
		ExpectedUpdatedWithProof bool
	}{
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(1), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(2), uint64(0), "spec", nil),
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(2), uint64(0), "newSpec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(3), uint64(0), "spec", nil),
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(4), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(5), uint64(0), "spec", nil),
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(5), uint64(42), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(6), uint64(42), "spec", nil),
				common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(6), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(42),
			ExpectedUpdatedWithProof: false,
		},
	}

	for _, testCase := range testCases {
		var existingCU, updatedWithProf = uint64(0), false
		for _, proof := range testCase.Proofs {
			rws := rewardserver.NewRewardServerWithStorage(&rewardsTxSenderDouble{}, nil, rewardStore)
			existingCU, updatedWithProf = rws.SendNewProof(context.TODO(), proof, uint64(proof.Epoch), "consumerAddress", "apiInterface")
		}
		require.Equal(t, testCase.ExpectedExistingCu, existingCU)
		require.Equal(t, testCase.ExpectedUpdatedWithProof, updatedWithProf)
	}
}

func TestUpdateEpoch(t *testing.T) {
	ts := setup(t)
	stubRewardsTxSender := rewardsTxSenderDouble{}
	db := rewardserver.NewMemoryDB()
	rewardStore := rewardserver.NewRewardStore(db)
	rws := rewardserver.NewRewardServerWithStorage(&stubRewardsTxSender, nil, rewardStore)

	for _, sessionId := range []uint64{1, 2, 3, 4, 5} {
		epoch := sessionId%2 + 1
		proof := common.BuildRelaySession(ts.ctx, "provider", []byte{}, sessionId, uint64(0), "spec", nil)
		proof.Epoch = int64(epoch)

		sig, err := sigs.SignRelay(ts.consumer.SK, *proof)
		proof.Sig = sig
		require.NoError(t, err)

		_, _ = rws.SendNewProof(context.Background(), proof, epoch, ts.consumer.Addr.String(), "apiInterface")
	}

	rws.UpdateEpoch(1)

	// 2 payments for epoch 1
	require.Len(t, stubRewardsTxSender.sentPayments, 2)

	rws.UpdateEpoch(2)

	// another 3 payments for epoch 2
	require.Len(t, stubRewardsTxSender.sentPayments, 5)
}

type testStruct struct {
	ctx      context.Context
	keepers  *testkeeper.Keepers
	servers  *testkeeper.Servers
	consumer common.Account
}

func setup(t *testing.T) *testStruct {
	ts := &testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	spec := common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), spec)

	plan := common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), plan)

	ctx := testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	ts.consumer = common.CreateNewAccount(ctx, *ts.keepers, 10000)

	return ts
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
