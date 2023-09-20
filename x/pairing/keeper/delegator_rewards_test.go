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
	// define tests - all the combinations of below:
	//   commission = {0, 47, 100}
	//   limit = {0, <totalDelegations, >totalDelegations}
	//   delegation_ratio = {0/100, 50/50, 14/86}
	tests := []struct {
		name           string
		commission     uint64
		limit          int64
		d1Amount       int64  // delegator1 delegation percentage (of provider stake(=total delegations))
		d2Amount       int64  // delegator2 delegation percentage (of provider stake(=total delegations))
		providerReward uint64 // expected provider's part of the total reward
		d1Reward       int64  // expected delegator1's part of the total reward
		d2Reward       int64  // expected delegator2's part of the total reward
	}{
		{"Test #1", 0, 0, 0, 100, 100, 0, 0},
		{"Test #2", 0, 0, 50, 50, 100, 0, 0},
		{"Test #3", 0, 0, 14, 86, 100, 0, 0},
		{"Test #4", 0, 50, 0, 100, 67, 0, 33},
		{"Test #5", 0, 50, 50, 50, 66, 17, 17},
		{"Test #6", 0, 50, 14, 86, 67, 4, 29},
		{"Test #7", 0, 150, 0, 100, 50, 0, 50},
		{"Test #8", 0, 150, 50, 50, 50, 25, 25},
		{"Test #9", 0, 150, 14, 86, 50, 7, 43},
		{"Test #10", 47, 0, 0, 100, 100, 0, 0},
		{"Test #11", 47, 0, 50, 50, 100, 0, 0},
		{"Test #12", 47, 0, 14, 86, 100, 0, 0},
		{"Test #13", 47, 50, 0, 100, 82, 0, 18},
		{"Test #14", 47, 50, 50, 50, 82, 9, 9},
		{"Test #15", 47, 50, 14, 86, 82, 2, 16},
		{"Test #16", 47, 150, 0, 100, 73, 0, 27},
		{"Test #17", 47, 150, 50, 50, 74, 13, 13},
		{"Test #18", 47, 150, 14, 86, 74, 3, 23},
		{"Test #19", 100, 0, 0, 100, 100, 0, 0},
		{"Test #20", 100, 0, 50, 50, 100, 0, 0},
		{"Test #21", 100, 0, 14, 86, 100, 0, 0},
		{"Test #22", 100, 10, 0, 100, 100, 0, 0},
		{"Test #23", 100, 10, 50, 50, 100, 0, 0},
		{"Test #24", 100, 10, 14, 86, 100, 0, 0},
		{"Test #25", 100, 150, 0, 100, 100, 0, 0},
		{"Test #26", 100, 150, 50, 50, 100, 0, 0},
		{"Test #27", 100, 150, 14, 86, 100, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			ts.setupForPayments(1, 1, 1)                   // 1 provider, 1 client, 1 providersToPair
			ts.AddAccount(common.CONSUMER, 1, testBalance) // add delegator1
			ts.AddAccount(common.CONSUMER, 2, testBalance) // add delegator2
			err := ts.TxProposalChangeParam(types.ModuleName, string(types.KeyMintCoinsPerCU), "\"1\"")
			require.Nil(t, err)

			providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
			clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
			_, delegator1 := ts.GetAccount(common.CONSUMER, 1)
			_, delegator2 := ts.GetAccount(common.CONSUMER, 2)

			ts.AdvanceEpoch() // to apply pairing

			// ** check provider reward without delegators ** //

			relayPaymentMessage := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
			ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, 100)

			// ** check provider reward with delegators ** //

			delegationAmount := testStake
			// delegate
			amount1 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(tt.d1Amount*delegationAmount/100))
			if amount1.IsZero() {
				amount1.Amount = sdk.OneInt()
			}
			_, err = ts.TxDualstakingDelegate(delegator1, provider, ts.spec.Index, amount1)
			require.Nil(t, err)

			amount2 := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(tt.d2Amount*delegationAmount/100))
			_, err = ts.TxDualstakingDelegate(delegator2, provider, ts.spec.Index, amount2)
			require.Nil(t, err)
			ts.AdvanceEpoch() // apply delegations

			// change delegation traits of stake entry and get the modified one
			stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
			require.True(t, found)
			stakeEntry.DelegateLimit = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(tt.limit*testStake/100))
			stakeEntry.DelegateCommission = tt.commission
			ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
			ts.AdvanceEpoch()
			stakeEntry, found, _ = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcc.Addr)
			require.True(t, found)

			// check that there are two delegators
			res, err := ts.QueryDualstakingProviderDelegators(provider, false)
			require.Nil(t, err)
			require.Equal(t, 2, len(res.Delegations))

			// calc useful consts
			totalReward := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx).MulInt64(int64(relayCuSum))

			// send relay and check provider balance according to expected providerRewardPerc (done inside payAndVerifyBalance)
			relayPaymentMessage = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
			ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, tt.providerReward)

			// Get the delegator rewards from the delegatorRewardsMap
			resRewards1, err := ts.QueryDualstakingDelegatorRewards(delegator1, provider, stakeEntry.Chain)
			require.Nil(t, err)
			resRewards2, err := ts.QueryDualstakingDelegatorRewards(delegator2, provider, stakeEntry.Chain)
			require.Nil(t, err)
			d1Reward := resRewards1.Rewards[0].Amount.Amount.Int64()
			d2Reward := resRewards2.Rewards[0].Amount.Amount.Int64()

			require.Equal(t, tt.d1Reward, d1Reward)
			require.Equal(t, tt.d2Reward, d2Reward)

			// check the delegators rewards sum up to the expected total
			totalDelegatorsReward := totalReward.TruncateInt64() * (100 - int64(tt.providerReward)) / 100
			require.Equal(t, totalDelegatorsReward, d1Reward+d2Reward)

			// claim delegator rewards and verify balance
			delegator1Addr, err := sdk.AccAddressFromBech32(delegator1)
			require.Nil(t, err)
			delegator2Addr, err := sdk.AccAddressFromBech32(delegator2)
			require.Nil(t, err)
			claimRewardsAndVerifyBalance(ts, delegator1Addr, provider, stakeEntry.Chain)
			claimRewardsAndVerifyBalance(ts, delegator2Addr, provider, stakeEntry.Chain)
		})
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
	relayPaymentMessage := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, 70)
	newBalance := ts.GetBalance(providerAcc.Addr)

	require.Equal(t, expectedReward.TruncateInt64(), newBalance-balance)

	// modify the stake entry to have a delegation limit lower than the total delegations
	stakeEntry.DelegateLimit = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(testStake))
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	providerReward, _ = ts.Keepers.Dualstaking.CalcRewards(stakeEntry, math.NewInt(int64(relayCuSum)))
	expectedReward = mint.MulInt64(providerReward.Int64())

	balance = ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, 70)
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
	relayPaymentMessage := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, 100)
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

	// the expected reward for the provider with 0% commission is half of the total rewards
	// (in this test specifically, effectiveDelegations = delegateTotal = providerStake)
	balance = ts.GetBalance(providerAcc.Addr)
	relayPaymentMessage = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, clientAcc.Addr, providerAcc.Addr, true, true, 50)
	newBalance = ts.GetBalance(providerAcc.Addr)
	require.Equal(t, expectedRewardForRelay.TruncateInt64()/2, newBalance-balance)

	// the delegator should get the total rewards
	resRewards, err = ts.QueryDualstakingDelegatorRewards(delegator1, "", "")
	require.Nil(t, err)
	require.Equal(t, 1, len(resRewards.Rewards))
	dReward = resRewards.Rewards[0]
	expectedDRewardForRelay := mint.MulInt64(totalReward.Int64())
	require.Equal(t, expectedDRewardForRelay.TruncateInt64()/2, dReward.Amount.Amount.Int64())

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

	delegationAmount := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(testStake))
	_, err = ts.TxDualstakingDelegate(delegator1, provider1, ts.spec.Index, delegationAmount)
	require.Nil(t, err)
	_, err = ts.TxDualstakingDelegate(delegator1, provider1, spec1.Index, delegationAmount)
	require.Nil(t, err)
	_, err = ts.TxDualstakingDelegate(delegator1, provider2, ts.spec.Index, delegationAmount)
	require.Nil(t, err)

	ts.AdvanceEpoch() // apply delegations

	relayPaymentMessage := sendRelay(ts, provider1, client1Acc, []string{ts.spec.Index, spec1.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, client1Acc.Addr, provider1Acc.Addr, true, true, 50)

	relayPaymentMessage = sendRelay(ts, provider2, client1Acc, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPaymentMessage, client1Acc.Addr, provider2Acc.Addr, true, true, 50)

	mintCuMultiplier := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx)
	expectedReward := mintCuMultiplier.MulInt64(int64(relayCuSum / 2))

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
	stakeEntry.DelegateLimit = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(testStake))
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, stakeEntry.Chain, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()
}
