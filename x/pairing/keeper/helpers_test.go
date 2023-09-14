package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type tester struct {
	common.Tester
	plan planstypes.Plan
	spec spectypes.Spec
}

const (
	testBalance int64 = 1000000
	testStake   int64 = 100000
)

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}

	ts.plan = ts.AddPlan("free", common.CreateMockPlan()).Plan("free")
	ts.spec = ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")

	ts.AdvanceEpoch()

	return ts
}

func (ts *tester) addClient(count int) {
	start := len(ts.Accounts(common.CONSUMER))
	for i := 0; i < count; i++ {
		_, addr := ts.AddAccount(common.CONSUMER, start+i, testBalance)
		_, err := ts.TxSubscriptionBuy(addr, addr, ts.plan.Index, 1)
		if err != nil {
			panic("addClient: failed to buy subscription: " + err.Error())
		}
	}
}

// addProvider: with default endpoints, geolocation, moniker
func (ts *tester) addProvider(count int) error {
	return ts.addProviderExtra(count, nil, 0, "prov") // default: endpoints, geolocation, moniker
}

// addProviderGelocation: with geolocation, and default endpoints, moniker
func (ts *tester) addProviderGeolocation(count int, geolocation int32) error {
	return ts.addProviderExtra(count, nil, geolocation, "prov")
}

// addProviderEndpoints: with endpoints, and default geolocation, moniker
func (ts *tester) addProviderEndpoints(count int, endpoints []epochstoragetypes.Endpoint) error {
	return ts.addProviderExtra(count, endpoints, 0, "prov")
}

// addProviderMoniker: with moniker, and default endpoints, geolocation
func (ts *tester) addProviderMoniker(count int, moniker string) error {
	return ts.addProviderExtra(count, nil, 0, moniker)
}

// addProviderExtra: with mock endpoints, and preset geolocation, moniker
func (ts *tester) addProviderExtra(
	count int,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	moniker string,
) error {
	start := len(ts.Accounts(common.PROVIDER))
	for i := 0; i < count; i++ {
		_, addr := ts.AddAccount(common.PROVIDER, start+i, testBalance)
		err := ts.StakeProviderExtra(addr, ts.spec, testStake, endpoints, geoloc, moniker)
		if err != nil {
			return err
		}
	}
	return nil
}

// setupForPayments creates staked providers and clients with subscriptions. They can be accessed
// using ts.Account(common.PROVIDER, idx) and ts.Account(common.PROVIDER, idx) respectively.
func (ts *tester) setupForPayments(providersCount, clientsCount, providersToPair int) *tester {
	if providersToPair > 0 {
		// will overwrite the default "free" plan
		ts.plan.PlanPolicy.MaxProvidersToPair = uint64(providersToPair)
		ts.AddPlan("free", ts.plan)
	}

	ts.addClient(clientsCount)
	err := ts.addProvider(providersCount)
	require.Nil(ts.T, err)

	ts.AdvanceEpoch()

	return ts
}

func newStubRelayRequest(relaySession *pairingtypes.RelaySession) *pairingtypes.RelayRequest {
	req := &pairingtypes.RelayRequest{
		RelaySession: relaySession,
		RelayData:    &pairingtypes.RelayPrivateData{Data: []byte("stub-data")},
	}
	return req
}

// payAndVerifyBalance performs payment and then verifies the balances
// (provider balance should increase and consumer should decrease)
// The providerRewardPerc arg is the part of the provider reward after dedcuting
// the delegators portion (in percentage)
func (ts *tester) payAndVerifyBalance(
	relayPayment pairingtypes.MsgRelayPayment,
	clientAddr sdk.AccAddress,
	providerAddr sdk.AccAddress,
	validConsumer bool,
	validPayment bool,
	providerRewardPerc uint64,
) {
	// get consumer's project and subscription before payment
	balance := ts.GetBalance(providerAddr)

	proj, err := ts.QueryProjectDeveloper(clientAddr.String())
	if !validConsumer {
		require.NotNil(ts.T, err)
		_, err = ts.TxPairingRelayPayment(relayPayment.Creator, relayPayment.Relays[0])
		require.NotNil(ts.T, err)
		return
	}
	// else: valid consumer
	require.Nil(ts.T, err)

	sub, err := ts.QuerySubscriptionCurrent(proj.Project.Subscription)
	require.Nil(ts.T, err)
	require.NotNil(ts.T, sub.Sub)

	originalProjectUsedCu := proj.Project.UsedCu
	originalSubCuLeft := sub.Sub.MonthCuLeft

	// perform payment
	_, err = ts.TxPairingRelayPayment(relayPayment.Creator, relayPayment.Relays...)
	if !validPayment {
		require.NotNil(ts.T, err)
		return
	}
	// else: valid payment
	require.Nil(ts.T, err)

	// calculate total used CU
	var totalCuUsed uint64
	var totalPaid uint64

	qosWeight := ts.Keepers.Pairing.QoSWeight(ts.Ctx)
	qosWeightComplement := sdk.OneDec().Sub(qosWeight)

	for _, relay := range relayPayment.Relays {
		cuUsed := relay.CuSum
		totalCuUsed += cuUsed
		if relay.QosReport != nil {
			score, err := relay.QosReport.ComputeQoS()
			require.Nil(ts.T, err)

			cuUsed = score.
				Mul(qosWeight).
				Add(qosWeightComplement).
				MulInt64(int64(cuUsed)).
				TruncateInt().
				Uint64()
		}

		totalPaid += cuUsed
	}

	providerReward := (totalPaid * providerRewardPerc) / 100

	// verify provider's balance
	mint := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx)
	want := mint.MulInt64(int64(providerReward))
	expectedReward := balance + want.TruncateInt64()
	actualReward := ts.GetBalance(providerAddr)
	require.Equal(ts.T, expectedReward, actualReward)

	// verify each project balance
	// (project used-cu should increase and respective subscription cu-left should decrease)
	proj, err = ts.QueryProjectDeveloper(clientAddr.String())
	require.Nil(ts.T, err)
	require.Equal(ts.T, originalProjectUsedCu+totalCuUsed, proj.Project.UsedCu)
	sub, err = ts.QuerySubscriptionCurrent(proj.Project.Subscription)
	require.Nil(ts.T, err)
	require.NotNil(ts.T, sub.Sub)
	require.Equal(ts.T, originalSubCuLeft-totalCuUsed, sub.Sub.MonthCuLeft)
}

// verifyRelayPayments verifies relay payments saved on-chain after getting payment
func (ts *tester) verifyRelayPayment(relaySession *pairingtypes.RelaySession, exists bool) {
	epoch := uint64(relaySession.Epoch)
	// Get EpochPayment struct from current epoch and perform basic verifications
	epochPayments, found, epochPaymentKey := ts.Keepers.Pairing.GetEpochPaymentsFromBlock(ts.Ctx, epoch)
	if exists {
		require.Equal(ts.T, true, found)
		require.Equal(ts.T, epochPaymentKey, epochPayments.Index)
	} else {
		require.Equal(ts.T, false, found)
		return
	}

	// note: assume a single client and a single provider, so these make sense:
	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, _ := ts.GetAccount(common.PROVIDER, 0)

	providerPaymentStorageKey := ts.Keepers.Pairing.GetProviderPaymentStorageKey(
		ts.Ctx, ts.spec.Name, epoch, providerAcct.Addr)

	// Get the providerPaymentStorage struct from epochPayments
	var providerPaymentStorageFromEpochPayments pairingtypes.ProviderPaymentStorage
	for _, paymentStorageKey := range epochPayments.GetProviderPaymentStorageKeys() {
		if paymentStorageKey == providerPaymentStorageKey {
			providerPaymentStorageFromEpochPayments, found = ts.Keepers.Pairing.GetProviderPaymentStorage(ts.Ctx, providerPaymentStorageKey)
			require.True(ts.T, found)
		}
	}
	require.NotEmpty(ts.T, providerPaymentStorageFromEpochPayments.Index)
	require.Equal(ts.T, epoch, providerPaymentStorageFromEpochPayments.Epoch)

	project, err := ts.QueryProjectDeveloper(client1Addr)
	require.Nil(ts.T, err)

	// Get the UniquePaymentStorageClientProvider key
	hexSessionID := strconv.FormatUint(relaySession.SessionId, 16)
	uniquePaymentStorageClientProviderKey := ts.Keepers.Pairing.EncodeUniquePaymentKey(
		ts.Ctx, project.Project.Index, providerAcct.Addr, hexSessionID, ts.spec.Name)

	// Get a uniquePaymentStorageClientProvider from providerPaymentStorageFromEpochPayments
	// (note, this is one of the uniqueXXXX structs. So usedCU was calculated above with a
	// function that takes into account all the structs)
	var uniquePaymentStorageClientProviderFromProviderPaymentStorage pairingtypes.UniquePaymentStorageClientProvider
	for _, paymentStorageKey := range providerPaymentStorageFromEpochPayments.UniquePaymentStorageClientProviderKeys {
		if paymentStorageKey == uniquePaymentStorageClientProviderKey {
			uniquePaymentStorageClientProviderFromProviderPaymentStorage, found = ts.Keepers.Pairing.GetUniquePaymentStorageClientProvider(ts.Ctx, paymentStorageKey)
			require.True(ts.T, found)
		}
	}
	require.NotEmpty(ts.T, uniquePaymentStorageClientProviderFromProviderPaymentStorage.Index)
	require.Equal(ts.T, epoch, uniquePaymentStorageClientProviderFromProviderPaymentStorage.Block)
	require.Equal(ts.T, relaySession.CuSum, uniquePaymentStorageClientProviderFromProviderPaymentStorage.UsedCU)

	// Get the providerPaymentStorage struct directly
	providerPaymentStorage, found := ts.Keepers.Pairing.GetProviderPaymentStorage(ts.Ctx, providerPaymentStorageKey)
	require.Equal(ts.T, true, found)
	require.Equal(ts.T, uint64(relaySession.Epoch), providerPaymentStorage.Epoch)

	// Get one of the UniquePaymentStorageClientProvider struct directly
	uniquePaymentStorageClientProvider, found := ts.Keepers.Pairing.GetUniquePaymentStorageClientProvider(ts.Ctx, uniquePaymentStorageClientProviderKey)
	require.Equal(ts.T, true, found)
	require.Equal(ts.T, epoch, uniquePaymentStorageClientProvider.Block)
	require.Equal(ts.T, relaySession.CuSum, uniquePaymentStorageClientProvider.UsedCU)
}

func (ts *tester) newRelaySession(
	addr string,
	session uint64,
	cusum uint64,
	epoch uint64,
	relay uint64,
) *pairingtypes.RelaySession {
	relaySession := &pairingtypes.RelaySession{
		Provider:    addr,
		ContentHash: []byte(ts.spec.ApiCollections[0].Apis[0].Name),
		SessionId:   session,
		SpecId:      ts.spec.Name,
		CuSum:       cusum,
		Epoch:       int64(epoch),
		RelayNum:    relay,
	}
	return relaySession
}
