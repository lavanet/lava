package keeper

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/v2/x/timerstore/types"
)

func (k Keeper) DistributeBlockReward(ctx sdk.Context) {
	// get params for validator rewards calculation
	bondedTargetFactor := k.BondedTargetFactor(ctx)
	blocksToNextTimerExpiry := k.BlocksToNextTimerExpiry(ctx)

	// get validator distribution pool balance
	coins := k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName)
	distributionPoolBalance := coins.AmountOf(k.stakingKeeper.BondDenom(ctx))
	if blocksToNextTimerExpiry == 0 {
		utils.LavaFormatWarning("blocksToNextTimerExpiry is zero", fmt.Errorf("critical: Attempt to divide by zero"),
			utils.LogAttr("blocksToNextTimerExpiry", blocksToNextTimerExpiry),
			utils.LogAttr("distributionPoolBalance", distributionPoolBalance),
		)
		return
	}

	// validators bonus rewards = (distributionPoolBalance * bondedTargetFactor) / blocksToNextTimerExpiry
	validatorsRewards := bondedTargetFactor.MulInt(distributionPoolBalance).QuoInt64(blocksToNextTimerExpiry).TruncateInt()
	if !validatorsRewards.IsZero() {
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
		monthsLeft = types.RewardsAllocationPoolsLifetime
	} else {
		monthsLeft = binary.BigEndian.Uint64(data)
	}

	burnRate := k.GetParams(ctx).LeftoverBurnRate
	k.refillDistributionPool(ctx, monthsLeft, types.ValidatorsRewardsAllocationPoolName, types.ValidatorsRewardsDistributionPoolName, burnRate)
	k.refillDistributionPool(ctx, monthsLeft, types.ProvidersRewardsAllocationPool, types.ProviderRewardsDistributionPool, sdk.OneDec())

	if monthsLeft > 1 {
		monthsLeft -= 1
	}

	// update the months left of the allocation pool and encode it
	monthsLeftBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(monthsLeftBytes, monthsLeft)

	// open a new timer for next month
	nextMonth := utils.NextMonth(ctx.BlockTime()).UTC()
	k.refillRewardsPoolTS.AddTimerByBlockTime(ctx, uint64(nextMonth.Unix()), []byte(types.RefillRewardsPoolTimerName), monthsLeftBytes)

	coins := k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName)
	valDistPoolBalance := coins.AmountOf(k.stakingKeeper.BondDenom(ctx)).Int64()
	coins = k.TotalPoolTokens(ctx, types.ProviderRewardsDistributionPool)
	providerDistPoolBalance := coins.AmountOf(k.stakingKeeper.BondDenom(ctx)).Int64()
	nextRefillBlock := k.blocksToNextTimerExpiry(ctx, nextMonth.Unix()-ctx.BlockTime().UTC().Unix()) + ctx.BlockHeight()
	details := map[string]string{
		"allocation_pool_remaining_lifetime":   strconv.FormatUint(monthsLeft, 10),
		"validators_distribution_pool_balance": strconv.FormatInt(valDistPoolBalance, 10),
		"providers_distribution_pool_balance":  strconv.FormatInt(providerDistPoolBalance, 10),
		"leftover_burn_rate":                   burnRate.String(),
		"next_refill_time":                     nextMonth.String(),
		"next_refill_block":                    strconv.FormatInt(nextRefillBlock, 10),
	}

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.DistributionPoolRefillEventName, details, "distribution rewards pools refilled successfully")
}

func (k Keeper) refillDistributionPool(ctx sdk.Context, monthsLeft uint64, allocationPool types.Pool, distributionPool types.Pool, burnRate sdkmath.LegacyDec) {
	// burn remaining tokens in the distribution pool
	coins := k.TotalPoolTokens(ctx, distributionPool)
	distPoolBalance := coins.AmountOf(k.stakingKeeper.BondDenom(ctx))
	tokensToBurn := burnRate.MulInt(distPoolBalance).TruncateInt()
	err := k.BurnPoolTokens(ctx, distributionPool, tokensToBurn, k.stakingKeeper.BondDenom(ctx))
	if err != nil {
		utils.LavaFormatError("critical - could not burn distribution pool tokens", err,
			utils.Attribute{Key: "distribution_pool", Value: string(distributionPool)},
			utils.Attribute{Key: "tokens_to_burn", Value: tokensToBurn.String()},
		)
	}

	// transfer the new monthly quota (if allocation pool is expired, rewards=0)
	coins = k.TotalPoolTokens(ctx, allocationPool)
	allocPoolBalance := coins.AmountOf(k.stakingKeeper.BondDenom(ctx))
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
// the calculated blocks are multiplied with a slack factor (for error margin)
func (k Keeper) BlocksToNextTimerExpiry(ctx sdk.Context) int64 {
	timeToNextTimerExpiry := k.TimeToNextTimerExpiry(ctx)
	return k.blocksToNextTimerExpiry(ctx, timeToNextTimerExpiry)
}

func (k Keeper) blocksToNextTimerExpiry(ctx sdk.Context, timeToNextTimerExpiry int64) int64 {
	blockCreationTime := int64(k.downtimeKeeper.GetParams(ctx).DowntimeDuration.Seconds())
	if blockCreationTime == 0 {
		return 30
	}

	effectiveTimeToNextTimerExpiry := sdkmath.LegacyNewDec(timeToNextTimerExpiry)
	if timeToNextTimerExpiry != math.MaxInt64 {
		effectiveTimeToNextTimerExpiry = types.BlocksToTimerExpirySlackFactor.MulInt64(timeToNextTimerExpiry)
	}

	blocksToNextTimerExpiry := effectiveTimeToNextTimerExpiry.QuoInt64(blockCreationTime).Ceil().TruncateInt64()
	if blocksToNextTimerExpiry < 2 {
		return 2
	}

	return blocksToNextTimerExpiry
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
