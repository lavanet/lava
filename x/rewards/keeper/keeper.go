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

		bankKeeper     types.BankKeeper
		accountKeeper  types.AccountKeeper
		downtimeKeeper types.DowntimeKeeper // used to get the approximate creation time for blocks
		stakingKeeper  types.StakingKeeper

		// account name used by the distribution module to reward validators
		feeCollectorName string

		// used to operate the monthly refill of the validators block pool mechanism
		// there is a single timer in all times that is expired in the next month
		// the timer subkey holds the block in which the timer will expire (not exact)
		refillBlockPoolTS timerstoretypes.TimerStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	downtimeKeeper types.DowntimeKeeper,
	stakingKeeper types.StakingKeeper,
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

		bankKeeper:     bankKeeper,
		accountKeeper:  accountKeeper,
		downtimeKeeper: downtimeKeeper,
		stakingKeeper:  stakingKeeper,

		feeCollectorName: feeCollectorName,
	}

	refillBlockPoolTimerCallback := func(ctx sdk.Context, subkey, _ []byte) {
		keeper.refillValidatorsBlockPool(ctx, subkey)
	}

	keeper.refillBlockPoolTS = *timerStoreKeeper.NewTimerStoreEndBlock(storeKey, types.RefillBlockPoolTimerPrefix).
		WithCallbackByBlockTime(refillBlockPoolTimerCallback)

	return &keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// refillValidatorsBlockPool is called once a month (as a timer callback). it does the following:
// 1. burns the current token in the validators block pool by the burn rate
// 2. transfers the monthly tokens quota from the validators pool to the validators block pool
// 3. opens a new timer for the next month (and encodes the expiry block in it)
func (k Keeper) refillValidatorsBlockPool(ctx sdk.Context, _ []byte) {
	// burn remaining tokens in the block pool
	burnRate := k.GetParams(ctx).LeftoverBurnRate
	tokensToBurn := burnRate.MulInt(k.TotalPoolTokens(ctx, types.ValidatorsBlockPoolName)).TruncateInt()
	err := k.BurnPoolTokens(ctx, types.ValidatorsBlockPoolName, tokensToBurn)
	if err != nil {
		utils.LavaFormatError("critical - could not burn validators block pool tokens", err)
	}

	// transfer the new monthly quota
	validatorPoolBalance := k.TotalPoolTokens(ctx, types.ValidatorsPoolName)
	err = k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.ValidatorsPoolName,
		types.ValidatorsBlockPoolName,
		sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, validatorPoolBalance)),
	)
	if err != nil {
		panic(err)
	}

	// calculate the block in which the timer will expire (+5% for errors)
	nextMonth := utils.NextMonth(ctx.BlockTime()).UTC().Unix()
	durationUntilNextMonth := nextMonth - ctx.BlockTime().UTC().Unix()
	blockCreationTime := k.downtimeKeeper.GetParams(ctx).DowntimeDuration.Seconds()
	blocksToNextTimerExpiry := durationUntilNextMonth / int64(blockCreationTime) * 105 / 100

	// open a new timer for next month
	blocksToNextTimerExpirybytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blocksToNextTimerExpirybytes, uint64(blocksToNextTimerExpiry))
	k.refillBlockPoolTS.AddTimerByBlockTime(ctx, uint64(nextMonth), blocksToNextTimerExpirybytes, nil)
}

// BlocksToNextTimerExpiry is a wrapper function that extracts the timer's expiry block
// for the timer's subkey and returns the amount of blocks remaining (according to the
// current block height)
func (k Keeper) BlocksToNextTimerExpiry(ctx sdk.Context) int64 {
	data, _ := k.refillBlockPoolTS.GetFrontTimers(ctx, timerstoretypes.BlockTime)
	if len(data) == 0 {
		// something is wrong, don't panic but make validators rewards 0
		return math.MaxInt64
	}
	return int64(binary.BigEndian.Uint64(data[0])) - ctx.BlockHeight()
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
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ValidatorsBlockPoolName, k.feeCollectorName, fees)
}
