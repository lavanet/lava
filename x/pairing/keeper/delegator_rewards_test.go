package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
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

	relayPaymentMessage := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
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

	relayPaymentMessage = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)

	totalDelegations := stakeEntry.DelegateTotal.Amount
	delegatorsReward := ts.Keepers.Dualstaking.CalcDelegatorsReward(stakeEntry, math.NewInt(int64(relayCuSum)))

	// ** verify the delegator reward map updates correctly for new entries ** //

	for _, delegation := range expectedDelegations {
		res, err := ts.QueryDualstakingDelegatorRewards(delegation.Delegator, delegation.Provider, delegation.ChainID)
		require.Nil(t, err)
		require.Equal(t, 1, len(res.Rewards))
		dReward := res.Rewards[0]
		expectedDReward := ts.Keepers.Dualstaking.CalcDelegatorReward(delegatorsReward, totalDelegations, delegation)
		require.True(t, expectedDReward.Equal(dReward.Amount.Amount))

		// claim delegator rewards and verify balance
		delegatorAddr, err := sdk.AccAddressFromBech32(delegation.Delegator)
		require.Nil(t, err)
		claimRewardsAndVerifyBalance(ts, delegatorAddr, delegation.Provider, delegation.ChainID)
	}

	// ** verify the delegator reward map updates correctly for existing entries ** //

	relayPaymentMessage = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)

	for _, delegation := range expectedDelegations {
		res, err := ts.QueryDualstakingDelegatorRewards(delegation.Delegator, delegation.Provider, delegation.ChainID)
		require.Nil(t, err)
		require.Equal(t, 1, len(res.Rewards))
		dReward := res.Rewards[0]
		expectedDReward := ts.Keepers.Dualstaking.CalcDelegatorReward(delegatorsReward, totalDelegations, delegation)
		// the reward saved on the map should be doubled since it's the second time we send relay
		require.True(t, expectedDReward.MulRaw(2).Equal(dReward.Amount.Amount))

		// claim delegator rewards and verify balance
		delegatorAddr, err := sdk.AccAddressFromBech32(delegation.Delegator)
		require.Nil(t, err)
		claimRewardsAndVerifyBalance(ts, delegatorAddr, delegation.Provider, delegation.ChainID)
	}
}

var (
	sessionID  uint64
	relayCuSum = uint64(100)
)

func sendRelay(ts *tester, provider string, clientAcc common.Account, chainIDs []string) types.MsgRelayPayment {
	var relays []*types.RelaySession

	// Create relay request. Change session ID each call to avoid double spending error
	for i, chainID := range chainIDs {
		relaySession := &types.RelaySession{
			Provider:    provider,
			ContentHash: []byte(ts.spec.ApiCollections[0].Apis[0].Name),
			SessionId:   sessionID,
			SpecId:      chainID,
			CuSum:       relayCuSum,
			Epoch:       int64(ts.BlockHeight()),
			RelayNum:    uint64(i),
		}
		sessionID += 1

		// Sign and send the payment requests
		sig, err := sigs.Sign(clientAcc.SK, *relaySession)
		relaySession.Sig = sig
		require.Nil(ts.T, err)

		relays = append(relays, relaySession)
	}

	return types.MsgRelayPayment{Creator: provider, Relays: relays}
}

// TestDelegationLimitNotAffectingProviderReward checks that the delegation limit doesn't influence
// the provider's reward (the limit should only affect pairing)
func TestDelegationLimitNotAffectingProviderReward(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 1)                   // 1 provider, 1 client, 1 providersToPair
	ts.AddAccount(common.CONSUMER, 1, testBalance) // add delegator1
	ts.AddAccount(common.CONSUMER, 2, testBalance) // add delegator2

	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, delegator1 := ts.GetAccount(common.CONSUMER, 1)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 2)

	ts.AdvanceEpoch() // to apply pairing

	delegationAmount1 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(100))
	delegationAmount2 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(200))

	_, err := ts.TxDualstakingDelegate(delegator1, provider, ts.spec.Index, delegationAmount1)
	require.Nil(t, err)
	_, err = ts.TxDualstakingDelegate(delegator2, provider, ts.spec.Index, delegationAmount2)
	require.Nil(t, err)
	ts.AdvanceEpoch() // apply delegations

	stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
	require.True(t, found)

	// modify the stake entry to have a delegation limit higher than the total delegations
	stakeEntry.DelegateLimit = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1000))
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	res, err := ts.QueryDualstakingProviderDelegators(provider, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Delegations))

	// this is the expected reward for the provider. This should not change when changing the delegation limit
	providerReward := ts.Keepers.Dualstaking.CalcProviderReward(stakeEntry, math.NewInt(int64(relayCuSum)))
	mint := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx)
	expectedReward := mint.MulInt64(providerReward.Int64())

	balance := ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)
	newBalance := ts.GetBalance(providerAcc.Addr)

	require.Equal(t, expectedReward.TruncateInt64(), newBalance-balance)

	// modify the stake entry to have a delegation limit lower than the total delegations
	stakeEntry.DelegateLimit = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1))
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	balance = ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
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
	delegator1Acc, delegator1 := ts.GetAccount(common.CONSUMER, 1)

	ts.AdvanceEpoch() // to apply pairing

	delegationAmount1 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(100))
	_, err := ts.TxDualstakingDelegate(delegator1, provider, ts.spec.Index, delegationAmount1)
	require.Nil(t, err)
	ts.AdvanceEpoch() // apply delegations

	stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
	require.True(t, found)

	// ** provider's commission is 100% ** //

	stakeEntry.DelegateCommission = 100
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	res, err := ts.QueryDualstakingProviderDelegators(provider, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Delegations))

	// the expected reward for the provider with 100% commission is the total rewards (delegators get nothing)
	totalReward := math.NewInt(int64(relayCuSum))
	providerReward := ts.Keepers.Dualstaking.CalcProviderReward(stakeEntry, totalReward)
	require.True(t, totalReward.Equal(providerReward))

	// check that the expected reward equals to the provider's new balance minus old balance
	mint := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx)
	expectedRewardForRelay := mint.MulInt64(providerReward.Int64())

	balance := ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)
	newBalance := ts.GetBalance(providerAcc.Addr)
	require.Equal(t, expectedRewardForRelay.TruncateInt64(), newBalance-balance)

	// the delegator should get no rewards
	resRewards, err := ts.QueryDualstakingDelegatorRewards(delegator1, "", "")
	require.Nil(t, err)
	require.Equal(t, 1, len(resRewards.Rewards))
	dReward := resRewards.Rewards[0]
	require.True(t, found)
	expectedDReward := math.ZeroInt()
	require.True(t, expectedDReward.Equal(dReward.Amount.Amount))

	// claim delegator rewards and verify balance
	claimRewardsAndVerifyBalance(ts, delegator1Acc.Addr, provider, ts.spec.Index)

	// ** provider's commission is 0% ** //

	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	// the expected reward for the provider with 0% commission is no rewards at all
	balance = ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, res.Delegations)
	newBalance = ts.GetBalance(providerAcc.Addr)
	require.Equal(t, balance, newBalance)

	// the delegator should get the total rewards
	resRewards, err = ts.QueryDualstakingDelegatorRewards(delegator1, "", "")
	require.Nil(t, err)
	require.Equal(t, 1, len(resRewards.Rewards))
	dReward = resRewards.Rewards[0]
	expectedDRewardForRelay := mint.MulInt64(totalReward.Int64())
	require.Equal(t, expectedDRewardForRelay.TruncateInt64(), dReward.Amount.Amount.Int64())

	// claim delegator rewards and verify balance
	claimRewardsAndVerifyBalance(ts, delegator1Acc.Addr, provider, ts.spec.Index)
}

func claimRewardsAndVerifyBalance(ts *tester, delegator sdk.AccAddress, provider string, chainID string) {
	balance := ts.GetBalance(delegator)

	res, err := ts.QueryDualstakingDelegatorRewards(delegator.String(), provider, chainID)
	require.Nil(ts.T, err)
	reward := res.Rewards[0]

	_, err = ts.Servers.DualstakingServer.ClaimRewards(ts.Ctx, dualstakingtypes.NewMsgClaimRewards(delegator.String(), provider))
	require.Nil(ts.T, err)

	newBalance := ts.GetBalance(delegator)
	require.Equal(ts.T, balance+reward.Amount.Amount.Int64(), newBalance)

	res, err = ts.QueryDualstakingDelegatorRewards(delegator.String(), provider, chainID)
	require.Nil(ts.T, err)
	require.Equal(ts.T, 0, len(res.Rewards))
}

func TestQueryDelegatorRewards(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(3, 1, 3)                   // 3 providers, 1 client, 3 providersToPair
	ts.AddAccount(common.CONSUMER, 1, testBalance) // add delegator1
	ts.AddAccount(common.CONSUMER, 2, testBalance) // add delegator2

	provider1Acc, provider1 := ts.GetAccount(common.PROVIDER, 0) // will have stake in "mock" and "mock1"
	provider2Acc, provider2 := ts.GetAccount(common.PROVIDER, 1) // will have stake in "mock"
	_, provider3 := ts.GetAccount(common.PROVIDER, 2)            // no one delegates to it
	client1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, delegator1 := ts.GetAccount(common.CONSUMER, 1)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 2) // delegates to no one

	spec1 := common.CreateMockSpec()
	spec1.Index = "mock1"
	spec1.Name = "mock1"
	ts.AddSpec(spec1.Index, spec1)
	err := ts.StakeProvider(provider1, spec1, testStake)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	makeProviderCommissionZero(ts, ts.spec.Index, provider1Acc.Addr)
	makeProviderCommissionZero(ts, spec1.Index, provider1Acc.Addr)
	makeProviderCommissionZero(ts, ts.spec.Index, provider2Acc.Addr)

	delegationAmount := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(100))
	_, err = ts.TxDualstakingDelegate(delegator1, provider1, ts.spec.Index, delegationAmount)
	require.Nil(t, err)
	_, err = ts.TxDualstakingDelegate(delegator1, provider1, spec1.Index, delegationAmount)
	require.Nil(t, err)
	_, err = ts.TxDualstakingDelegate(delegator1, provider2, ts.spec.Index, delegationAmount)
	require.Nil(t, err)

	ts.AdvanceEpoch() // apply delegations

	expectedDelegation1 := dualstakingtypes.NewDelegation(delegator1, provider1, ts.spec.Index)
	expectedDelegation11 := dualstakingtypes.NewDelegation(delegator1, provider1, spec1.Index)
	expectedDelegations1 := []dualstakingtypes.Delegation{expectedDelegation1, expectedDelegation11}

	relayPaymentMessage := sendRelay(ts, provider1, client1Acc, []string{ts.spec.Index, spec1.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, client1Acc.Addr, provider1Acc.Addr, true, true, expectedDelegations1)

	expectedDelegation2 := dualstakingtypes.NewDelegation(delegator1, provider2, ts.spec.Index)
	expectedDelegations2 := []dualstakingtypes.Delegation{expectedDelegation2}
	relayPaymentMessage = sendRelay(ts, provider2, client1Acc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, client1Acc.Addr, provider2Acc.Addr, true, true, expectedDelegations2)

	// when the provider's commission is zero, the delegator get all the rewards from the relay (relayCu * coinPerCu)
	mintCuMultiplier := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx)
	expectedReward := mintCuMultiplier.MulInt64(int64(relayCuSum))

	// define tests
	tests := []struct {
		name            string
		delegator       string
		provider        string
		chainID         string
		expectedRewards math.LegacyDec
	}{
		{"assigned provider+chainID", delegator1, provider1, ts.spec.Index, expectedReward},
		{"assigned provider", delegator1, provider1, "", expectedReward.MulInt64(2)},
		{"nothing assigned", delegator1, "", "", expectedReward.MulInt64(3)},
		{"invalid delegator", delegator2, provider2, spec1.Index, sdk.ZeroDec()},
		{"invalid provider", delegator1, provider3, ts.spec.Index, sdk.ZeroDec()},
		{"invalid chain ID", delegator1, provider2, spec1.Index, sdk.ZeroDec()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ts.QueryDualstakingDelegatorRewards(tt.delegator, tt.provider, tt.chainID)
			require.Nil(t, err)
			delegatorReward := int64(0)
			for _, reward := range res.Rewards {
				delegatorReward += reward.Amount.Amount.Int64()
			}
			require.Equal(t, tt.expectedRewards.TruncateInt64(), delegatorReward)
		})
	}
}

func makeProviderCommissionZero(ts *tester, chainID string, provider sdk.AccAddress) {
	stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, chainID, provider)
	require.True(ts.T, found)
	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, stakeEntry.Chain, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()
}
