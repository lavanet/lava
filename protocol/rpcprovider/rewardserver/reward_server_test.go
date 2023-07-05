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
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
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
	db := rewardserver.NewMemoryDB()
	rewardDB := rewardserver.NewRewardDB(db)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	testCases := []struct {
		Proofs                   []*pairingtypes.RelaySession
		ExpectedExistingCu       uint64
		ExpectedUpdatedWithProof bool
	}{
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(1), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(2), uint64(0), "spec", nil),
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(2), uint64(0), "newSpec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(3), uint64(0), "spec", nil),
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(4), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(5), uint64(0), "spec", nil),
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(5), uint64(42), "spec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(6), uint64(42), "spec", nil),
				common.BuildRelaySession(ctx, "provider", []byte{}, uint64(6), uint64(0), "spec", nil),
			},
			ExpectedExistingCu:       uint64(42),
			ExpectedUpdatedWithProof: false,
		},
	}

	for _, testCase := range testCases {
		var existingCU, updatedWithProf = uint64(0), false
		for _, proof := range testCase.Proofs {
			rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardDB)
			existingCU, updatedWithProf = rws.SendNewProof(context.TODO(), proof, uint64(proof.Epoch), "consumerAddress", "apiInterface")
		}
		require.Equal(t, testCase.ExpectedExistingCu, existingCU)
		require.Equal(t, testCase.ExpectedUpdatedWithProof, updatedWithProf)
	}
}

func TestSendNewProofWillSetBadgeWhenPrefProofDoesNotHaveOneSet(t *testing.T) {
	ts := setup(t)
	db := rewardserver.NewMemoryDB()
	rewardStore := rewardserver.NewRewardStore(db)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)

	prevProof := common.BuildRelaySessionWithBadge(ts.ctx, "provider", []byte{}, uint64(1), uint64(0), "spec", nil, &pairingtypes.Badge{})
	prevProof.Epoch = int64(1)
	newProof := common.BuildRelaySession(ts.ctx, "provider", []byte{}, uint64(1), uint64(42), "spec", nil)
	newProof.Epoch = int64(1)

	rws.SendNewProof(ts.ctx, prevProof, uint64(1), "consumer", "apiinterface")
	_, updated := rws.SendNewProof(ts.ctx, newProof, uint64(1), "consumer", "apiinterface")

	require.NotNil(t, newProof.Badge)
	require.True(t, updated)
}

func TestSendNewProofWillNotSetBadgeWhenPrefProofHasOneSet(t *testing.T) {
	ts := setup(t)
	db := rewardserver.NewMemoryDB()
	rewardStore := rewardserver.NewRewardStore(db)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)

	prevProof := common.BuildRelaySessionWithBadge(ts.ctx, "provider", []byte{}, uint64(1), uint64(0), "spec", nil, &pairingtypes.Badge{LavaChainId: "43"})
	prevProof.Epoch = int64(1)
	newProof := common.BuildRelaySessionWithBadge(ts.ctx, "provider", []byte{}, uint64(1), uint64(42), "spec", nil, &pairingtypes.Badge{LavaChainId: "42"})
	newProof.Epoch = int64(1)

	rws.SendNewProof(ts.ctx, prevProof, uint64(1), "consumer", "apiinterface")
	_, updated := rws.SendNewProof(ts.ctx, newProof, uint64(1), "consumer", "apiinterface")

	require.NotNil(t, newProof.Badge)
	require.Equal(t, "42", newProof.Badge.LavaChainId)
	require.True(t, updated)
}

func TestUpdateEpoch(t *testing.T) {
	privKey, acc := sigs.GenerateFloatingKey()

	stubRewardsTxSender := rewardsTxSenderDouble{}
	db := rewardserver.NewMemoryDB()
	rewardDB := rewardserver.NewRewardDB(db)
	rws := rewardserver.NewRewardServer(&stubRewardsTxSender, nil, rewardDB)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	for _, sessionId := range []uint64{1, 2, 3, 4, 5} {
		epoch := sessionId%2 + 1
		proof := common.BuildRelaySession(ctx, "provider", []byte{}, sessionId, uint64(0), "spec", nil)
		proof.Epoch = int64(epoch)

		sig, err := sigs.SignRelay(privKey, *proof)
		proof.Sig = sig
		require.NoError(t, err)

		_, _ = rws.SendNewProof(context.Background(), proof, epoch, acc.String(), "apiInterface")
	}

	rws.UpdateEpoch(1)

	// 2 payments for epoch 1
	require.Len(t, stubRewardsTxSender.sentPayments, 2)

	rws.UpdateEpoch(2)

	// another 3 payments for epoch 2
	require.Len(t, stubRewardsTxSender.sentPayments, 5)
}

func BenchmarkSendNewProofInMemory(b *testing.B) {
	ts := setup(b)
	db := rewardserver.NewMemoryDB()
	rewardStore := rewardserver.NewRewardStore(db)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)
	proofs := generateProofs(ts.ctx, b.N)

	b.ResetTimer()
	sendProofs(ts.ctx, proofs, rws)
}

func BenchmarkSendNewProofLocal(b *testing.B) {
	ts := setup(b)
	db := rewardserver.NewLocalDB("badger_test")
	defer func(db *rewardserver.BadgerDB) {
		_ = db.Close()
	}(db)
	rewardStore := rewardserver.NewRewardStore(db)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)

	proofs := generateProofs(ts.ctx, b.N)

	b.ResetTimer()
	sendProofs(ts.ctx, proofs, rws)
}

func generateProofs(ctx context.Context, n int) []*pairingtypes.RelaySession {
	var proofs []*pairingtypes.RelaySession
	for i := 0; i < n; i++ {
		proof := common.BuildRelaySession(ctx, "provider", []byte{}, uint64(1), uint64(0), "spec", nil)
		proof.Epoch = 1
		proofs = append(proofs, proof)
	}
	return proofs
}

func sendProofs(ctx context.Context, proofs []*pairingtypes.RelaySession, rws *rewardserver.RewardServer) {
	for index, proof := range proofs {
		prefix := index%2 + 1
		consumerKey := fmt.Sprintf("consumer%d", prefix)
		apiInterface := fmt.Sprintf("apiInterface%d", prefix)
		rws.SendNewProof(ctx, proof, uint64(proof.Epoch), consumerKey, apiInterface)
	}
}

type testStruct struct {
	ctx      context.Context
	keepers  *testkeeper.Keepers
	servers  *testkeeper.Servers
	consumer common.Account
}

func setup(t testing.TB) *testStruct {
	ts := &testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	spec := common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), spec)

	plan := common.CreateMockPlan()
	_ = ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), plan)

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
