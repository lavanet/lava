package rewardserver

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/lavanet/lava/v2/utils/sigs"
	"golang.org/x/net/context"

	terderminttypes "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func stubPaymentEvents(num int, specId string, sessionId uint64) (tos []map[string]string) {
	generateOne := func(uniqueIdentifier uint64) (to map[string]string) {
		// stub data
		relay := pairingtypes.RelaySession{
			SpecId:      specId,
			ContentHash: []byte{},
			SessionId:   sessionId,
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
		from["epoch"] = strconv.FormatUint(uint64(relay.Epoch), 10)
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

func paymentEventsToEventAttributes(attributesList []map[string]string) (eventAttrs []terderminttypes.EventAttribute) {
	for _, attributes := range attributesList {
		for key, val := range attributes {
			eventAttrs = append(eventAttrs, terderminttypes.EventAttribute{Key: key, Value: val})
		}
	}
	return
}

func deepCompareRewards(t *testing.T, expected map[uint64]*EpochRewards, actual map[uint64]*EpochRewards) {
	for epoch, epochReward := range expected {
		epochRewardActual, found := actual[epoch]
		require.True(t, found)

		for consumerKey, consumerRewards := range epochReward.consumerRewards {
			consumerRewardActual, found := epochRewardActual.consumerRewards[consumerKey]
			require.True(t, found)

			for sessionId, proof := range consumerRewards.proofs {
				proofActual, found := consumerRewardActual.proofs[sessionId]
				require.True(t, found)

				require.Equal(t, proof.CuSum, proofActual.CuSum)
				require.Equal(t, proof.LavaChainId, proofActual.LavaChainId)
				require.Equal(t, proof.RelayNum, proofActual.RelayNum)
				require.Equal(t, proof.SessionId, proofActual.SessionId)
				require.Equal(t, proof.Sig, proofActual.Sig)
			}
		}
	}
}

func createInMemoryRewardDb(specs []string) (*RewardDB, error) {
	rewardDB := NewRewardDB()
	for _, spec := range specs {
		db := NewMemoryDB(spec)
		err := rewardDB.AddDB(db)
		if err != nil {
			return nil, err
		}
	}
	return rewardDB, nil
}

func TestPayments(t *testing.T) {
	internalRelays := 5
	attributesList := stubPaymentEvents(internalRelays, "spec", 1)
	eventAttrs := paymentEventsToEventAttributes(attributesList)
	event := terderminttypes.Event{Type: utils.EventPrefix + pairingtypes.RelayPaymentEventName, Attributes: eventAttrs}
	paymentRequests, err := BuildPaymentFromRelayPaymentEvent(event, 500)
	require.NoError(t, err)
	require.Equal(t, internalRelays, len(paymentRequests))
}

func TestSendNewProof(t *testing.T) {
	rand.InitRandomSeed()
	const providerAddr = "providerAddr"
	specId := "specId"
	rewardDB, err := createInMemoryRewardDb([]string{specId, "newSpec"})
	require.NoError(t, err)

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
		rws := NewRewardServer(&rewardsTxSenderMock{}, nil, rewardDB, "badger_test", 1, 20, nil)
		existingCU, updatedWithProf := uint64(0), false
		for _, proof := range testCase.Proofs {
			existingCU, updatedWithProf = rws.SendNewProof(context.TODO(), proof, uint64(proof.Epoch), "consumerAddress", "apiInterface")
		}
		require.Equal(t, testCase.ExpectedExistingCu, existingCU)
		require.Equal(t, testCase.ExpectedUpdatedWithProof, updatedWithProf)
	}
}

func TestSendNewProofWillSetBadgeWhenPrefProofDoesNotHaveOneSet(t *testing.T) {
	rand.InitRandomSeed()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	rewardDB, err := createInMemoryRewardDb([]string{"specId"})
	require.NoError(t, err)

	rws := NewRewardServer(&rewardsTxSenderMock{}, nil, rewardDB, "badger_test", 1, 10, nil)

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
	rand.InitRandomSeed()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	db := NewMemoryDB("specId")
	rewardStore := NewRewardDB()
	err := rewardStore.AddDB(db)
	require.NoError(t, err)

	rws := NewRewardServer(&rewardsTxSenderMock{}, nil, rewardStore, "badger_test", 1, 10, nil)

	const providerAddr = "providerAddr"
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
	rand.InitRandomSeed()
	setupRewardsServer := func() (*RewardServer, *rewardsTxSenderMock, *RewardDB) {
		stubRewardsTxSender := rewardsTxSenderMock{}
		rewardDB, err := createInMemoryRewardDb([]string{"spec"})
		require.NoError(t, err)

		rws := NewRewardServer(&stubRewardsTxSender, nil, rewardDB, "badger_test", 1, 10, nil)

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

		// Make sure that the rewards are flushed to DB
		rws.resetSnapshotTimerAndSaveRewardsSnapshotToDB()

		rws.runRewardServerEpochUpdate(1)

		// 2 payments for epoch 1
		require.Len(t, stubRewardsTxSender.sentPayments, 2)

		rws.runRewardServerEpochUpdate(2)

		// another 3 payments for epoch 2
		require.Len(t, stubRewardsTxSender.sentPayments, 5)
	})

	t.Run("does not send payment if too many epochs have passed", func(t *testing.T) {
		rws, stubRewardsTxSender, db := setupRewardsServer()
		privKey, acc := sigs.GenerateFloatingKey()

		ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
		epoch := uint64(1)
		for _, sessionId := range []uint64{1, 2, 3, 4, 5} {
			proof := common.BuildRelayRequestWithSession(ctx, "provider", []byte{}, sessionId, uint64(0), "spec", nil)
			proof.Epoch = int64(epoch)

			sig, err := sigs.Sign(privKey, *proof)
			proof.Sig = sig
			require.NoError(t, err)

			_, _ = rws.SendNewProof(context.Background(), proof, epoch, acc.String(), "apiInterface")
		}

		// Make sure that the rewards are flushed to DB
		rws.resetSnapshotTimerAndSaveRewardsSnapshotToDB()

		stubRewardsTxSender.earliestBlockInMemory = 2

		rws.runRewardServerEpochUpdate(3)
		// ensure no payments have been sent
		require.Len(t, stubRewardsTxSender.sentPayments, 0)
		rewards, err := db.FindAll()
		require.NoError(t, err)
		// ensure rewards have been deleted
		require.Len(t, rewards, 0)
	})
}

func TestSaveRewardsToDB(t *testing.T) {
	rand.InitRandomSeed()
	const providerAddr = "providerAddr"
	specs := []string{"spec1", "spec2"}

	rewardDB, err := createInMemoryRewardDb(specs)
	require.NoError(t, err)

	rws := NewRewardServer(&rewardsTxSenderMock{}, nil, rewardDB, "badger_test", 2, 1000, nil)

	epoch := uint64(1)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proofs := []*pairingtypes.RelaySession{}
	for _, spec := range specs {
		proofs = append(proofs, common.BuildRelayRequestWithSession(ctx, providerAddr, make([]byte, 0), uint64(1), uint64(10), spec, nil))
	}

	for _, proof := range proofs {
		_, _ = rws.SendNewProof(context.TODO(), proof, epoch, "consumerAddress", "apiInterface")
	}

	rws.rewardsSnapshotThresholdCh <- struct{}{}

	epochRewardsDB, err := rewardDB.FindAll()
	require.NoError(t, err)

	deepCompareRewards(t, rws.rewards, epochRewardsDB)
}

func TestDeleteRewardsFromDBWhenRewardApproved(t *testing.T) {
	rand.InitRandomSeed()
	const providerAddr = "providerAddr"
	specs := []string{"spec1", "spec2"}

	rewardDB, err := createInMemoryRewardDb(specs)
	require.NoError(t, err)

	rws := NewRewardServer(&rewardsTxSenderMock{}, nil, rewardDB, "badger_test", 1, 100, nil)

	epoch, sessionId := uint64(1), uint64(1)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proofs := []*pairingtypes.RelaySession{}
	for _, spec := range specs {
		proofs = append(proofs, common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, sessionId, uint64(10), spec, nil))
	}

	for _, proof := range proofs {
		_, _ = rws.SendNewProof(context.TODO(), proof, epoch, "consumerAddress", "apiInterface")
	}

	rws.rewardsSnapshotThresholdCh <- struct{}{}

	epochRewardsDB, err := rewardDB.FindAll()
	require.NoError(t, err)
	deepCompareRewards(t, rws.rewards, epochRewardsDB)

	for _, spec := range specs {
		attributesList := stubPaymentEvents(1, spec, sessionId)
		eventAttributes := paymentEventsToEventAttributes(attributesList)
		event := terderminttypes.Event{Type: utils.EventPrefix + pairingtypes.RelayPaymentEventName, Attributes: eventAttributes}
		paymentRequests, err := BuildPaymentFromRelayPaymentEvent(event, int64(epoch)+5)
		require.NoError(t, err)

		rws.PaymentHandler(paymentRequests[0])
	}

	epochRewardsDB, err = rewardDB.FindAll()
	require.NoError(t, err)
	deepCompareRewards(t, rws.rewards, epochRewardsDB)
}

func TestDeleteRewardsFromDBWhenRewardEpochNotInMemory(t *testing.T) {
	rand.InitRandomSeed()
	const providerAddr = "providerAddr"
	specs := []string{"LAV1", "ETH1"}

	rewardDB, err := createInMemoryRewardDb(specs)
	require.NoError(t, err)

	stubRewardsTxSender := rewardsTxSenderMock{}

	rws := NewRewardServer(&stubRewardsTxSender, nil, rewardDB, "badger_test", 1, 100, nil)

	epoch, sessionId := uint64(1), uint64(1)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proofs := []*pairingtypes.RelaySession{}
	for _, chainId := range specs {
		proofs = append(proofs, common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, sessionId, uint64(10), chainId, nil))
	}

	for _, proof := range proofs {
		_, _ = rws.SendNewProof(context.TODO(), proof, epoch, "consumerAddress", "apiInterface")
	}

	rws.rewardsSnapshotThresholdCh <- struct{}{}

	epochRewardsDB, err := rewardDB.FindAll()
	require.NoError(t, err)

	deepCompareRewards(t, rws.rewards, epochRewardsDB)

	newEpoch := epoch + 1
	stubRewardsTxSender.earliestBlockInMemory = newEpoch
	rws.runRewardServerEpochUpdate(newEpoch)
	for _, chainId := range specs {
		epochRewards, err := rewardDB.FindAllInDB(chainId)
		require.NoError(t, err)

		_, found := epochRewards[epoch]
		require.False(t, found)
	}
}

func TestRestoreRewardsFromDB(t *testing.T) {
	rand.InitRandomSeed()
	const providerAddr = "providerAddr"
	specs := []string{"spec1", "spec2"}

	rewardDB, err := createInMemoryRewardDb(specs)
	require.NoError(t, err)

	stubRewardsTxSender := rewardsTxSenderMock{}

	rws := NewRewardServer(&stubRewardsTxSender, nil, rewardDB, "badger_test", 1, 1, nil)

	epoch, sessionId := uint64(1), uint64(1)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	privKey, acc := sigs.GenerateFloatingKey()
	for _, spec := range specs {
		proof := common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, sessionId, uint64(10), spec, nil)
		proof.Epoch = int64(epoch)

		sig, err := sigs.Sign(privKey, *proof)
		require.NoError(t, err)
		proof.Sig = sig

		_, _ = rws.SendNewProof(context.TODO(), proof, epoch, acc.String(), "apiInterface")
	}

	rws.rewardsSnapshotThresholdCh <- struct{}{}

	stubRewardsTxSender = rewardsTxSenderMock{}
	rws = NewRewardServer(&stubRewardsTxSender, nil, rewardDB, "badger_test", 1, 1, nil)

	for _, spec := range specs {
		rws.restoreRewardsFromDB(spec)
	}

	rws.runRewardServerEpochUpdate(epoch + 3)
	require.Equal(t, 2, len(stubRewardsTxSender.sentPayments))
}

func TestFailedPaymentRequestAttemptsHappyFlow(t *testing.T) {
	rand.InitRandomSeed()
	const providerAddr = "providerAddr"
	spec := "spec1"
	var rewardTxSent int

	rewardDB, err := createInMemoryRewardDb([]string{spec})
	require.NoError(t, err)

	stubRewardsTxSender := rewardsTxSenderMock{
		txRelayPaymentCallback: func(ctx context.Context, rs []*pairingtypes.RelaySession, s string, lbr []*pairingtypes.LatestBlockReport) error {
			rewardTxSent++
			return fmt.Errorf("Some error")
		},
	}

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	rws := NewRewardServer(&stubRewardsTxSender, nil, rewardDB, "badger_test", 1, 1, nil)

	session := common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, uint64(1), uint64(42), spec, nil)
	rws.SendNewProof(ctx, session, 1, "consumerAddress", "apiInterface")

	for i := 0; i < MaxPaymentRequestsRetiresForSession-1; i++ {
		rws.sendRewardsClaim(ctx, 1)
		require.Equal(t, 1, len(rws.failedRewardsPaymentRequests))
	}

	rws.sendRewardsClaim(ctx, 1)
	require.Equal(t, 0, len(rws.failedRewardsPaymentRequests))
	require.Equal(t, MaxPaymentRequestsRetiresForSession, rewardTxSent)
}

func TestFailedPaymentRequestAttemptsHappyMultipleSessions(t *testing.T) {
	rand.InitRandomSeed()
	const providerAddr = "providerAddr"
	spec := "spec1"
	var rewardTxSent int

	rewardDB, err := createInMemoryRewardDb([]string{spec})
	require.NoError(t, err)

	stubRewardsTxSender := rewardsTxSenderMock{
		txRelayPaymentCallback: func(ctx context.Context, rs []*pairingtypes.RelaySession, s string, lbr []*pairingtypes.LatestBlockReport) error {
			rewardTxSent += len(rs)
			return fmt.Errorf("Some error")
		},
	}

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	rws := NewRewardServer(&stubRewardsTxSender, nil, rewardDB, "badger_test", 10000, 10000, nil)

	require.Equal(t, 3, MaxPaymentRequestsRetiresForSession,
		"This test assumes that the MaxPaymentRequestsRetiresForSession is 3. "+
			"If that's not the case, please change the test accordingly")

	privKey, acc := sigs.GenerateFloatingKey()

	buildAndSendRelayPaymentRequest := func(sessionId uint64, epoch int64) {
		session := common.BuildRelayRequestWithSession(ctx, providerAddr, []byte{}, sessionId, uint64(42), spec, nil)
		session.Epoch = epoch
		sig, err := sigs.Sign(privKey, *session)
		require.NoError(t, err)
		session.Sig = sig
		rws.SendNewProof(ctx, session, 1, acc.String(), "apiInterface")
		rws.sendRewardsClaim(ctx, 1)
	}

	buildAndSendRelayPaymentRequest(1, 1)
	require.Equal(t, 1, rewardTxSent)
	require.Equal(t, 1, len(rws.failedRewardsPaymentRequests)) // sessions: [1]
	require.Equal(t, uint64(1), rws.failedRewardsPaymentRequests[1].relaySession.SessionId)

	buildAndSendRelayPaymentRequest(2, 1)
	require.Equal(t, 3, rewardTxSent)
	require.Equal(t, 2, len(rws.failedRewardsPaymentRequests)) // sessions: [1, 2]
	require.Equal(t, uint64(1), rws.failedRewardsPaymentRequests[1].relaySession.SessionId)
	require.Equal(t, uint64(2), rws.failedRewardsPaymentRequests[2].relaySession.SessionId)

	buildAndSendRelayPaymentRequest(3, 2)
	require.Equal(t, 6, rewardTxSent)
	require.Equal(t, 2, len(rws.failedRewardsPaymentRequests)) // sessions: [2, 3]
	require.Equal(t, uint64(2), rws.failedRewardsPaymentRequests[2].relaySession.SessionId)
	require.Equal(t, uint64(3), rws.failedRewardsPaymentRequests[3].relaySession.SessionId)

	rws.sendRewardsClaim(ctx, 3)
	require.Equal(t, 8, rewardTxSent)
	require.Equal(t, 1, len(rws.failedRewardsPaymentRequests)) // sessions: [3]
	require.Equal(t, uint64(3), rws.failedRewardsPaymentRequests[3].relaySession.SessionId)

	rws.sendRewardsClaim(ctx, 5)
	require.Equal(t, 9, rewardTxSent)
	require.Equal(t, 0, len(rws.failedRewardsPaymentRequests)) // No sessions should be left
}

func BenchmarkSendNewProofInMemory(b *testing.B) {
	rand.InitRandomSeed()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	specs := []string{"spec", "spec2"}
	rewardDB, err := createInMemoryRewardDb(specs)
	require.NoError(b, err)

	rws := NewRewardServer(&rewardsTxSenderMock{}, nil, rewardDB, "badger_test", 1, 10, nil)
	proofs := generateProofs(ctx, specs, b.N)

	b.ResetTimer()
	sendProofs(ctx, proofs, rws)
}

func BenchmarkSendNewProofLocal(b *testing.B) {
	rand.InitRandomSeed()
	os.RemoveAll("badger_test")

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	db1 := NewLocalDB("badger_test", "provider", "spec", 0)
	db2 := NewLocalDB("badger_test", "provider", "spec2", 0)
	rewardStore := NewRewardDB()
	err := rewardStore.AddDB(db1)
	require.NoError(b, err)

	err = rewardStore.AddDB(db2)
	require.NoError(b, err)

	defer func() {
		rewardStore.Close()
	}()
	rws := NewRewardServer(&rewardsTxSenderMock{}, nil, rewardStore, "badger_test", 1, 10, nil)

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

func sendProofs(ctx context.Context, proofs []*pairingtypes.RelaySession, rws *RewardServer) {
	for index, proof := range proofs {
		prefix := index%2 + 1
		consumerKey := fmt.Sprintf("consumer%d", prefix)
		apiInterface := fmt.Sprintf("apiInterface%d", prefix)
		rws.SendNewProof(ctx, proof, uint64(proof.Epoch), consumerKey, apiInterface)
	}
}

type rewardsTxSenderMock struct {
	earliestBlockInMemory  uint64
	sentPayments           []*pairingtypes.RelaySession
	txRelayPaymentCallback func(context.Context, []*pairingtypes.RelaySession, string, []*pairingtypes.LatestBlockReport) error
}

func (rts *rewardsTxSenderMock) defaultTxRelayPaymentCallback(_ context.Context, payments []*pairingtypes.RelaySession, _ string, _ []*pairingtypes.LatestBlockReport) error {
	rts.sentPayments = append(rts.sentPayments, payments...)

	return nil
}

func (rts *rewardsTxSenderMock) TxRelayPayment(ctx context.Context, payments []*pairingtypes.RelaySession,
	description string, latestBlocks []*pairingtypes.LatestBlockReport,
) error {
	if rts.txRelayPaymentCallback != nil {
		return rts.txRelayPaymentCallback(ctx, payments, description, latestBlocks)
	}

	return rts.defaultTxRelayPaymentCallback(ctx, payments, description, latestBlocks)
}

func (rts *rewardsTxSenderMock) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(_ context.Context) (uint64, error) {
	return 0, nil
}

func (rts *rewardsTxSenderMock) LatestBlock() int64 {
	return 0
}

func (rts *rewardsTxSenderMock) GetEpochSize(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (rts *rewardsTxSenderMock) GetAverageBlockTime() time.Duration {
	return time.Second
}

func (rts *rewardsTxSenderMock) EarliestBlockInMemory(_ context.Context) (uint64, error) {
	return rts.earliestBlockInMemory, nil
}
