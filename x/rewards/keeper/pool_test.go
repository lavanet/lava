package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
	"github.com/stretchr/testify/require"
)

// GENERAL NOTES:
// 1. To avoid simulating Cosmos' distribution module, all tests check the "Fee Collector"
//    account's balance since the distribution module takes funds from there to reward
//    the validators every new block
//
// 2. The rewards pool mechanism relies on a monthly EndBlock timer callback that refills the pool
//	  and opens a new timer. In some tests you'll see: AdvanceMonth -> AdvanceBlock -> EndBlock
//	  This is because AdvanceMonth advances a month minus 5 seconds, AdvanceBlock advances the
//	  time by 300sec (block time) and EndBlock calls the EndBlock functions of all keepers (so
//    the timer callback will be called). We need to call EndBlock because AdvanceBlock calls
//	  EndBlock for the previous block, updates the context with the new block height and time
//	  and then calls BeginBlock. The timer callback will be called only through EndBlock that
//	  uses the new height and time, so EndBlock needs to be called explicitly.

// TestRewardsModuleSetup tests that the setup of the rewards module is as expected
// The setup does the following (as in the Rewards module genesis):
//  1. transfer funds to the allocation pools
//  2. inits the refill rewards pools timer store
//  3. calls RefillRewardsPools to transfer funds to the distribution pools
//
// The expected results (after an epoch has passed) is that:
//  1. The allocation pool has the expected allocated funds minus one block reward
//  2. The distribution pool has the expected monthly quota minus one block reward
//  3. The fee collector has one block reward
func TestRewardsModuleSetup(t *testing.T) {
	ts := newTester(t, false)
	lifetime := int64(types.RewardsAllocationPoolsLifetime)

	// on init, the allocation pool lifetime should decrease by one
	res, err := ts.QueryRewardsPools()
	require.NoError(t, err)
	require.Equal(t, lifetime-1, res.AllocationPoolMonthsLeft)

	// in the end of the setup, there is an advancement of one block, so validator
	// rewards were distributed once. Since the block rewards depends on the distribution
	// pool balance (and it's not negligble), we'll calculate it manually
	expectedDistPoolBalanceBeforeReward := allocationPoolBalance / lifetime
	expectedBlocksToNextExpiry := ts.Keepers.Rewards.BlocksToNextTimerExpiry(ts.Ctx)
	require.NotEqual(t, int64(0), expectedBlocksToNextExpiry)
	expectedTargetFactor := int64(1)
	blockReward := expectedTargetFactor * expectedDistPoolBalanceBeforeReward / expectedBlocksToNextExpiry

	// after setup, the allocation pool got funded and sent the monthly quota to the distribution pool
	for _, pool := range res.Pools {
		switch pool.Name {
		case string(types.ValidatorsRewardsAllocationPoolName):
			require.Equal(t, allocationPoolBalance*(lifetime-1)/lifetime, pool.Balance.AmountOf(ts.BondDenom()).Int64())
		case string(types.ValidatorsRewardsDistributionPoolName):
			require.Equal(t, (allocationPoolBalance/lifetime)-blockReward, pool.Balance.AmountOf(ts.BondDenom()).Int64())
		case string(types.IprpcPoolName):
			require.True(t, pool.Balance.Empty())
		case string(types.PendingIprpcPoolName):
			require.True(t, pool.Balance.Empty())
		}
	}

	// check the fee collector's balance is the block reward (see general note 1 above)
	balance := ts.GetBalance(ts.feeCollector())
	require.Equal(t, blockReward, balance)
}

// TestBurnRateParam tests that the BurnRate param influences tokens burning as expected
// BurnRate = 1 -> on monthly refill, burn all previous funds in the distribution pool
// BurnRate = 0 -> on monthly refill, burn none of the previous funds in the distribution pool
func TestBurnRateParam(t *testing.T) {
	ts := newTester(t, true)
	lifetime := int64(types.RewardsAllocationPoolsLifetime)
	allocPoolBalance := ts.getPoolBalance(types.ValidatorsRewardsAllocationPoolName, ts.BondDenom()).Int64()
	// advance a month to trigger monthly pool refill callback
	// to see why these 3 are called, see general note 2
	resp, err := ts.QueryRewardsPools()
	require.NoError(t, err)
	ts.AdvanceBlock(time.Duration(resp.TimeToRefill) * time.Second)
	testkeeper.EndBlock(ts.Ctx, ts.Keepers)

	// default burn rate = 1, distribution pool's old balance should be wiped
	// current balance should be exactly the expected monthly quota minus block reward
	expectedMonthlyQuota := allocPoolBalance / (lifetime - 1)
	distPoolBalance := ts.getPoolBalance(types.ValidatorsRewardsDistributionPoolName, ts.BondDenom()).Int64()
	require.Equal(t, expectedMonthlyQuota, distPoolBalance)

	// change the burn rate param to be zero
	paramKey := string(types.KeyLeftoverBurnRate)
	zeroBurnRate, err := sdk.ZeroDec().MarshalJSON()
	require.NoError(t, err)
	paramVal := string(zeroBurnRate)
	err = ts.TxProposalChangeParam(types.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// advance a month to trigger monthly pool refill callback
	resp, err = ts.QueryRewardsPools()
	require.NoError(t, err)
	ts.AdvanceBlock(time.Duration(resp.TimeToRefill) * time.Second)
	prevDistPoolBalance := ts.getPoolBalance(types.ValidatorsRewardsDistributionPoolName, ts.BondDenom()).Int64()
	testkeeper.EndBlock(ts.Ctx, ts.Keepers)

	// burn rate = 0, distribution pool's old balance should not be wiped
	// current balance should be previous balance (minus block reward) plus new quota
	distPoolBalance = ts.getPoolBalance(types.ValidatorsRewardsDistributionPoolName, ts.BondDenom()).Int64()
	require.Equal(t, prevDistPoolBalance+expectedMonthlyQuota, distPoolBalance)
}

// TestAllocationPoolMonthlyQuota tests that the allocation pool transfers to the distribution pool
// its balance divided by months left (which should decrease with time). Also checks that if there are
// no months left, quota = 0 (and the chain doesn't panic)
func TestAllocationPoolMonthlyQuota(t *testing.T) {
	// after init, the allocation pool transfers funds to the distribution pool (no need to wait a month)
	ts := newTester(t, false)
	lifetime := int64(types.RewardsAllocationPoolsLifetime)

	// calc expectedMonthlyQuota. Check that it was subtracted from the allocation pool and added
	// to the distribution pool (its balance should be the monthly quota minus the fee collector's balance)
	expectedMonthlyQuota := allocationPoolBalance / lifetime
	currentAllocPoolBalance := ts.getPoolBalance(types.ValidatorsRewardsAllocationPoolName, ts.BondDenom())
	require.Equal(t, expectedMonthlyQuota, allocationPoolBalance-currentAllocPoolBalance.Int64())

	feeCollectorBalance := ts.GetBalance(ts.feeCollector())
	currentDistPoolBalance := ts.getPoolBalance(types.ValidatorsRewardsDistributionPoolName, ts.BondDenom())
	require.Equal(t, expectedMonthlyQuota, feeCollectorBalance+currentDistPoolBalance.Int64())

	// check the monthly quota is as expected with advancement of months
	for i := 0; i < int(lifetime-2); i++ {
		// to see why these 3 are called, see general note 2
		ts.AdvanceMonths(1)
		ts.AdvanceBlock()
		testkeeper.EndBlock(ts.Ctx, ts.Keepers)

		monthsLeft := ts.Keepers.Rewards.AllocationPoolMonthsLeft(ts.Ctx)
		prevAllocPoolBalance := currentAllocPoolBalance
		currentAllocPoolBalance = ts.getPoolBalance(types.ValidatorsRewardsAllocationPoolName, ts.BondDenom())

		var monthlyQuota int64
		if monthsLeft != 0 {
			monthlyQuota = currentAllocPoolBalance.Int64() / monthsLeft
			require.Equal(t, expectedMonthlyQuota, monthlyQuota) // the monthly quota is constant throughout the allocation pool lifetime
		}

		require.Equal(t, monthlyQuota, prevAllocPoolBalance.Sub(currentAllocPoolBalance).Int64())
	}

	// in the last month, the allocation pool's balance should be equal to the monthly quota
	require.Equal(t, expectedMonthlyQuota, currentAllocPoolBalance.Int64())

	// advance month to deplet the allocation pool funds
	// advance 2 blocks to transfer the last quota and send it to the fee collector
	ts.AdvanceMonths(1)
	ts.AdvanceBlock()
	testkeeper.EndBlock(ts.Ctx, ts.Keepers)
	currentAllocPoolBalance = ts.getPoolBalance(types.ValidatorsRewardsAllocationPoolName, ts.BondDenom())
	require.True(t, currentAllocPoolBalance.IsZero())

	// advance another month to distribute the last quota to the fee collector
	ts.AdvanceMonths(1)
	ts.AdvanceBlock()
	feeCollectorBalance = ts.GetBalance(ts.feeCollector())

	// the several more months and verify the allocation pool is empty and that the fee collector balance is the same
	for i := 0; i < 3; i++ {
		ts.AdvanceMonths(1)
		ts.AdvanceBlock()
		testkeeper.EndBlock(ts.Ctx, ts.Keepers)

		currentAllocPoolBalance = ts.getPoolBalance(types.ValidatorsRewardsAllocationPoolName, ts.BondDenom())
		require.True(t, currentAllocPoolBalance.IsZero())

		currentFeeCollectorBalance := ts.GetBalance(ts.feeCollector())
		require.Equal(t, feeCollectorBalance, currentFeeCollectorBalance)
	}
}

// TestValidatorBlockRewards tests that the expected block reward is transferred to the fee collector
// the reward should be: (distributionPoolBalance * bondedTargetFactor) / blocksToNextTimerExpiry
func TestValidatorBlockRewards(t *testing.T) {
	ts := newTester(t, false)

	// create validator
	stakingSupply := ts.Keepers.StakingKeeper.StakingTokenSupply(ts.Ctx)
	valInitBalance := stakingSupply.QuoRaw(3) // specifically picked to make staking module's BondedRatio to be 0.25
	ts.AddAccount(common.VALIDATOR, 0, valInitBalance.Int64())
	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(validator, valInitBalance)

	// by default, BondedRatio staking module param is smaller than MinBonded rewards module param
	// so bondedTargetFactor = 1. We change MinBonded to zero to change bondedTargetFactor
	params := types.DefaultParams()
	params.MinBondedTarget = sdk.ZeroDec()
	params.MaxBondedTarget = sdk.NewDecWithPrec(8, 1) // 0.8
	params.LowFactor = sdk.NewDecWithPrec(5, 1)       // 0.5
	params.LeftoverBurnRate = sdk.OneDec()
	ts.Keepers.Rewards.SetParams(ts.Ctx, params)

	// calc the expected BondedTargetFactor with its formula. with the values defined above,
	// and bondedRatio = 0.25, should be (0.8 - 0.25) / 0.8 + 0.5 * (0.25/0.8) = 0.84375
	// compare the new block reward to refBlockReward
	expectedBondedTargetFactor := sdk.NewDecWithPrec(84375, 5).TruncateInt() // 0.84375

	// verify that the current reward amount is as expected by checking the bondedTargetFactor alone
	res, err := ts.QueryRewardsBlockReward()
	require.NoError(t, err)
	blockReward := res.Reward.Amount
	distPoolBalance := ts.getPoolBalance(types.ValidatorsRewardsDistributionPoolName, ts.BondDenom())
	blocksToNextExpiry := ts.Keepers.Rewards.BlocksToNextTimerExpiry(ts.Ctx)
	bondedTargetFactor := sdk.OneDec().MulInt(blockReward).MulInt64(blocksToNextExpiry).QuoInt(distPoolBalance).TruncateInt()
	require.True(t, bondedTargetFactor.Equal(expectedBondedTargetFactor))

	// return the params to default values
	ts.Keepers.Rewards.SetParams(ts.Ctx, types.DefaultParams())
	minBonded := ts.Keepers.Rewards.GetParams(ts.Ctx).MinBondedTarget
	require.True(t, minBonded.Equal(types.DefaultMinBondedTarget))

	// get new reference reward
	res, err = ts.QueryRewardsBlockReward()
	require.NoError(t, err)
	blockReward = res.Reward.Amount

	// transfer half of the total distribution pool balance to the allocation pool
	distPoolBalance = ts.getPoolBalance(types.ValidatorsRewardsDistributionPoolName, ts.BondDenom())
	err = ts.Keepers.BankKeeper.SendCoinsFromModuleToModule(
		ts.Ctx,
		string(types.ValidatorsRewardsDistributionPoolName),
		string(types.ValidatorsRewardsAllocationPoolName),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), distPoolBalance.QuoRaw(2))),
	)
	require.NoError(t, err)

	// since we only halved the distribution pool balance, the reward should be half of the reference block reward
	expectedBlockReward := blockReward.QuoRaw(2)
	res, err = ts.QueryRewardsBlockReward()
	require.NoError(t, err)
	blockReward = res.Reward.Amount
	require.True(t, blockReward.Equal(expectedBlockReward))

	// transfer funds back
	err = ts.Keepers.BankKeeper.SendCoinsFromModuleToModule(
		ts.Ctx,
		string(types.ValidatorsRewardsAllocationPoolName),
		string(types.ValidatorsRewardsDistributionPoolName),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), distPoolBalance.QuoRaw(2))),
	)
	require.NoError(t, err)

	// finally, check that the blocksToNextExpiry affects the block reward as expected
	// first, get the reference blockToExpiry and advance a block
	refBlocksToExpiry := ts.Keepers.Rewards.BlocksToNextTimerExpiry(ts.Ctx) - 1
	ts.AdvanceBlock()

	// query for the current reward and isolate the blocksToExpiry. Compare it to the ref value
	res, err = ts.QueryRewardsBlockReward()
	require.NoError(t, err)
	blockReward = res.Reward.Amount
	bondedTargetFactor = ts.Keepers.Rewards.BondedTargetFactor(ts.Ctx).TruncateInt()
	distPoolBalance = ts.getPoolBalance(types.ValidatorsRewardsDistributionPoolName, ts.BondDenom())
	blocksToNextExpiry = bondedTargetFactor.Mul(distPoolBalance).Quo(blockReward).Int64()
	require.Equal(t, refBlocksToExpiry, blocksToNextExpiry)
}

// TestBlocksAndTimeToNextExpiry tests that the time/blocks to the next timer expiry are as expected
func TestBlocksAndTimeToNextExpiry(t *testing.T) {
	ts := newTester(t, false)

	// TimeToNextTimerExpiry should be equal to the number of seconds in a month
	blockTime := ts.BlockTime()
	nextMonth := utils.NextMonth(blockTime)
	secondsInAMonth := nextMonth.UTC().Unix() - blockTime.UTC().Unix()
	timeToExpiry := ts.Keepers.Rewards.TimeToNextTimerExpiry(ts.Ctx)
	require.Equal(t, secondsInAMonth, timeToExpiry)

	// BlocksToNextTimerExpiry should be equal to the number of blocks that pass in a month (rounding up) +5%
	blockCreationTime := int64(ts.Keepers.Downtime.GetParams(ts.Ctx).DowntimeDuration.Seconds())
	blocksInAMonth := types.BlocksToTimerExpirySlackFactor.MulInt64(secondsInAMonth).QuoInt64(blockCreationTime).Ceil().TruncateInt64()
	blocksToExpiry := ts.Keepers.Rewards.BlocksToNextTimerExpiry(ts.Ctx)
	require.Equal(t, blocksInAMonth, blocksToExpiry)

	// Advance 3 blocks and check again
	ts.AdvanceBlocks(3)
	expectedTimeToExpiry := secondsInAMonth - 3*blockCreationTime
	timeToExpiry = ts.Keepers.Rewards.TimeToNextTimerExpiry(ts.Ctx)
	require.Equal(t, expectedTimeToExpiry, timeToExpiry)

	expectedBlocksToExpiry := blocksInAMonth - 3
	blocksToExpiry = ts.Keepers.Rewards.BlocksToNextTimerExpiry(ts.Ctx)
	require.Equal(t, expectedBlocksToExpiry, blocksToExpiry)
}

// TestBondedTargetFactorEdgeCases checks the bondedTargetFactor's calculation edge cases
// BondedTargetFactor = 1  --  if bondedRatio < minBonded
// BondedTargetFactor = (maxBonded - bondedRatio) / (maxBonded - minBonded) + lowFactor * (bondedRatio - minBonded) / (maxBonded - minBonded)  --  if bondedRatio > minBonded
// BondedTargetFactor = 1  --  if bondedRatio > maxBonded
func TestBondedTargetFactorEdgeCases(t *testing.T) {
	ts := newTester(t, false)
	// Initial BondedRatio value is 0
	playbook := []struct {
		name                       string
		minBonded                  string
		maxBonded                  string // must be maxBonded > minBonded for all testcases
		lowFactor                  string
		unzeroBondedRatio          bool // make bondedRatio value to be 0.25
		expectedBondedTargetFactor string
	}{
		{"bondedRatio < minBonded", "0.6", "0.8", "0.5", false, "1"},                              // expected = 1
		{"bondedRatio < minBonded different params", "0.001", "0.3", "0.1", false, "1"},           // expected = 1
		{"bondedRatio = minBonded", "0.25", "0.3", "0.1", true, "1"},                              // expected = 1, from here down BondedRatio = 0.25
		{"bondedRatio > minBonded (bondedRatio < maxBonded)", "0.2", "0.3", "0.5", false, "0.75"}, // expected = long calc = 0.75
		{"bondedRatio > maxBonded", "0.1", "0.2", "0.7", false, "0.7"},                            // expected = LowFactor
	}

	for _, tt := range playbook {
		t.Run(tt.name, func(t *testing.T) {
			params := types.Params{
				MinBondedTarget:                     sdk.MustNewDecFromStr(tt.minBonded),
				MaxBondedTarget:                     sdk.MustNewDecFromStr(tt.maxBonded),
				LowFactor:                           sdk.MustNewDecFromStr(tt.lowFactor),
				LeftoverBurnRate:                    types.DefaultLeftOverBurnRate,
				MaxRewardBoost:                      types.DefaultMaxRewardBoost,
				ValidatorsSubscriptionParticipation: types.DefaultValidatorsSubscriptionParticipation,
				IbcIprpcExpiration:                  types.DefaultIbcIprpcExpiration,
			}
			ts.Keepers.Rewards.SetParams(ts.Ctx, params)

			if tt.unzeroBondedRatio {
				stakingSupply := ts.Keepers.StakingKeeper.StakingTokenSupply(ts.Ctx)
				valInitBalance := stakingSupply.QuoRaw(3) // specifically picked to make staking module's BondedRatio to be 0.25
				ts.AddAccount(common.VALIDATOR, 0, valInitBalance.Int64())
				validator, _ := ts.GetAccount(common.VALIDATOR, 0)
				ts.TxCreateValidator(validator, valInitBalance)
			}

			bondedTargetFactor := ts.Keepers.Rewards.BondedTargetFactor(ts.Ctx)
			require.Equal(t, sdk.MustNewDecFromStr(tt.expectedBondedTargetFactor), bondedTargetFactor)
		})
	}
}

// TestRefillPoolsTimerStore checks that the refill rewards pool timer store acts as expected:
// 1. There's a single timer at all times the expires after a month
// 2. The timer's data contains the months left before the allocation pool's funds are depleted
func TestRefillPoolsTimerStore(t *testing.T) {
	ts := newTester(t, false)
	lifetime := int64(types.RewardsAllocationPoolsLifetime)

	req := &timerstoretypes.QueryAllTimersRequest{
		StoreKey: types.StoreKey,
		Prefix:   types.RefillRewardsPoolTimerPrefix,
	}
	// check everything throughout the entire lifetime of the allocation pool (and beyond)
	defaultBlockTime := ts.Keepers.Downtime.GetParams(ts.Ctx).DowntimeDuration.Seconds()
	month := ts.GetNextMonth(ts.BlockTime()) - ts.BlockTime().UTC().Unix()
	for i := 0; i < int(lifetime+2); i++ {
		res, err := ts.Keepers.TimerStoreKeeper.AllTimers(ts.GoCtx, req)
		require.NoError(t, err)
		require.Equal(t, 1, len(res.BlockTimeTimers))
		require.Equal(t, 0, len(res.BlockHeightTimers))

		expiry := ts.Keepers.Rewards.TimeToNextTimerExpiry(ts.Ctx)
		require.InDelta(t, month, expiry, defaultBlockTime)

		var expectedMonthsLeft int64
		if i < int(lifetime) {
			expectedMonthsLeft = lifetime - int64(i) - 1 // setup progressed one month so we check with -1
		}
		if expectedMonthsLeft == 0 {
			expectedMonthsLeft = 1
		}
		monthsLeft := ts.Keepers.Rewards.AllocationPoolMonthsLeft(ts.Ctx)
		require.Equal(t, expectedMonthsLeft, monthsLeft)

		ts.AdvanceMonths(1)
		ts.AdvanceBlock()
		testkeeper.EndBlock(ts.Ctx, ts.Keepers)
		month = ts.GetNextMonth(ts.BlockTime()) - ts.BlockTime().UTC().Unix()
	}
}
