package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/utils/slices"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// TestProviderDelegatorsRewards tests that the provider's reward (considering delegations) is as expected
// Also, it checks that the delegator reward map is updated as expected
func TestProviderDelegatorsRewards(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 1)                   // 1 provider, 1 client, 1 providersToPair
	ts.AddAccount(common.CONSUMER, 1, testBalance) // add delegator1
	ts.AddAccount(common.CONSUMER, 2, testBalance) // add delegator2

	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, delegator1 := ts.GetAccount(common.CONSUMER, 1)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 2)

	ts.AdvanceEpoch() // to apply pairing

	// ** check provider reward without delegators ** //

	relayPaymentMessage := sendRelay(ts, provider, clientAcc)
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, nil)

	// ** check provider reward with delegators ** //

	delegationAmount1 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(100))
	delegationAmount2 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(200))

	expectedDelegation1 := dualstakingtypes.NewDelegation(delegator1, provider, ts.spec.Index)
	expectedDelegation2 := dualstakingtypes.NewDelegation(delegator2, provider, ts.spec.Index)
	expectedDelegations := []dualstakingtypes.Delegation{expectedDelegation1, expectedDelegation2}

	_, err := ts.TxDualstakingDelegate(delegator1, provider, ts.spec.Index, delegationAmount1)
	require.Nil(t, err)
	_, err = ts.TxDualstakingDelegate(delegator2, provider, ts.spec.Index, delegationAmount2)
	require.Nil(t, err)
	ts.AdvanceEpoch() // apply delegations

	stakeEntry, found, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
	require.True(t, found)

	res, err := ts.QueryDualstakingProviderDelegators(provider, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Delegations))

	relayPaymentMessage = sendRelay(ts, provider, clientAcc)
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)

	totalDelegations := stakeEntry.DelegateTotal.Amount
	_, delegatorsReward := ts.Keepers.Dualstaking.CalcRewards(stakeEntry, math.NewInt(int64(relayCuSum)))

	// ** verify the delegator reward map updates correctly for new entries ** //

	for _, delegation := range expectedDelegations {
		ind := dualstakingtypes.DelegationKey(provider, delegation.Delegator, ts.spec.Index)
		dReward, found := ts.Keepers.Dualstaking.GetDelegatorReward(ts.Ctx, ind)
		require.True(t, found)
		expectedDReward := ts.Keepers.Dualstaking.CalcDelegatorReward(delegatorsReward, totalDelegations, delegation)
		require.True(t, expectedDReward.Equal(dReward.Amount.Amount))
	}

	// ** verify the delegator reward map updates correctly for existing entries ** //

	relayPaymentMessage = sendRelay(ts, provider, clientAcc)
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)

	for _, delegation := range expectedDelegations {
		ind := dualstakingtypes.DelegationKey(provider, delegation.Delegator, ts.spec.Index)
		dReward, found := ts.Keepers.Dualstaking.GetDelegatorReward(ts.Ctx, ind)
		require.True(t, found)
		expectedDReward := ts.Keepers.Dualstaking.CalcDelegatorReward(delegatorsReward, totalDelegations, delegation)
		// the reward saved on the map should be doubled since it's the second time we send relay
		require.True(t, expectedDReward.MulRaw(2).Equal(dReward.Amount.Amount))
	}
}

var (
	sessionID  uint64
	relayCuSum = uint64(100)
)

func sendRelay(ts *tester, provider string, clientAcc common.Account) types.MsgRelayPayment {
	// Create relay request. Change session ID each call to avoid double spending error
	relaySession := ts.newRelaySession(provider, sessionID, relayCuSum, ts.BlockHeight(), 0)
	sessionID += 1

	// Sign and send the payment requests
	sig, err := sigs.Sign(clientAcc.SK, *relaySession)
	relaySession.Sig = sig
	require.Nil(ts.T, err)

	return types.MsgRelayPayment{Creator: provider, Relays: slices.Slice(relaySession)}
}

// TestDelegationLimitAffectingProviderReward checks that the delegation limit influences the provider's reward
func TestDelegationLimitAffectingProviderReward(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 1)                   // 1 provider, 1 client, 1 providersToPair
	ts.AddAccount(common.CONSUMER, 1, testBalance) // add delegator1
	ts.AddAccount(common.CONSUMER, 2, testBalance) // add delegator2

	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, delegator1 := ts.GetAccount(common.CONSUMER, 1)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 2)

	ts.AdvanceEpoch() // to apply pairing

	delegationAmount1 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(uint64(testStake)/2))
	delegationAmount2 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(uint64(testStake)))

	_, err := ts.TxDualstakingDelegate(delegator1, provider, ts.spec.Index, delegationAmount1)
	require.Nil(t, err)
	_, err = ts.TxDualstakingDelegate(delegator2, provider, ts.spec.Index, delegationAmount2)
	require.Nil(t, err)
	ts.AdvanceEpoch() // apply delegations

	stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
	require.True(t, found)

	// modify the stake entry to have a delegation limit higher than the total delegations
	stakeEntry.DelegateLimit = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(2*testStake))
	stakeEntry.DelegateCommission = 50
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()
	stakeEntry, found, stakeEntryIndex = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
	require.True(t, found)

	res, err := ts.QueryDualstakingProviderDelegators(provider, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Delegations))

	// check the provider's balance
	providerReward, _ := ts.Keepers.Dualstaking.CalcRewards(stakeEntry, math.NewInt(int64(relayCuSum)))
	mint := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx)
	expectedReward := mint.MulInt64(providerReward.Int64())

	balance := ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage := sendRelay(ts, provider, clientAcc)
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)
	newBalance := ts.GetBalance(providerAcc.Addr)

	require.Equal(t, expectedReward.TruncateInt64(), newBalance-balance)

	// modify the stake entry to have a delegation limit lower than the total delegations
	stakeEntry.DelegateLimit = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(testStake))
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	providerReward, _ = ts.Keepers.Dualstaking.CalcRewards(stakeEntry, math.NewInt(int64(relayCuSum)))
	expectedReward = mint.MulInt64(providerReward.Int64())

	balance = ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage = sendRelay(ts, provider, clientAcc)
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)
	newBalance = ts.GetBalance(providerAcc.Addr)
	require.Equal(t, expectedReward.TruncateInt64(), newBalance-balance)
}

func TestProviderRewardWithCommission(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 1)                   // 1 provider, 1 client, 1 providersToPair
	ts.AddAccount(common.CONSUMER, 1, testBalance) // add delegator1
	ts.AddAccount(common.CONSUMER, 2, testBalance) // add delegator2

	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, delegator1 := ts.GetAccount(common.CONSUMER, 1)

	ts.AdvanceEpoch() // to apply pairing

	delegationAmount1 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(testStake))
	_, err := ts.TxDualstakingDelegate(delegator1, provider, ts.spec.Index, delegationAmount1)
	require.Nil(t, err)
	ts.AdvanceEpoch() // apply delegations

	stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
	require.True(t, found)

	// ** provider's commission is 100% ** //

	stakeEntry.DelegateCommission = 100
	stakeEntry.DelegateLimit = delegationAmount1
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()
	stakeEntry, found, stakeEntryIndex = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
	require.True(t, found)

	res, err := ts.QueryDualstakingProviderDelegators(provider, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Delegations))

	// the expected reward for the provider with 100% commission is the total rewards (delegators get nothing)
	totalReward := math.NewInt(int64(relayCuSum))
	providerReward, _ := ts.Keepers.Dualstaking.CalcRewards(stakeEntry, totalReward)

	require.True(t, totalReward.Equal(providerReward))

	// check that the expected reward equals to the provider's new balance minus old balance
	mint := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx)
	expectedRewardForRelay := mint.MulInt64(providerReward.Int64())

	balance := ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage := sendRelay(ts, provider, clientAcc)
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)
	newBalance := ts.GetBalance(providerAcc.Addr)
	require.Equal(t, expectedRewardForRelay.TruncateInt64(), newBalance-balance)

	// the delegator should get no rewards
	ind := dualstakingtypes.DelegationKey(provider, delegator1, ts.spec.Index)
	dReward, found := ts.Keepers.Dualstaking.GetDelegatorReward(ts.Ctx, ind)
	require.True(t, found)
	expectedDReward := math.ZeroInt()
	require.True(t, expectedDReward.Equal(dReward.Amount.Amount))

	// ** provider's commission is 0% ** //

	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	// the expected reward for the provider with 0% commission is half of the total rewards
	// (in this test specifically, effectiveDelegations = delegateTotal = providerStake)
	balance = ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage = sendRelay(ts, provider, clientAcc)
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)
	newBalance = ts.GetBalance(providerAcc.Addr)
	require.Equal(t, expectedRewardForRelay.TruncateInt64()/2, newBalance-balance)

	// the delegator should get the half of the total rewards
	dReward, found = ts.Keepers.Dualstaking.GetDelegatorReward(ts.Ctx, ind)
	require.True(t, found)
	expectedDRewardForRelay := mint.MulInt64(totalReward.Int64())
	require.Equal(t, expectedDRewardForRelay.TruncateInt64()/2, dReward.Amount.Amount.Int64())
}
