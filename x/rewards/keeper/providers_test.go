package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/x/rewards/types"
	subscription "github.com/lavanet/lava/x/subscription/keeper"
	"github.com/stretchr/testify/require"
)

// the rewards here zero
func TestZeroProvidersRewards(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	_, err := ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	require.Nil(ts.T, err)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err = ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
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
	ts.addValidators(1)
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	_, err := ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	require.Nil(ts.T, err)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err = ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
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
	require.Equal(t, res.Rewards[0].Amount.Amount.Uint64(), baserewards*subscription.LIMIT_TOKEN_PER_CU+baserewards*subscription.LIMIT_TOKEN_PER_CU*ts.Keepers.Rewards.GetParams(ts.Ctx).Providers.MaxRewardBoost)
}

// the rewards here is spec payout allocation (full rewards from the pool), in this test the rewards for the providers are medium (second third of the graph)
func TestSpecAllocationProvidersRewards(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	_, err := ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	require.Nil(ts.T, err)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err = ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
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
	distBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.ProviderDistributionPool)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)
	require.Equal(t, res.Rewards[0].Amount, ts.plan.Price.AddAmount(distBalance))
}

// the rewards here is the diminishing part of the reward, in this test the rewards for the providers are high (third third of the graph)
func TestProvidersDiminishingRewards(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	_, err := ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	require.Nil(ts.T, err)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err = ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
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
	distBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.ProviderDistributionPool)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorRewards(providerAAcc.Addr.String(), providerAAcc.Addr.String(), "")
	require.Nil(t, err)
	require.Len(t, res.Rewards, 1)

	require.Equal(t, res.Rewards[0].Amount.Amount, sdk.NewDecWithPrec(15, 1).MulInt(distBalance).Sub(sdk.NewDecWithPrec(5, 1).MulInt(ts.plan.Price.Amount.MulRaw(7))).TruncateInt())
}

// the rewards here is the zero since base rewards are very big, in this test the rewards for the providers are at the end of the graph
func TestProvidersEndRewards(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	_, err := ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	require.Nil(ts.T, err)

	providerAAcc, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err = ts.StakeProvider(providerAAcc.Addr.String(), ts.spec, testBalance)
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
