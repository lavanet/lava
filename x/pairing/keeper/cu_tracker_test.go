package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/stretchr/testify/require"
)

// TestAddingTrackedCuWithoutPay checks adding tracked CU works properly.
// Up until a month_of_blocks the CU should be counted for the old month.
// After that it needs to be counted for the new month. The old entry should
// reset only after month_of_blocks + blocks_to_save
func TestAddingTrackedCuWithoutPay(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 0) // 2 providers, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	_, provider2Addr := ts.GetAccount(common.PROVIDER, 1)

	_, err := ts.TxSubscriptionBuy(client1Addr, client1Addr, "free", 1, false, false) // extend by a month so the sub won't expire
	require.NoError(t, err)

	res, err := ts.QuerySubscriptionCurrent(client1Addr)
	require.NoError(t, err)
	sub := res.Sub

	// define flags for advancing time/blocks (BTS_1=BlocksToSave-EpochBlocks)
	type Advance int
	var (
		NONE  Advance = 0
		MONTH Advance = 1
		BTS_1 Advance = 2
		BTS   Advance = 3
	)

	// define tests - only after month+blocksToSave CU should stop aggregating
	tests := []struct {
		name       string
		advance    Advance // amount to advance
		live       bool
		expectedCu uint64
	}{
		{"FirstBlock", NONE, true, relayCuSum},
		{"AfterMonth", MONTH, true, relayCuSum * 2},
		{"AfterAlmostBlocksToSave", BTS_1, true, relayCuSum * 2}, // after a month has passed, there should be a new trackedCu entry so the old one doesn't change
		{"AfterBlocksToSave", BTS, false, 0},
	}

	sessionId := uint64(0)
	originalSubBlock := sub.Block
	var relayBlock uint64

	for _, tt := range tests {
		sessionId += 1
		t.Run(tt.name, func(t *testing.T) {
			switch tt.advance {
			case NONE:
				relayBlock = ts.BlockHeight()
			case MONTH:
				ts.AdvanceMonths(1)
				relayBlock = ts.BlockHeight() // note that this is before a month has passed since AdvanceMonths is actually month-5sec
				ts.AdvanceBlocks(ts.EpochBlocks() - 1)
			case BTS_1:
				ts.AdvanceBlocks(ts.BlocksToSave() - ts.EpochBlocks())
				relayBlock = ts.BlockHeight()
			case BTS:
				ts.AdvanceBlockUntilStale()
				ts.AdvanceBlocks(3) // advance enough blocks to delete the entry
			}

			// when advancing for the last time, don't send another relay to verify the original isn't found
			if tt.advance != BTS {
				relaySession := ts.newRelaySession(provider1Addr, sessionId, relayCuSum, relayBlock, 0)
				sig, err := sigs.Sign(client1Acct.SK, *relaySession)
				require.NoError(t, err)
				relaySession.Sig = sig

				relayPaymentMessage := types.MsgRelayPayment{
					Creator: provider1Addr,
					Relays:  lavaslices.Slice(relaySession),
				}

				ts.relayPaymentWithoutPay(relayPaymentMessage, true)
			}

			// check trackedCU only updated on provider 1
			cu, found, _ := ts.Keepers.Subscription.GetTrackedCu(ts.Ctx, sub.Consumer, provider1Addr, ts.spec.Index, originalSubBlock)
			if tt.live {
				require.True(t, found)
				require.Equal(t, tt.expectedCu, cu)
			} else {
				require.False(t, found)
			}

			_, found, _ = ts.Keepers.Subscription.GetTrackedCu(ts.Ctx, sub.Consumer, provider2Addr, ts.spec.Index, originalSubBlock)
			require.False(t, found)
		})
	}
}

// TestTrackedCuWithExpiredSubscription tests that the provider gets paid even if the subscription expired
func TestTrackedCuWithExpiredSubscription(t *testing.T) {
	ts := newTester(t)

	// buy subscription for 1 month
	ts.plan.PlanPolicy.MaxProvidersToPair = 1
	ts.AddPlan(ts.plan.Index, ts.plan)

	clientAcct, clientAddr := ts.AddAccount(common.CONSUMER, 0, testBalance)
	_, err := ts.TxSubscriptionBuy(clientAddr, clientAddr, ts.plan.Index, 1, false, false)
	require.NoError(t, err)

	err = ts.addProvider(1)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// payAndVerifyBalance advances a month and checks for balances (if it works, the test is ok)
	relayPayment := sendRelay(ts, providerAddr, clientAcct, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPayment, clientAcct.Addr, providerAcct.Vault.Addr, true, true, 100)

	// sanity check: verify the subscription cannot be found
	res, err := ts.QuerySubscriptionCurrent(clientAddr)
	require.NoError(t, err)
	require.Nil(t, res.Sub)
}

// TestTrackedCuWithDelegations checks that rewards are sent correctly with delegations in presence
func TestTrackedCuWithDelegations(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 client, default providers-to-pair

	clientAcct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, provider := ts.GetAccount(common.PROVIDER, 0)

	// change the provider's delegation limit and commission
	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	stakeEntry.DelegateLimit = sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake))
	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.SetStakeEntryCurrent(ts.Ctx, stakeEntry)
	ts.AdvanceEpoch()

	// delegate testStake/2 (with commission=0) -> provider should get 66% of the reward
	_, delegator := ts.AddAccount(common.CONSUMER, 1, testBalance)

	_, err := ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake/2)))
	require.NoError(t, err)
	ts.AdvanceEpoch()

	// payAndVerifyBalance advances a month and checks for balances (if it works, the test is ok)
	relayPayment := sendRelay(ts, provider, clientAcct, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPayment, clientAcct.Addr, providerAcct.Vault.Addr, true, true, 66)

	// sanity check - delegator reward should be 34% of the reward (which is the plan price)
	res, err := ts.QueryDualstakingDelegatorRewards(delegator, provider, ts.spec.Index)
	require.NoError(t, err)
	require.Equal(t, ts.plan.Price.Amount.Int64()*34/100, res.Rewards[0].Amount.AmountOf(ts.BondDenom()).Int64())
}

// TestTrackedCuWithQos checks that the tracked CU is counted properly when considering QoS
// first case: p1 is worse than p2 -> gets less reward
// second case: p1 and p2 are equally bad -> both get the same reward
func TestTrackedCuWithQos(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 2) // 2 providers, 1 client, providers-to-pair=2

	client1Acct, client := ts.GetAccount(common.CONSUMER, 0)
	provider1Acc, provider1 := ts.GetAccount(common.PROVIDER, 0)
	provider2Acc, provider2 := ts.GetAccount(common.PROVIDER, 1)

	_, err := ts.TxSubscriptionBuy(client, client, "free", 1, false, false) // extend by a month so the sub won't expire
	require.NoError(t, err)

	badQoS := &types.QualityOfServiceReport{
		Latency:      sdk.ZeroDec(),
		Availability: sdk.ZeroDec(),
		Sync:         sdk.ZeroDec(),
	}
	goodQos := &types.QualityOfServiceReport{
		Latency:      sdk.OneDec(),
		Availability: sdk.OneDec(),
		Sync:         sdk.OneDec(),
	}

	// Simulate QosWeight to be 0.5 - the default value at the time of this writing
	initQos := sdk.NewDecWithPrec(5, 1)
	initQosBytes, _ := initQos.MarshalJSON()
	initQosStr := string(initQosBytes)

	// change the QoS weight parameter to 0.5
	paramKey := string(types.KeyQoSWeight)
	paramVal := initQosStr
	err = ts.TxProposalChangeParam(types.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	planPrice := ts.plan.Price.Amount.Int64()

	// define tests
	tests := []struct {
		name             string
		bothBadQos       bool
		p1ExpectedReward int64
		p2ExpectedReward int64
	}{
		{"provider1WorstThanOther", false, planPrice * 33 / 100, planPrice * 66 / 100},
		{"bothBadQos", true, planPrice / 2, planPrice / 2},
	}

	sessionId := uint64(0)

	for _, tt := range tests {
		sessionId += 1
		t.Run(tt.name, func(t *testing.T) {
			relaySession1 := ts.newRelaySession(provider1, sessionId, relayCuSum, ts.BlockHeight(), 0)
			relaySession2 := ts.newRelaySession(provider2, sessionId, relayCuSum, ts.BlockHeight(), 0)
			if tt.bothBadQos {
				relaySession1.QosReport = badQoS
				relaySession2.QosReport = badQoS
			} else {
				relaySession1.QosReport = badQoS
				relaySession2.QosReport = goodQos
			}

			sig, err := sigs.Sign(client1Acct.SK, *relaySession1)
			require.NoError(t, err)
			relaySession1.Sig = sig
			sig, err = sigs.Sign(client1Acct.SK, *relaySession2)
			require.NoError(t, err)
			relaySession2.Sig = sig

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: provider1,
				Relays:  lavaslices.Slice(relaySession1),
			}
			ts.relayPaymentWithoutPay(relayPaymentMessage, true)

			relayPaymentMessage2 := types.MsgRelayPayment{
				Creator: provider2,
				Relays:  lavaslices.Slice(relaySession2),
			}
			ts.relayPaymentWithoutPay(relayPaymentMessage2, true)

			// advance month + blocksToSave + 1 to trigger the provider monthly payment
			ts.AdvanceMonths(1)
			ts.AdvanceEpoch()
			ts.AdvanceBlocks(ts.BlocksToSave() + 1)

			balance1 := ts.GetBalance(provider1Acc.Vault.Addr)
			balance2 := ts.GetBalance(provider2Acc.Vault.Addr)

			reward, err := ts.QueryDualstakingDelegatorRewards(provider1Acc.GetVaultAddr(), provider1Acc.Addr.String(), ts.spec.Index)
			require.Nil(ts.T, err)
			require.Len(t, reward.Rewards, 1)
			require.Equal(ts.T, tt.p1ExpectedReward, reward.Rewards[0].Amount.AmountOf(ts.BondDenom()).Int64())
			_, err = ts.TxDualstakingClaimRewards(provider1Acc.GetVaultAddr(), provider1Acc.Addr.String())
			require.Nil(ts.T, err)

			reward, err = ts.QueryDualstakingDelegatorRewards(provider2Acc.GetVaultAddr(), provider2Acc.Addr.String(), ts.spec.Index)
			require.Nil(ts.T, err)
			require.Equal(ts.T, tt.p2ExpectedReward, reward.Rewards[0].Amount.AmountOf(ts.BondDenom()).Int64())
			_, err = ts.TxDualstakingClaimRewards(provider2Acc.GetVaultAddr(), provider2Acc.Addr.String())
			require.Nil(ts.T, err)

			newBalance1 := ts.GetBalance(provider1Acc.Vault.Addr)
			newBalance2 := ts.GetBalance(provider2Acc.Vault.Addr)

			require.Equal(t, balance1+tt.p1ExpectedReward, newBalance1)
			require.Equal(t, balance2+tt.p2ExpectedReward, newBalance2)
		})
	}
}

// TestTrackedCuMultipleChains checks that if a provider provided service on two chains or
// provided double the service on one chain, it's equivalent
func TestTrackedCuMultipleChains(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 client, default providers-to-pair

	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	provider1Acc, provider1 := ts.GetAccount(common.PROVIDER, 0)
	provider2Acc, provider2 := ts.AddAccount(common.PROVIDER, 1, testBalance)

	// stake the provider on an additional chain and apply pairing (advance epoch)
	spec1 := ts.spec
	spec1Name := "spec1"
	spec1.Index = spec1Name
	spec1.Name = spec1Name
	ts.AddSpec(spec1Name, spec1)
	err := ts.StakeProvider(provider2Acc.GetVaultAddr(), provider2, spec1, testStake)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	// send two relay of relayCuSum on "spec"
	relaySession := ts.newRelaySession(provider1, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession1 := ts.newRelaySession(provider1, 1, relayCuSum, ts.BlockHeight(), 0)
	sig, err := sigs.Sign(clientAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig
	sig, err = sigs.Sign(clientAcc.SK, *relaySession1)
	require.NoError(t, err)
	relaySession1.Sig = sig

	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider1,
		Relays:  lavaslices.Slice(relaySession, relaySession1),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// send one relay of 2*relayCuSum on spec1Name
	relaySession2 := ts.newRelaySession(provider2, 0, 2*relayCuSum, ts.BlockHeight(), 0)
	relaySession2.SpecId = spec1Name
	sig, err = sigs.Sign(clientAcc.SK, *relaySession2)
	require.NoError(t, err)
	relaySession2.Sig = sig

	relayPaymentMessage2 := types.MsgRelayPayment{
		Creator: provider2,
		Relays:  lavaslices.Slice(relaySession2),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage2, true)

	// advance month + blocksToSave + 1 to trigger the provider monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	balance1 := ts.GetBalance(provider1Acc.Addr)
	balance2 := ts.GetBalance(provider2Acc.Addr)

	require.Equal(t, balance1, balance2)
}

// TestTrackedCuPlanPriceChange checks that the reward is calculated with the original plan price of
// the sub when bought
func TestTrackedCuPlanPriceChange(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 2)

	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)

	originalPlanPrice := ts.plan.Price.Amount.Int64()

	newPlan := ts.plan
	newPlan.Price.Amount = ts.plan.Price.Amount.MulRaw(2)
	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []planstypes.Plan{newPlan}, false)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	relayPayment := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.relayPaymentWithoutPay(relayPayment, true)

	balanceBeforePay := ts.GetBalance(providerAcc.Vault.Addr)

	// advance month + blocksToSave + 1 to trigger the provider monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	reward, err := ts.QueryDualstakingDelegatorRewards(providerAcc.GetVaultAddr(), providerAcc.Addr.String(), ts.spec.Index)
	require.Nil(ts.T, err)
	require.Equal(ts.T, originalPlanPrice, reward.Rewards[0].Amount.AmountOf(ts.BondDenom()).Int64())
	_, err = ts.TxDualstakingClaimRewards(providerAcc.GetVaultAddr(), providerAcc.Addr.String())
	require.Nil(ts.T, err)

	balance := ts.GetBalance(providerAcc.Vault.Addr)
	require.Equal(t, balanceBeforePay+originalPlanPrice, balance)
}

// TestVaultProviderTrackedCu tests that a relay payment sent by the provider get its CU
// tracked for the vault address
// Scenarios:
//  1. normal tracked CU pipeline -> tracked CU should be registered to the vault address
//  2. after that, advance a month + unbonding time and verify that there are claimable rewards of the vault
//     and not provider
//  3. provider tries to claim rewards -> fails
//  4. claim rewards with vault -> success
//  5. vault address tries to send relay payment -> should fail
func TestVaultProviderTrackedCu(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 client, default providers-to-pair

	providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)
	provider := providerAcc.Addr.String()
	vault := providerAcc.GetVaultAddr()

	// send relay payment with provider to have tracked CU
	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	relaySession := ts.newRelaySession(provider, 0, relayCuSum, ts.BlockHeight(), 0)
	sig, err := sigs.Sign(clientAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider,
		Relays:  lavaslices.Slice(relaySession),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// send relay payment with vault - should not be paid for
	relaySession = ts.newRelaySession(vault, 0, relayCuSum, ts.BlockHeight(), 0)
	sig, err = sigs.Sign(clientAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig
	relayPaymentMessage = types.MsgRelayPayment{
		Creator: vault,
		Relays:  lavaslices.Slice(relaySession),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, false)

	// verify tracked CU - vault should have none and provider should have some rewards
	res, err := ts.QueryPairingProviderMonthlyPayout(provider)
	require.NoError(t, err)
	require.Len(t, res.Details, 1)
	require.NotEqual(t, uint64(0), res.Total)

	res, err = ts.QueryPairingProviderMonthlyPayout(vault)
	require.NoError(t, err)
	require.Len(t, res.Details, 0)
	require.Equal(t, uint64(0), res.Total)

	// advance month + blocksToSave + 1 to trigger the provider monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	// claim rewards (provider should have none, vault should have some)
	tests := []struct {
		name     string
		creator  string
		provider string
		rewarded bool // should the creator get reward
	}{
		{"provider creator (bad)", provider, provider, false},
		{"vault provider (bad)", provider, vault, false},
		{"vault creator happy flow", vault, provider, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creatorAcc, err := sdk.AccAddressFromBech32(tt.creator)
			require.NoError(t, err)

			res, err := ts.QueryDualstakingDelegatorRewards(tt.creator, tt.provider, ts.spec.Index)
			require.NoError(t, err)
			require.Equal(t, tt.rewarded, len(res.Rewards) > 0)

			before := ts.GetBalance(creatorAcc)
			_, err = ts.TxDualstakingClaimRewards(tt.creator, tt.provider)
			require.NoError(t, err)
			after := ts.GetBalance(creatorAcc)
			require.Equal(t, tt.rewarded, after > before)
		})
	}
}

// TestProviderMonthlyPayoutQuery tests the monthly-payout query
// Scenario: the provider provided service on two chains and in one of them he has a delegator
func TestProviderMonthlyPayoutQuery(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 client, default providers-to-pair

	clientAcc, client := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, provider := ts.GetAccount(common.PROVIDER, 0)

	_, err := ts.TxSubscriptionBuy(client, client, "free", 1, false, false) // extend by a month so the sub won't expire
	require.NoError(t, err)

	// stake the provider on an additional chain and apply pairing (advance epoch)
	spec1 := ts.spec
	spec1Name := "spec1"
	spec1.Index = spec1Name
	spec1.Name = spec1Name
	ts.AddSpec(spec1Name, spec1)
	err = ts.StakeProvider(providerAcct.GetVaultAddr(), provider, spec1, testStake)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	// change the provider's delegation limit and commission
	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	stakeEntry.DelegateLimit = sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake))
	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.SetStakeEntryCurrent(ts.Ctx, stakeEntry)
	ts.AdvanceEpoch()

	// delegate testStake/2 (with commission=0) -> provider should get 66% of the reward
	_, delegator := ts.AddAccount(common.CONSUMER, 1, testBalance)
	_, err = ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake/2)))
	require.NoError(t, err)
	ts.AdvanceEpoch()
	ts.AdvanceMonths(1).AdvanceEpoch() // advance first month of delegation so it'll apply
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	// send two relay payments in spec and spec1
	relaySession := ts.newRelaySession(provider, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession2 := ts.newRelaySession(provider, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession2.SpecId = spec1Name
	sig, err := sigs.Sign(clientAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig
	sig, err = sigs.Sign(clientAcc.SK, *relaySession2)
	require.NoError(t, err)
	relaySession2.Sig = sig
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider,
		Relays:  lavaslices.Slice(relaySession, relaySession2),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// check for expected balance: credit*100/200 (from spec1) + credit*(100/200)*(2/3) (from spec, considering delegations)
	// for credit=100 (first month there was no use, so no credit was spent), expected monthly payout is 50+33
	expectedTotalPayout := uint64(83)
	expectedPayouts := []types.SubscriptionPayout{
		{Subscription: clientAcc.Addr.String(), ChainId: ts.spec.Index, Amount: 33},
		{Subscription: clientAcc.Addr.String(), ChainId: spec1.Index, Amount: 50},
	}
	res, err := ts.QueryPairingProviderMonthlyPayout(provider)
	require.NoError(t, err)
	require.Equal(t, expectedTotalPayout, res.Total)
	details := []types.SubscriptionPayout{}
	for _, p := range res.Details {
		details = append(details, *p)
	}
	require.True(t, lavaslices.UnorderedEqual(expectedPayouts, details))

	// check the expected subscrription payout
	subRes, err := ts.QueryPairingSubscriptionMonthlyPayout(client)
	require.NoError(t, err)
	require.Equal(t, uint64(100), subRes.Total) // total reward = credit
	expectedSubPayouts := []types.ChainIDPayout{
		{
			ChainId: ts.spec.Index, Payouts: []*types.ProviderPayout{
				{Provider: provider, Amount: 50},
			},
		},
		{
			ChainId: ts.spec.Index, Payouts: []*types.ProviderPayout{
				{Provider: provider, Amount: 50},
			},
		},
	}
	require.Equal(t, 2, len(subRes.Details))
	for _, payout := range subRes.Details {
		if payout.ChainId == expectedSubPayouts[0].ChainId || payout.ChainId == expectedSubPayouts[1].ChainId {
			if payout.Payouts[0].Provider == expectedSubPayouts[0].Payouts[0].Provider && payout.Payouts[0].Amount == expectedSubPayouts[0].Payouts[0].Amount {
				break
			}
		}
		require.FailNow(t, "sub expected payout don't match with query output")
	}

	// advance month + blocksToSave + 1 to trigger the monthly payment
	oldBalance := ts.GetBalance(providerAcct.Vault.Addr)

	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	resq, err := ts.QueryDualstakingDelegatorRewards(providerAcct.GetVaultAddr(), provider, "")
	require.NoError(t, err)
	fmt.Printf("resq.Rewards: %v\n", resq.Rewards)

	_, err = ts.TxDualstakingClaimRewards(providerAcct.GetVaultAddr(), providerAcct.Addr.String())
	require.Nil(ts.T, err)

	// another month has passed, so the expected payout is doubled
	balance := ts.GetBalance(providerAcct.Vault.Addr)
	require.Equal(t, expectedTotalPayout*2, uint64(balance-oldBalance))

	// verify that the monthly payout query return 0 after the payment was transferred to the provider
	res, err = ts.QueryPairingProviderMonthlyPayout(provider)
	require.NoError(t, err)
	require.Equal(t, uint64(0), res.Total)
	require.Nil(t, res.Details)
}

func TestProviderMonthlyPayoutQueryWithContributor(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 client, default providers-to-pair

	clientAcc, client := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, provider := ts.GetAccount(common.PROVIDER, 0)
	// stake the provider on an additional chain and apply pairing (advance epoch)
	spec1 := ts.spec
	spec1Name := "spec1"
	spec1.Index = spec1Name
	spec1.Name = spec1Name
	contributorAccount, contributorAddress := ts.AddAccount("contributor", 0, 0)
	contributorAccount2, contributorAddress2 := ts.AddAccount("contributor2", 0, 0)
	spec1.Contributor = []string{contributorAddress, contributorAddress2}
	percentage := math.LegacyNewDecWithPrec(5, 1) // half the rewards
	spec1.ContributorPercentage = &percentage
	ts.AddSpec(spec1Name, spec1)
	err := ts.StakeProvider(providerAcct.GetVaultAddr(), provider, spec1, testStake)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	// change the provider's delegation limit and commission
	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	stakeEntry.DelegateLimit = sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake))
	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.SetStakeEntryCurrent(ts.Ctx, stakeEntry)
	ts.AdvanceEpoch()

	// delegate testStake/2 (with commission=0) -> provider should get 66% of the reward
	_, delegator := ts.AddAccount(common.CONSUMER, 1, testBalance)
	_, err = ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake/2)))
	require.NoError(t, err)
	delegationTime := ts.BlockTime()
	ts.AdvanceEpoch()

	// delegations are calculated in the reward process only if a month had passed since they were
	// delegated. If we advanced a month, we would trigger the payment code (which is not desired
	// now since we want to check the expected reward, before it gets transferred). So, we need to artificially
	// change the delegations' timestamp to be a month forward
	fakeTimestamp := ts.BlockTime().AddDate(0, -2, 0)
	err = ts.ChangeDelegationTimestamp(provider, delegator, ts.spec.Index, ts.BlockHeight(), ts.GetNextMonth(fakeTimestamp))
	require.NoError(t, err)

	// send two relay payments in spec and spec1
	relaySession := ts.newRelaySession(provider, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession2 := ts.newRelaySession(provider, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession2.SpecId = spec1Name
	sig, err := sigs.Sign(clientAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig
	sig, err = sigs.Sign(clientAcc.SK, *relaySession2)
	require.NoError(t, err)
	relaySession2.Sig = sig
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider,
		Relays:  lavaslices.Slice(relaySession, relaySession2),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// check for expected balance: planPrice*100/200 (from spec1) + planPrice*(100/200)*(2/3) (from spec, considering delegations)
	// for planPrice=100, expected monthly payout is 50 (spec1 with contributor) + 33 (normal spec no contributor)
	expectedContributorPay := uint64(12) // half the plan payment for spec1:25 then divided between contributors half half rounded down
	expectedTotalPayout := uint64(83) - expectedContributorPay*2
	expectedPayouts := []types.SubscriptionPayout{
		{Subscription: clientAcc.Addr.String(), ChainId: ts.spec.Index, Amount: 33},
		{Subscription: clientAcc.Addr.String(), ChainId: spec1.Index, Amount: 26}, // 50 - 26 for contributors (each contributor gets 12)
	}
	res, err := ts.QueryPairingProviderMonthlyPayout(provider)
	require.NoError(t, err)
	require.Equal(t, expectedTotalPayout, res.Total)
	details := []types.SubscriptionPayout{}
	for _, p := range res.Details {
		details = append(details, *p)
	}
	require.True(t, lavaslices.UnorderedEqual(expectedPayouts, details))

	// check the expected subscrription payout
	subRes, err := ts.QueryPairingSubscriptionMonthlyPayout(client)
	require.NoError(t, err)
	require.Equal(t, uint64(100), subRes.Total) // total reward = plan price
	expectedSubPayouts := []types.ChainIDPayout{
		{
			ChainId: ts.spec.Index, Payouts: []*types.ProviderPayout{
				{Provider: provider, Amount: 50},
			},
		},
		{
			ChainId: ts.spec.Index, Payouts: []*types.ProviderPayout{
				{Provider: provider, Amount: 50},
			},
		},
	}
	require.Equal(t, 2, len(subRes.Details))
	for _, payout := range subRes.Details {
		if payout.ChainId == expectedSubPayouts[0].ChainId || payout.ChainId == expectedSubPayouts[1].ChainId {
			if payout.Payouts[0].Provider == expectedSubPayouts[0].Payouts[0].Provider && payout.Payouts[0].Amount == expectedSubPayouts[0].Payouts[0].Amount {
				break
			}
		}
		require.FailNow(t, "sub expected payout don't match with query output")
	}

	// advance month + blocksToSave + 1 to trigger the monthly payment
	// (also restore delegation original timestamp)
	oldBalance := ts.GetBalance(providerAcct.Vault.Addr)
	err = ts.ChangeDelegationTimestamp(provider, delegator, ts.spec.Index, ts.BlockHeight(), ts.GetNextMonth(delegationTime))
	require.NoError(t, err)

	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	_, err = ts.TxDualstakingClaimRewards(providerAcct.GetVaultAddr(), providerAcct.Addr.String())
	require.Nil(ts.T, err)

	balance := ts.GetBalance(providerAcct.Vault.Addr)
	require.Equal(t, expectedTotalPayout, uint64(balance-oldBalance))

	balance = ts.GetBalance(contributorAccount.Addr)
	require.Equal(t, expectedContributorPay, uint64(balance))

	balance = ts.GetBalance(contributorAccount2.Addr)
	require.Equal(t, expectedContributorPay, uint64(balance))

	// verify that the monthly payout query return 0 after the payment was transferred to the provider
	res, err = ts.QueryPairingProviderMonthlyPayout(provider)
	require.NoError(t, err)
	require.Equal(t, uint64(0), res.Total)
	require.Nil(t, res.Details)
}

// TestFrozenProviderGetReward checks that frozen providers still get rewards.
// providers can freeze themselves or be forcibly frozen due to unresponsiveness
func TestFrozenProviderGetReward(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 2)

	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)

	relayPayment := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.relayPaymentWithoutPay(relayPayment, true)

	balanceBeforePay := ts.GetBalance(providerAcc.Vault.Addr)

	err := ts.Keepers.Pairing.FreezeProvider(ts.Ctx, provider, []string{ts.spec.Index}, "unresponsiveness")
	require.NoError(t, err)

	// advance month + blocksToSave + 1 to trigger the provider monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	planPrice := ts.plan.Price.Amount.Int64()
	reward, err := ts.QueryDualstakingDelegatorRewards(providerAcc.GetVaultAddr(), providerAcc.Addr.String(), ts.spec.Index)
	require.Nil(ts.T, err)
	require.Equal(ts.T, planPrice, reward.Rewards[0].Amount.AmountOf(ts.BondDenom()).Int64())
	_, err = ts.TxDualstakingClaimRewards(providerAcc.GetVaultAddr(), providerAcc.Addr.String())
	require.Nil(ts.T, err)

	balance := ts.GetBalance(providerAcc.Vault.Addr)
	require.Equal(t, balanceBeforePay+planPrice, balance)
}

// TestTrackedCuDeletion makes sure that if there are two trackedCU entries
// both with some CU, when the deletion occurs it deletes the older one
func TestTrackedCuDeletion(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 2)

	clientAcc, client := ts.GetAccount(common.CONSUMER, 0)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	_, err := ts.TxSubscriptionBuy(client, client, "free", 1, false, false) // extend by a month so the sub won't expire
	require.NoError(t, err)

	// send relay to track CU
	relayPayment := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.relayPaymentWithoutPay(relayPayment, true)
	oldMonthBlock := relayPayment.Relays[0].Epoch

	// send relay after a month (opens a new trackedCU entry)
	ts.AdvanceMonths(1).AdvanceEpoch()
	relayPayment = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.relayPaymentWithoutPay(relayPayment, true)

	// send another relay but to the old trackedCU entry
	relayPayment = sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	relayPayment.Relays[0].Epoch = oldMonthBlock
	sig, err := sigs.Sign(clientAcc.SK, *relayPayment.Relays[0])
	relayPayment.Relays[0].Sig = sig
	require.Nil(ts.T, err)
	ts.relayPaymentWithoutPay(relayPayment, true)

	// advance blocksToSave to trigger a reset for the old CU tracker
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	// advance blocks until the old trackedCU entry stale period is over -> should be deleted
	ts.AdvanceBlockUntilStale()

	// try to find the old tracked CU (should not succeed)
	// also verify that the existent trackedCU has relayCuSum and not 2*relayCuSum (as the old one has)
	_, found, _ := ts.Keepers.Subscription.GetTrackedCu(ts.Ctx, client, provider, ts.spec.Index, uint64(oldMonthBlock))
	require.False(t, found)

	sub, found := ts.Keepers.Subscription.GetSubscription(ts.Ctx, client)
	require.True(t, found)

	cu, found, _ := ts.Keepers.Subscription.GetTrackedCu(ts.Ctx, client, provider, ts.spec.Index, sub.Block)
	require.True(t, found)
	require.Equal(t, relayCuSum, cu)
}
