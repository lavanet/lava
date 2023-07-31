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
	providerAddr := "providerAddr"
	specId := "specId"
	db := rewardserver.NewMemoryDB(specId)
	db2 := rewardserver.NewMemoryDB("newSpec")
	rewardDB := rewardserver.NewRewardDB(db)
	rewardDB.AddDB(db2)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	testCases := []struct {
		Proofs                   []*pairingtypes.RelaySession
		ExpectedExistingCu       uint64
		ExpectedUpdatedWithProof bool
	}{
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(1), uint64(0), specId, nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(2), uint64(0), specId, nil),
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(2), uint64(0), "newSpec", nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(3), uint64(0), specId, nil),
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(4), uint64(0), specId, nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(5), uint64(0), specId, nil),
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(5), uint64(42), specId, nil),
			},
			ExpectedExistingCu:       uint64(0),
			ExpectedUpdatedWithProof: true,
		},
		{
			Proofs: []*pairingtypes.RelaySession{
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(6), uint64(42), specId, nil),
				common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(6), uint64(0), specId, nil),
			},
			ExpectedExistingCu:       uint64(42),
			ExpectedUpdatedWithProof: false,
		},
	}

	for _, testCase := range testCases {
		existingCU, updatedWithProf := uint64(0), false
		for _, proof := range testCase.Proofs {
			rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardDB)
			existingCU, updatedWithProf = rws.SendNewProof(context.TODO(), proof, uint64(proof.Epoch), "consumerAddress", "apiInterface")
		}
		require.Equal(t, testCase.ExpectedExistingCu, existingCU)
		require.Equal(t, testCase.ExpectedUpdatedWithProof, updatedWithProf)
	}
}

func TestSendNewProofWillSetBadgeWhenPrefProofDoesNotHaveOneSet(t *testing.T) {
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	db := rewardserver.NewMemoryDB("specId")
	rewardStore := rewardserver.NewRewardDB(db)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)

	prevProof := common.BuildRelayRequestWithBadge(ctx, "providerAddr", []byte{}, uint64(1), uint64(0), "specId", nil, &pairingtypes.Badge{})
	prevProof.Epoch = int64(1)
	newProof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(42), "specId", nil)
	newProof.Epoch = int64(1)

	rws.SendNewProof(ctx, prevProof, uint64(1), "consumer", "apiinterface")
	_, updated := rws.SendNewProof(ctx, newProof, uint64(1), "consumer", "apiinterface")

	require.NotNil(t, newProof.Badge)
	require.True(t, updated)
}

func TestSendNewProofWillNotSetBadgeWhenPrefProofHasOneSet(t *testing.T) {
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	db := rewardserver.NewMemoryDB("specId")
	rewardStore := rewardserver.NewRewardDB(db)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)

	providerAddr := "providerAddr"
	specId := "specId"
	prevProof := common.BuildRelayRequestWithBadge(ctx, providerAddr, []byte{}, uint64(1), uint64(0), specId, nil, &pairingtypes.Badge{LavaChainId: "43"})
	prevProof.Epoch = int64(1)
	newProof := common.BuildRelayRequestWithBadge(ctx, providerAddr, []byte{}, uint64(1), uint64(42), specId, nil, &pairingtypes.Badge{LavaChainId: "42"})
	newProof.Epoch = int64(1)

	rws.SendNewProof(ctx, prevProof, uint64(1), "consumer", "apiinterface")
	_, updated := rws.SendNewProof(ctx, newProof, uint64(1), "consumer", "apiinterface")

	require.NotNil(t, newProof.Badge)
	require.Equal(t, "42", newProof.Badge.LavaChainId)
	require.True(t, updated)
}

func TestUpdateEpoch(t *testing.T) {
	setupRewardsServer := func() (*rewardserver.RewardServer, *rewardsTxSenderDouble, *rewardserver.RewardDB) {
		stubRewardsTxSender := rewardsTxSenderDouble{}
		db := rewardserver.NewMemoryDB("spec")
		rewardDB := rewardserver.NewRewardDB(db)
		rws := rewardserver.NewRewardServer(&stubRewardsTxSender, nil, rewardDB)

		return rws, &stubRewardsTxSender, rewardDB
	}

	t.Run("sends payment when epoch is updated", func(t *testing.T) {
		rws, stubRewardsTxSender, _ := setupRewardsServer()
		privKey, acc := sigs.GenerateFloatingKey()

		ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
		for _, sessionId := range []uint64{1, 2, 3, 4, 5} {
			epoch := sessionId%2 + 1
			proof := common.BuildRelayRequestWithSession(ctx, "provider", []byte{}, sessionId, uint64(0), "spec", nil)
			proof.Epoch = int64(epoch)

			sig, err := sigs.Sign(privKey, *proof)
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
	})

	t.Run("does not send payment if too many epochs have passed", func(t *testing.T) {
		rws, stubRewardsTxSender, db := setupRewardsServer()
		privKey, acc := sigs.GenerateFloatingKey()

		ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
		for _, sessionId := range []uint64{1, 2, 3, 4, 5} {
			epoch := uint64(1)
			proof := common.BuildRelayRequestWithSession(ctx, "provider", []byte{}, sessionId, uint64(0), "spec", nil)
			proof.Epoch = int64(epoch)

			sig, err := sigs.Sign(privKey, *proof)
			proof.Sig = sig
			require.NoError(t, err)

			_, _ = rws.SendNewProof(context.Background(), proof, epoch, acc.String(), "apiInterface")
		}

		stubRewardsTxSender.earliestBlockInMemory = 2

		rws.UpdateEpoch(3)

		// ensure no payments have been sent
		require.Len(t, stubRewardsTxSender.sentPayments, 0)

		rewards, err := db.FindAll()
		require.NoError(t, err)
		// ensure rewards have been deleted
		require.Len(t, rewards, 0)
	})
}

func BenchmarkSendNewProofInMemory(b *testing.B) {
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	db1 := rewardserver.NewMemoryDB("spec")
	db2 := rewardserver.NewMemoryDB("spec2")
	rewardStore := rewardserver.NewRewardDB(db1)
	rewardStore.AddDB(db2)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)
	proofs := generateProofs(ctx, []string{"spec", "spec2"}, b.N)

	b.ResetTimer()
	sendProofs(ctx, proofs, rws)
}

func BenchmarkSendNewProofLocal(b *testing.B) {
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	db1 := rewardserver.NewLocalDB("badger_test", "provider", "spec", 0)
	db2 := rewardserver.NewLocalDB("badger_test", "provider", "spec2", 0)
	defer func() {
		_ = db1.Close()
		_ = db2.Close()
	}()
	rewardStore := rewardserver.NewRewardDB(db1)
	rewardStore.AddDB(db2)
	rws := rewardserver.NewRewardServer(&rewardsTxSenderDouble{}, nil, rewardStore)

	proofs := generateProofs(ctx, []string{"spec", "spec2"}, b.N)

	b.ResetTimer()
	sendProofs(ctx, proofs, rws)
}

func generateProofs(ctx context.Context, specs []string, n int) []*pairingtypes.RelaySession {
	var proofs []*pairingtypes.RelaySession
	for i := 0; i < n; i++ {
		proof := common.BuildRelayRequestWithSession(ctx, "provider", []byte{}, uint64(1), uint64(0), specs[i%2], nil)
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

type rewardsTxSenderDouble struct {
	earliestBlockInMemory uint64
	sentPayments          []*pairingtypes.RelaySession
}

func (rts *rewardsTxSenderDouble) TxRelayPayment(_ context.Context, payments []*pairingtypes.RelaySession, _ string) error {
	rts.sentPayments = append(rts.sentPayments, payments...)

	return nil
}

func (rts *rewardsTxSenderDouble) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(_ context.Context) (uint64, error) {
	return 0, nil
}

func (rts *rewardsTxSenderDouble) EarliestBlockInMemory(_ context.Context) (uint64, error) {
	return rts.earliestBlockInMemory, nil
}
