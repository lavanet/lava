package keeper

import (
	"encoding/binary"
	"fmt"
	"math"

	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper        types.BankKeeper
		accountKeeper     types.AccountKeeper
		specKeeper        types.SpecKeeper
		epochstorage      types.EpochstorageKeeper
		downtimeKeeper    types.DowntimeKeeper
		stakingKeeper     types.StakingKeeper
		dualstakingKeeper types.DualStakingKeeper
		// account name used by the distribution module to reward validators
		feeCollectorName string

		// used to operate the monthly refill of the validators and providers rewards pool mechanism
		// there is always a single timer that is expired in the next month
		// the timer subkey holds the block in which the timer will expire (not exact)
		// the timer data holds the number of months left for the allocation pools (until all funds are gone)
		refillRewardsPoolTS timerstoretypes.TimerStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	specKeeper types.SpecKeeper,
	epochStorageKeeper types.EpochstorageKeeper,
	downtimeKeeper types.DowntimeKeeper,
	stakingKeeper types.StakingKeeper,
	dualstakingKeeper types.DualStakingKeeper,
	feeCollectorName string,
	timerStoreKeeper types.TimerStoreKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	keeper := Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,

		bankKeeper:        bankKeeper,
		accountKeeper:     accountKeeper,
		specKeeper:        specKeeper,
		epochstorage:      epochStorageKeeper,
		downtimeKeeper:    downtimeKeeper,
		stakingKeeper:     stakingKeeper,
		dualstakingKeeper: dualstakingKeeper,

		feeCollectorName: feeCollectorName,
	}

	refillRewardsPoolTimerCallback := func(ctx sdk.Context, subkey, data []byte) {
		keeper.RefillRewardsPools(ctx, subkey, data)
		keeper.DistributeMonthlyBonusRewards(ctx)
	}

	// making an EndBlock timer store to make sure it'll happen after the BeginBlock that pays validators
	keeper.refillRewardsPoolTS = *timerStoreKeeper.NewTimerStoreEndBlock(storeKey, types.RefillRewardsPoolTimerPrefix).
		WithCallbackByBlockTime(refillRewardsPoolTimerCallback)

	return &keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// redeclaring BeginBlock for testing (this is not called outside of unit tests)
func (k Keeper) BeginBlock(ctx sdk.Context) {
	err := k.DistributeBlockReward(ctx)
	if err != nil {
		panic(err)
	}
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

	k.RefillValidatorsAllocationPool(ctx, monthsLeft, types.ValidatorsRewardsAllocationPoolName, types.ValidatorsRewardsDistributionPoolName)
	k.RefillValidatorsAllocationPool(ctx, monthsLeft, types.ProvidersAllocationPool, types.ProviderDistributionPool)

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

func (k Keeper) RefillValidatorsAllocationPool(ctx sdk.Context, monthsLeft uint64, allocationPool types.Pool, distributionPool types.Pool) {
	// burn remaining tokens in the block pool
	burnRate := k.GetParams(ctx).LeftoverBurnRate
	tokensToBurn := burnRate.MulInt(k.TotalPoolTokens(ctx, distributionPool)).TruncateInt()
	err := k.BurnPoolTokens(ctx, distributionPool, tokensToBurn)
	if err != nil {
		utils.LavaFormatError("critical - could not burn validators block pool tokens", err)
	}

	// transfer the new monthly quota (if allocation pool is expired, rewards=0)
	monthlyQuota := sdk.Coins{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.ZeroInt()}}
	if monthsLeft != 0 {
		validatorPoolBalance := k.TotalPoolTokens(ctx, allocationPool)
		monthlyQuota[0] = monthlyQuota[0].AddAmount(validatorPoolBalance.QuoRaw(int64(monthsLeft)))

		err = k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			string(allocationPool),
			string(distributionPool),
			monthlyQuota,
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

func (k Keeper) AllocationPoolMonthsLeft(ctx sdk.Context) int64 {
	_, _, data := k.refillRewardsPoolTS.GetFrontTimers(ctx, timerstoretypes.BlockTime)
	if len(data) == 0 {
		// something is wrong, don't panic but return largest duration
		utils.LavaFormatError("could not get allocation pool months left", fmt.Errorf("no timers found"))
		return math.MaxInt64
	}

	return int64(binary.BigEndian.Uint64(data[0]))
}

// BondedTargetFactor calculates the bonded target factor which is used to calculate the validators
// block rewards
func (k Keeper) BondedTargetFactor(ctx sdk.Context) cosmosMath.LegacyDec {
	params := k.GetParams(ctx)

	minBonded := params.MinBondedTarget
	maxBonded := params.MaxBondedTarget
	lowFactor := params.LowFactor
	bonded := k.stakingKeeper.BondedRatio(ctx)

	if bonded.GT(maxBonded) {
		return lowFactor
	}

	if bonded.LTE(minBonded) {
		return cosmosMath.LegacyOneDec()
	} else {
		// equivalent to: (maxBonded - bonded) / (maxBonded - minBonded)
		// 					  + lowFactor * (bonded - minBonded) / (maxBonded - minBonded)
		min_max_diff := maxBonded.Sub(minBonded)
		e1 := maxBonded.Sub(bonded).Quo(min_max_diff)
		e2 := bonded.Sub(minBonded).Quo(min_max_diff)
		return e1.Add(e2.Mul(lowFactor))
	}
}

// AddCollectedFees transfer the validators block rewards from the validators block pool to
// the fee collector account. This account is used by Cosmos' distribution module to send the
// validator rewards
func (k Keeper) AddCollectedFees(ctx sdk.Context, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, string(types.ValidatorsRewardsDistributionPoolName), k.feeCollectorName, fees)
}

func (k Keeper) InitRewardsRefillTS(ctx sdk.Context, gs timerstoretypes.GenesisState) {
	k.refillRewardsPoolTS.Init(ctx, gs)
}

// ExportRewardsRefillTS exports refill block pools timers data (for genesis)
func (k Keeper) ExportRewardsRefillTS(ctx sdk.Context) timerstoretypes.GenesisState {
	return k.refillRewardsPoolTS.Export(ctx)
}
