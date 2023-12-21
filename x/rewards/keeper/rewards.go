package keeper

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
)

func (k Keeper) DistributeBlockReward(ctx sdk.Context) {
	// get params for validator rewards calculation
	bondedTargetFactor := k.bondedTargetFactor(ctx)
	blocksToNextTimerExpiry := k.BlocksToNextTimerExpiry(ctx)

	// get validator distribution pool balance
	distributionPoolBalance := k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName)

	// validators bonus rewards = (distributionPoolBalance * bondedTargetFactor) / blocksToNextTimerExpiry
	validatorsRewards := bondedTargetFactor.MulInt(distributionPoolBalance).QuoInt64(blocksToNextTimerExpiry).TruncateInt()
	if validatorsRewards.IsZero() {
		// no rewards is not necessarily an error -> print warning and return
		utils.LavaFormatWarning("validators block rewards is zero", fmt.Errorf(""),
			utils.Attribute{Key: "bonded_target_factor", Value: bondedTargetFactor.String()},
			utils.Attribute{Key: "distribution_pool_balance", Value: distributionPoolBalance.String()},
			utils.Attribute{Key: "blocks_to_next_timer_expiry", Value: strconv.FormatInt(blocksToNextTimerExpiry, 10)},
		)
	} else {
		coins := sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), validatorsRewards))

		// distribute rewards to validators (same as Cosmos mint module)
		err := k.addCollectedFees(ctx, coins)
		if err != nil {
			utils.LavaFormatWarning("could not send validators rewards to fee collector", err,
				utils.Attribute{Key: "rewards", Value: coins.String()},
			)
		}
	}
}

// addCollectedFees transfer the validators block rewards from the validators distribution pool to
// the fee collector account. This account is used by Cosmos' distribution module to send the
// validator rewards
func (k Keeper) addCollectedFees(ctx sdk.Context, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, string(types.ValidatorsRewardsDistributionPoolName), k.feeCollectorName, fees)
}

// RefillRewardsPools is called once a month (as a timer callback). it does the following for validators:
//  1. burns the current token in the validators distribution pool by the burn rate
//  2. transfers the monthly tokens quota from the validators allocation pool to the validators distribution pool
//  3. opens a new timer for the next month (and encodes the expiry block and months left to allocation pool in it)
//
// for providers:
// TBD
func (k Keeper) RefillRewardsPools(ctx sdk.Context, _ []byte, data []byte) {
	// get the months left for the allocation pools
	var monthsLeft uint64
	if len(data) == 0 {
		monthsLeft = uint64(types.RewardsAllocationPoolsLifetime)
	} else {
		monthsLeft = binary.BigEndian.Uint64(data)
	}

	k.refillDistributionPool(ctx, monthsLeft, types.ValidatorsRewardsAllocationPoolName, types.ValidatorsRewardsDistributionPoolName, k.GetParams(ctx).LeftoverBurnRate)
	k.refillDistributionPool(ctx, monthsLeft, types.ProvidersRewardsAllocationPool, types.ProviderRewardsDistributionPool, sdk.OneDec())

	if monthsLeft > 0 {
		monthsLeft -= 1
	}

	// calculate the block in which the timer will expire (+5% for errors)
	nextMonth := utils.NextMonth(ctx.BlockTime()).UTC().Unix()
	durationUntilNextMonth := nextMonth - ctx.BlockTime().UTC().Unix()
	blockCreationTime := k.downtimeKeeper.GetParams(ctx).DowntimeDuration.Seconds()
	blocksToNextTimerExpiry := ((durationUntilNextMonth / int64(blockCreationTime)) * 105 / 100) + ctx.BlockHeight()

	// update the months left of the allocation pool and encode it
	monthsLeftBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(monthsLeftBytes, monthsLeft)

	// open a new timer for next month
	blocksToNextTimerExpirybytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blocksToNextTimerExpirybytes, uint64(blocksToNextTimerExpiry))
	k.refillRewardsPoolTS.AddTimerByBlockTime(ctx, uint64(nextMonth), blocksToNextTimerExpirybytes, monthsLeftBytes)
}

func (k Keeper) refillDistributionPool(ctx sdk.Context, monthsLeft uint64, allocationPool types.Pool, distributionPool types.Pool, burnRate sdkmath.LegacyDec) {
	// burn remaining tokens in the distribution pool
	tokensToBurn := burnRate.MulInt(k.TotalPoolTokens(ctx, distributionPool)).TruncateInt()
	err := k.BurnPoolTokens(ctx, distributionPool, tokensToBurn)
	if err != nil {
		utils.LavaFormatError("critical - could not burn distribution pool tokens", err,
			utils.Attribute{Key: "distribution_pool", Value: string(distributionPool)},
			utils.Attribute{Key: "tokens_to_burn", Value: tokensToBurn.String()},
		)
	}

	// transfer the new monthly quota (if allocation pool is expired, rewards=0)
	allocPoolBalance := k.TotalPoolTokens(ctx, allocationPool)
	if monthsLeft != 0 && !allocPoolBalance.IsZero() {
		monthlyQuota := sdk.Coin{Denom: k.stakingKeeper.BondDenom(ctx), Amount: allocPoolBalance.QuoRaw(int64(monthsLeft))}

		err = k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			string(allocationPool),
			string(distributionPool),
			sdk.NewCoins(monthlyQuota),
		)
		if err != nil {
			panic(err)
		}
	}
}

// BlocksToNextTimerExpiry extracts the timer's expiry block from the timer's subkey
// and returns the amount of blocks remaining (according to the current block height)
func (k Keeper) BlocksToNextTimerExpiry(ctx sdk.Context) int64 {
	keys, _, _ := k.refillRewardsPoolTS.GetFrontTimers(ctx, timerstoretypes.BlockTime)
	if len(keys) == 0 {
		// something is wrong, don't panic but make validators rewards 0
		utils.LavaFormatError("could not get blocks to next timer expiry", fmt.Errorf("no timers found"))
		return math.MaxInt64
	}
	return int64(binary.BigEndian.Uint64(keys[0])) - ctx.BlockHeight()
}

// TimeToNextTimerExpiry returns the time in which the timer will expire (according
// to the current block time)
func (k Keeper) TimeToNextTimerExpiry(ctx sdk.Context) int64 {
	_, expiries, _ := k.refillRewardsPoolTS.GetFrontTimers(ctx, timerstoretypes.BlockTime)
	if len(expiries) == 0 {
		// something is wrong, don't panic but return largest duration
		utils.LavaFormatError("could not get time to next timer expiry", fmt.Errorf("no timers found"))
		return math.MaxInt64
	}

	return int64(expiries[0]) - ctx.BlockTime().UTC().Unix()
}

// AllocationPoolMonthsLeft returns the amount of months the allocation pools
// have left before all their are depleted
func (k Keeper) AllocationPoolMonthsLeft(ctx sdk.Context) int64 {
	_, _, data := k.refillRewardsPoolTS.GetFrontTimers(ctx, timerstoretypes.BlockTime)
	if len(data) == 0 {
		// something is wrong, don't panic but return largest duration
		utils.LavaFormatError("could not get allocation pool months left", fmt.Errorf("no timers found"))
		return math.MaxInt64
	}

	return int64(binary.BigEndian.Uint64(data[0]))
}
