package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/x/rewards/types"
	subscription "github.com/lavanet/lava/x/subscription/keeper"
	"github.com/stretchr/testify/require"
)

// the rewards here zero
func TestZeroProvidersRewards(t *testing.T) {
	ts := newTester(t)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err := ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	consumerAAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAAcc.Addr.String(), consumerAAcc.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 0)

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 0)
}

// the rewards here is maxboost*totalbaserewards, in this test the rewards for the providers are low (first third of the graph)
func TestBasicBoostProvidersRewards(t *testing.T) {
	ts := newTester(t)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err := ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	consumerAAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAAcc.Addr.String(), consumerAAcc.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	baserewards := uint64(100)
	// the rewards by the subscription will be limited by LIMIT_TOKEN_PER_CU
	msg := ts.SendRelay(providerAAcc.Addr.String(), consumerAAcc, []string{ts.spec.Index}, baserewards)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res.Rewards[0].Amount.Amount.Uint64(), baserewards*subscription.LIMIT_TOKEN_PER_CU)

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res.Rewards[0].Amount.Amount.Uint64(), baserewards*subscription.LIMIT_TOKEN_PER_CU+baserewards*subscription.LIMIT_TOKEN_PER_CU*ts.Keepers.Rewards.GetParams(ts.Ctx).MaxRewardBoost)
}

// the rewards here is spec payout allocation (full rewards from the pool), in this test the rewards for the providers are medium (second third of the graph)
func TestSpecAllocationProvidersRewards(t *testing.T) {
	ts := newTester(t)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err := ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	consumerAAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAAcc.Addr.String(), consumerAAcc.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	msg := ts.SendRelay(providerAAcc.Addr.String(), consumerAAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64())
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	//	require.Equal(t, res.Rewards[0].Amount, ts.plan.Price)

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	distBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.ProviderRewardsDistributionPool)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res.Rewards[0].Amount, ts.plan.Price.AddAmount(distBalance))
}

// the rewards here is the diminishing part of the reward, in this test the rewards for the providers are high (third third of the graph)
func TestProvidersDiminishingRewards(t *testing.T) {
	ts := newTester(t)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err := ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	for i := 0; i < 7; i++ {
		consumerAAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
		_, err = ts.TxSubscriptionBuy(consumerAAcc.Addr.String(), consumerAAcc.Addr.String(), ts.plan.Index, 1, false)
		require.Nil(t, err)

		msg := ts.SendRelay(providerAAcc.Addr.String(), consumerAAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64())
		_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
		require.Nil(t, err)
	}

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res.Rewards[0].Amount.Amount, ts.plan.Price.Amount.MulRaw(7))
	ts.TxDualstakingClaimRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String())

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	distBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.ProviderRewardsDistributionPool)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)

	require.Equal(t, res.Rewards[0].Amount.Amount, sdk.NewDecWithPrec(15, 1).MulInt(distBalance).Sub(sdk.NewDecWithPrec(5, 1).MulInt(ts.plan.Price.Amount.MulRaw(7))).TruncateInt())
}

// the rewards here is the zero since base rewards are very big, in this test the rewards for the providers are at the end of the graph
func TestProvidersEndRewards(t *testing.T) {
	ts := newTester(t)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err := ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	for i := 0; i < 50; i++ {
		consumerAAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
		_, err = ts.TxSubscriptionBuy(consumerAAcc.Addr.String(), consumerAAcc.Addr.String(), ts.plan.Index, 1, false)
		require.Nil(t, err)

		msg := ts.SendRelay(providerAAcc.Addr.String(), consumerAAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64())
		_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
		require.Nil(t, err)
	}

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res.Rewards[0].Amount.Amount, ts.plan.Price.Amount.MulRaw(50))
	ts.TxDualstakingClaimRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String())

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 0)
}

func Test2SpecsZeroShares(t *testing.T) {
	ts := newTester(t)
	spec2 := ts.spec
	spec2.Index = "mock2"
	spec2.Name = spec2.Index
	spec2.Shares = 0
	ts.AddSpec(spec2.Index, spec2)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, 2*testBalance)
	err := ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	err = ts.StakeProvider(providerAAcc.Addr.String(), spec2, testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	consumerAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAcc.Addr.String(), consumerAcc.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	msg := ts.SendRelay(providerAAcc.Addr.String(), consumerAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64())
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	consumerAcc2, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAcc2.Addr.String(), consumerAcc2.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	msg = ts.SendRelay(providerAAcc.Addr.String(), consumerAcc2, []string{spec2.Index}, ts.plan.Price.Amount.Uint64())
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 2)
	require.Equal(t, res.Rewards[0].Amount, ts.plan.Price)
	require.Equal(t, res.Rewards[1].Amount, ts.plan.Price)
	ts.TxDualstakingClaimRewards(providerAAcc.Addr.String(), "")

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	distBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.ProviderRewardsDistributionPool)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res.Rewards[0].Amount.Amount, distBalance)
}

func Test2SpecsDoubleShares(t *testing.T) {
	ts := newTester(t)
	spec2 := ts.spec
	spec2.Index = "mock2"
	spec2.Name = spec2.Index
	spec2.Shares *= 2
	ts.AddSpec(spec2.Index, spec2)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, 2*testBalance)
	err := ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	err = ts.StakeProvider(providerAAcc.Addr.String(), spec2, testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	consumerAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAcc.Addr.String(), consumerAcc.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	msg := ts.SendRelay(providerAAcc.Addr.String(), consumerAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64())
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	consumerAcc2, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAcc2.Addr.String(), consumerAcc2.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	msg = ts.SendRelay(providerAAcc.Addr.String(), consumerAcc2, []string{spec2.Index}, ts.plan.Price.Amount.Uint64())
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 2)
	require.Equal(t, res.Rewards[0].Amount, ts.plan.Price)
	require.Equal(t, res.Rewards[1].Amount, ts.plan.Price)
	ts.TxDualstakingClaimRewards(providerAAcc.Addr.String(), "")

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 2)
	require.Equal(t, res.Rewards[0].Amount.Amount, res.Rewards[1].Amount.Amount.MulRaw(2))
}

func TestBonusRewards3Providers(t *testing.T) {
	ts := newTester(t)

	providerAcc1, _ := ts.AddAccount(common.PROVIDER, 1, 2*testBalance)
	err := ts.StakeProvider(providerAcc1.Addr.String(), ts.spec, testBalance)
	require.Nil(t, err)

	providerAcc2, _ := ts.AddAccount(common.PROVIDER, 2, 2*testBalance)
	err = ts.StakeProvider(providerAcc2.Addr.String(), ts.spec, 2*testBalance)
	require.Nil(t, err)

	providerAcc3, _ := ts.AddAccount(common.PROVIDER, 3, 3*testBalance)
	err = ts.StakeProvider(providerAcc3.Addr.String(), ts.spec, 3*testBalance)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	consumerAcc, _ := ts.AddAccount(common.CONSUMER, 1, ts.plan.Price.Amount.Int64())
	_, err = ts.TxSubscriptionBuy(consumerAcc.Addr.String(), consumerAcc.Addr.String(), ts.plan.Index, 1, false)
	require.Nil(t, err)

	msg := ts.SendRelay(providerAcc1.Addr.String(), consumerAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64()/2)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	msg = ts.SendRelay(providerAcc2.Addr.String(), consumerAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64())
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	msg = ts.SendRelay(providerAcc3.Addr.String(), consumerAcc, []string{ts.spec.Index}, ts.plan.Price.Amount.Uint64()*2)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.Nil(t, err)

	// first months there are no bonus rewards, just payment ffrom the subscription
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	res, err := ts.QueryDualstakingDelegatorRewards(providerAcc1.Addr.String(), "", "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	ts.TxDualstakingClaimRewards(providerAcc1.Addr.String(), "")

	res, err = ts.QueryDualstakingDelegatorRewards(providerAcc2.Addr.String(), "", "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	ts.TxDualstakingClaimRewards(providerAcc2.Addr.String(), "")

	res, err = ts.QueryDualstakingDelegatorRewards(providerAcc3.Addr.String(), "", "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	ts.TxDualstakingClaimRewards(providerAcc3.Addr.String(), "")

	// now the provider should get all of the provider allocation
	ts.AdvanceMonths(1)
	distBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.ProviderRewardsDistributionPool)
	ts.AdvanceEpoch()

	res1, err := ts.QueryDualstakingDelegatorRewards(providerAcc1.Addr.String(), "", "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res1.Rewards[0].Amount.Amount, distBalance.QuoRaw(7).SubRaw(3))

	res2, err := ts.QueryDualstakingDelegatorRewards(providerAcc2.Addr.String(), "", "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res2.Rewards[0].Amount.Amount, distBalance.QuoRaw(7).MulRaw(2))

	res3, err := ts.QueryDualstakingDelegatorRewards(providerAcc3.Addr.String(), "", "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res3.Rewards[0].Amount.Amount, distBalance.QuoRaw(7).MulRaw(4).AddRaw(6))
}
