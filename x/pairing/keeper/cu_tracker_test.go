package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/utils/slices"
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

	res, err := ts.QuerySubscriptionCurrent(client1Addr)
	require.Nil(t, err)
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
				require.Nil(t, err)
				relaySession.Sig = sig

				relayPaymentMessage := types.MsgRelayPayment{
					Creator: provider1Addr,
					Relays:  slices.Slice(relaySession),
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
	_, err := ts.TxSubscriptionBuy(clientAddr, clientAddr, ts.plan.Index, 1)
	require.Nil(t, err)

	err = ts.addProvider(1)
	require.Nil(t, err)
	ts.AdvanceEpoch()

	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// payAndVerifyBalance advances a month and checks for balances (if it works, the test is ok)
	relayPayment := sendRelay(ts, providerAddr, clientAcct, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPayment, clientAcct.Addr, providerAcct.Addr, true, true, 100)

	// sanity check: verify the subscription cannot be found
	res, err := ts.QuerySubscriptionCurrent(clientAddr)
	require.Nil(t, err)
	require.Nil(t, res.Sub)
}

// TestTrackedCuWithDelegations checks that rewards are sent correctly with delegations in presence
func TestTrackedCuWithDelegations(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 client, default providers-to-pair

	clientAcct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, provider := ts.GetAccount(common.PROVIDER, 0)

	// change the provider's delegation limit and commission
	stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcct.Addr)
	require.True(t, found)
	stakeEntry.DelegateLimit = sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake))
	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	// delegate testStake/2 (with commission=0) -> provider should get 66% of the reward
	_, delegator := ts.AddAccount(common.CONSUMER, 1, testBalance)

	_, err := ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake/2)))
	require.Nil(t, err)
	ts.AdvanceEpoch()

	// payAndVerifyBalance advances a month and checks for balances (if it works, the test is ok)
	relayPayment := sendRelay(ts, provider, clientAcct, []string{ts.spec.Index})
	ts.payAndVerifyBalance(relayPayment, clientAcct.Addr, providerAcct.Addr, true, true, 66)

	// sanity check - delegator reward should be 34% of the reward (which is the plan price)
	res, err := ts.QueryDualstakingDelegatorRewards(delegator, provider, ts.spec.Index)
	require.Nil(t, err)
	require.Equal(t, ts.plan.Price.Amount.Int64()*34/100, res.Rewards[0].Amount.Amount.Int64())
}

// TestTrackedCuWithQos checks that the tracked CU is counted properly when considering QoS
// first case: p1 is worse than p2 -> gets less reward
// second case: p1 and p2 are equally bad -> both get the same reward
func TestTrackedCuWithQos(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 2) // 2 providers, 1 client, providers-to-pair=2

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	provider1Acc, provider1 := ts.GetAccount(common.PROVIDER, 0)
	provider2Acc, provider2 := ts.GetAccount(common.PROVIDER, 1)

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
	err := ts.TxProposalChangeParam(types.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

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
			require.Nil(t, err)
			relaySession1.Sig = sig
			sig, err = sigs.Sign(client1Acct.SK, *relaySession2)
			require.Nil(t, err)
			relaySession2.Sig = sig

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: provider1,
				Relays:  slices.Slice(relaySession1),
			}
			ts.relayPaymentWithoutPay(relayPaymentMessage, true)

			relayPaymentMessage2 := types.MsgRelayPayment{
				Creator: provider2,
				Relays:  slices.Slice(relaySession2),
			}
			ts.relayPaymentWithoutPay(relayPaymentMessage2, true)

			balance1 := ts.GetBalance(provider1Acc.Addr)
			balance2 := ts.GetBalance(provider2Acc.Addr)

			// advance month + blocksToSave + 1 to trigger the monthly payment
			ts.AdvanceMonths(1)
			ts.AdvanceBlocks(ts.BlocksToSave() + 1)

			newBalance1 := ts.GetBalance(provider1Acc.Addr)
			newBalance2 := ts.GetBalance(provider2Acc.Addr)

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
	err := ts.StakeProvider(provider2, spec1, testStake)
	require.Nil(t, err)
	ts.AdvanceEpoch()

	// send two relay of relayCuSum on "spec"
	relaySession := ts.newRelaySession(provider1, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession1 := ts.newRelaySession(provider1, 1, relayCuSum, ts.BlockHeight(), 0)
	sig, err := sigs.Sign(clientAcc.SK, *relaySession)
	require.Nil(t, err)
	relaySession.Sig = sig
	sig, err = sigs.Sign(clientAcc.SK, *relaySession1)
	require.Nil(t, err)
	relaySession1.Sig = sig

	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider1,
		Relays:  slices.Slice(relaySession, relaySession1),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// send one relay of 2*relayCuSum on spec1Name
	relaySession2 := ts.newRelaySession(provider2, 0, 2*relayCuSum, ts.BlockHeight(), 0)
	relaySession2.SpecId = spec1Name
	sig, err = sigs.Sign(clientAcc.SK, *relaySession2)
	require.Nil(t, err)
	relaySession2.Sig = sig

	relayPaymentMessage2 := types.MsgRelayPayment{
		Creator: provider2,
		Relays:  slices.Slice(relaySession2),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage2, true)

	// advance month + blocksToSave + 1 to trigger the monthly payment
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
	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []planstypes.Plan{newPlan})
	require.Nil(t, err)
	ts.AdvanceEpoch()

	relayPayment := sendRelay(ts, provider, clientAcc, []string{ts.spec.Index})
	ts.relayPaymentWithoutPay(relayPayment, true)

	balanceBeforePay := ts.GetBalance(providerAcc.Addr)

	// advance month + blocksToSave + 1 to trigger the monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	balance := ts.GetBalance(providerAcc.Addr)
	require.Equal(t, balanceBeforePay+originalPlanPrice, balance)
}

// TestMonthlyPayoutQuery tests the monthly-payout query
// Scenario: the provider provided service on two chains and in one of them he has a delegator
func TestMonthlyPayoutQuery(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 client, default providers-to-pair

	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, provider := ts.GetAccount(common.PROVIDER, 0)
	// stake the provider on an additional chain and apply pairing (advance epoch)
	spec1 := ts.spec
	spec1Name := "spec1"
	spec1.Index = spec1Name
	spec1.Name = spec1Name
	ts.AddSpec(spec1Name, spec1)
	err := ts.StakeProvider(provider, spec1, testStake)
	require.Nil(t, err)
	ts.AdvanceEpoch()

	// change the provider's delegation limit and commission
	stakeEntry, found, stakeEntryIndex := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, providerAcct.Addr)
	require.True(t, found)
	stakeEntry.DelegateLimit = sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake))
	stakeEntry.DelegateCommission = 0
	ts.Keepers.Epochstorage.ModifyStakeEntryCurrent(ts.Ctx, ts.spec.Index, stakeEntry, stakeEntryIndex)
	ts.AdvanceEpoch()

	// delegate testStake/2 (with commission=0) -> provider should get 66% of the reward
	_, delegator := ts.AddAccount(common.CONSUMER, 1, testBalance)
	_, err = ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(testStake/2)))
	require.Nil(t, err)
	ts.AdvanceEpoch()

	// send two relay payments in spec and spec1
	relaySession := ts.newRelaySession(provider, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession2 := ts.newRelaySession(provider, 0, relayCuSum, ts.BlockHeight(), 0)
	relaySession2.SpecId = spec1Name
	sig, err := sigs.Sign(clientAcc.SK, *relaySession)
	require.Nil(t, err)
	relaySession.Sig = sig
	sig, err = sigs.Sign(clientAcc.SK, *relaySession2)
	require.Nil(t, err)
	relaySession2.Sig = sig
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider,
		Relays:  slices.Slice(relaySession, relaySession2),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// check for expected balance: planPrice*100/200 (from spec1) + planPrice*(100/200)*(2/3) (from spec, considering delegations)
	// for planPrice=100, expected monthly payout is 50+33
	expectedPayout := uint64(83)
	res, err := ts.QueryPairingMonthlyPayout(provider)
	require.Nil(t, err)
	require.Equal(t, expectedPayout, res.Amount)

	// advance month + blocksToSave + 1 to trigger the monthly payment
	oldBalance := ts.GetBalance(providerAcct.Addr)

	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	balance := ts.GetBalance(providerAcct.Addr)
	require.Equal(t, expectedPayout, uint64(balance-oldBalance))

	// verify that the monthly payout query return 0 after the payment was transferred to the provider
	res, err = ts.QueryPairingMonthlyPayout(provider)
	require.Nil(t, err)
	require.Equal(t, uint64(0), res.Amount)
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

	balanceBeforePay := ts.GetBalance(providerAcc.Addr)

	err := ts.Keepers.Pairing.FreezeProvider(ts.Ctx, provider, []string{ts.spec.Index}, "unresponsiveness")
	require.Nil(t, err)

	// advance month + blocksToSave + 1 to trigger the monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	balance := ts.GetBalance(providerAcc.Addr)
	planPrice := ts.plan.Price.Amount.Int64()
	require.Equal(t, balanceBeforePay+planPrice, balance)
}

// TestTrackedCuDeletion makes sure that if there are two trackedCU entries
// both with some CU, when the deletion occurs it deletes the older one
func TestTrackedCuDeletion(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 2)

	clientAcc, client := ts.GetAccount(common.CONSUMER, 0)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

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
