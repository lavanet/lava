package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/testutil/common"
	testutil "github.com/lavanet/lava/v2/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
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

	err := ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		ts.Keepers.AccountKeeper.GetModuleAddress(string(rewardstypes.ValidatorsRewardsAllocationPoolName)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	err = ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		ts.Keepers.AccountKeeper.GetModuleAddress(string(rewardstypes.ProvidersRewardsAllocationPool)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	ts.DisableParticipationFees()

	ts.addValidators(1)

	ts.plan = ts.AddPlan("free", common.CreateMockPlan()).Plan("free")
	ts.spec = ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")

	ts.AdvanceEpoch()

	return ts
}

func (ts *tester) addClient(count int) {
	start := len(ts.Accounts(common.CONSUMER))
	for i := 0; i < count; i++ {
		_, addr := ts.AddAccount(common.CONSUMER, start+i, testBalance)
		_, err := ts.TxSubscriptionBuy(addr, addr, ts.plan.Index, 1, false, false)
		if err != nil {
			panic("addClient: failed to buy subscription: " + err.Error())
		}
	}
}

func (ts *tester) addValidators(count int) {
	start := len(ts.Accounts(common.VALIDATOR))
	for i := 0; i < count; i++ {
		acc, _ := ts.AddAccount(common.VALIDATOR, start+i, testBalance)
		ts.TxCreateValidator(acc, math.NewInt(testBalance))
	}
}

// addProvider: with default endpoints, geolocation, moniker
func (ts *tester) addProvider(count int) error {
	d := common.MockDescription()
	return ts.addProviderExtra(count, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details) // default: endpoints, geolocation, moniker
}

// addProviderGelocation: with geolocation, and default endpoints, moniker
func (ts *tester) addProviderGeolocation(count int, geolocation int32) error {
	d := common.MockDescription()
	return ts.addProviderExtra(count, nil, geolocation, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
}

// addProviderEndpoints: with endpoints, and default geolocation, moniker
func (ts *tester) addProviderEndpoints(count int, endpoints []epochstoragetypes.Endpoint) error {
	d := common.MockDescription()
	return ts.addProviderExtra(count, endpoints, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
}

// addProviderDescription: with description, and default endpoints, geolocation
func (ts *tester) addProviderDescription(count int, moniker string, identity string, website string, securityContact string, descriptionDetails string) error {
	return ts.addProviderExtra(count, nil, 0, moniker, identity, website, securityContact, descriptionDetails)
}

// addProviderExtra: with mock endpoints, and preset geolocation, description details
func (ts *tester) addProviderExtra(
	count int,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	moniker string,
	identity string,
	website string,
	securityContact string,
	descriptionDetails string,
) error {
	start := len(ts.Accounts(common.PROVIDER))
	for i := 0; i < count; i++ {
		acc, addr := ts.AddAccount(common.PROVIDER, start+i, testBalance)
		err := ts.StakeProviderExtra(acc.GetVaultAddr(), addr, ts.spec, testStake, endpoints, geoloc, moniker, identity, website, securityContact, descriptionDetails)
		if err != nil {
			return err
		}
	}
	return nil
}

// setupForPayments creates staked providers and clients with subscriptions. They can be accessed
// using ts.Account(common.PROVIDER, idx) and ts.Account(common.PROVIDER, idx) respectively.
func (ts *tester) setupForPayments(providersCount, clientsCount, providersToPair int) *tester {
	err := ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		testutil.GetModuleAddress(string(rewardstypes.ValidatorsRewardsAllocationPoolName)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	err = ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		testutil.GetModuleAddress(string(rewardstypes.ProvidersRewardsAllocationPool)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	ts.addValidators(1)
	if providersToPair > 0 {
		// will overwrite the default "free" plan
		ts.plan.PlanPolicy.MaxProvidersToPair = uint64(providersToPair)
		ts.AddPlan("free", ts.plan)
	}

	ts.addClient(clientsCount)
	err = ts.addProvider(providersCount)
	require.Nil(ts.T, err)

	ts.AdvanceEpoch()

	return ts
}

// payAndVerifyBalance performs payment and then verifies the balances
// (provider balance should increase and consumer should decrease)
// The providerRewardPerc arg is the part of the provider reward after dedcuting
// the delegators portion (in percentage)
func (ts *tester) payAndVerifyBalance(
	relayPayment pairingtypes.MsgRelayPayment,
	clientAddr sdk.AccAddress,
	providerVault sdk.AccAddress,
	validConsumer bool,
	validPayment bool,
	providerRewardPerc uint64,
) {
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
	res, err := ts.TxPairingRelayPayment(relayPayment.Creator, relayPayment.Relays...)
	if !validPayment {
		if err == nil {
			require.True(ts.T, res.RejectedRelays)
			return
		}
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

	// verify each project balance
	// (project used-cu should increase and respective subscription cu-left should decrease)
	proj, err = ts.QueryProjectDeveloper(clientAddr.String())
	require.Nil(ts.T, err)
	require.Equal(ts.T, originalProjectUsedCu+totalCuUsed, proj.Project.UsedCu)
	sub, err = ts.QuerySubscriptionCurrent(proj.Project.Subscription)
	require.Nil(ts.T, err)
	require.NotNil(ts.T, sub.Sub)
	require.Equal(ts.T, originalSubCuLeft-totalCuUsed, sub.Sub.MonthCuLeft)

	// advance month + blocksToSave + 1 to trigger the provider monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	// verify provider's balance
	credit := sub.Sub.Credit.Amount.QuoRaw(int64(sub.Sub.DurationLeft))
	want := sdk.ZeroInt()
	if totalCuUsed != 0 {
		want = credit.MulRaw(int64(providerReward)).QuoRaw(int64(totalCuUsed))
	}

	balanceWant := ts.GetBalance(providerVault) + want.Int64()
	reward, err := ts.QueryDualstakingDelegatorRewards(providerVault.String(), relayPayment.Creator, "")
	require.Nil(ts.T, err)
	for _, reward := range reward.Rewards {
		want = want.Sub(reward.Amount.AmountOf(ts.BondDenom()))
	}
	require.True(ts.T, want.IsZero())
	_, err = ts.TxDualstakingClaimRewards(providerVault.String(), relayPayment.Creator)
	require.Nil(ts.T, err)

	balance := ts.GetBalance(providerVault) + want.Int64()
	require.Equal(ts.T, balanceWant, balance)
}

// verifyRelayPayments verifies relay payments saved on-chain after getting payment
func (ts *tester) verifyRelayPayment(relaySession *pairingtypes.RelaySession, exists bool) {
	epoch := uint64(relaySession.Epoch)
	chainID := ts.spec.Name
	cu := relaySession.CuSum
	sessionID := relaySession.SessionId

	// note: assume a single client and a single provider, so these make sense:
	_, consumer := ts.GetAccount(common.CONSUMER, 0)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	project, err := ts.GetProjectForDeveloper(consumer, epoch)
	require.NoError(ts.T, err)

	found := ts.Keepers.Pairing.IsUniqueEpochSessionExists(ts.Ctx, epoch, provider, project.Index, chainID, sessionID)
	require.Equal(ts.T, exists, found)

	pec, found := ts.Keepers.Pairing.GetProviderEpochCu(ts.Ctx, epoch, provider, chainID)
	require.Equal(ts.T, exists, found)
	if exists {
		require.GreaterOrEqual(ts.T, pec.ServicedCu, cu)
	}

	pcec, found := ts.Keepers.Pairing.GetProviderConsumerEpochCu(ts.Ctx, epoch, provider, project.Index, chainID)
	require.Equal(ts.T, exists, found)
	if exists {
		require.GreaterOrEqual(ts.T, pcec.Cu, cu)
	}
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
		Epoch:       int64(ts.EpochStart(epoch)),
		RelayNum:    relay,
	}
	return relaySession
}
