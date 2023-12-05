package keeper

import (
	"fmt"

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
		downtimeKeeper types.DowntimeKeeper
		stakingKeeper  types.StakingKeeper

		feeCollectorName string

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

// refillValidatorsBlockPool should transfer the monthly quota from the validators pool to the validators block pool
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

	// open a new timer for next month
	nextMonth := utils.NextMonth(ctx.BlockTime()).UTC().Unix()
	k.refillBlockPoolTS.AddTimerByBlockTime(ctx, uint64(nextMonth), nil, nil)
}

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

func (k Keeper) BlocksToNextEmission(ctx sdk.Context) int64 {
	// get the duration until the end of the next timer from the current block time
	blockTime := ctx.BlockTime().UTC().Unix()
	refillTime := k.refillBlockPoolTS.GetNextTimeoutBlockTime(ctx)
	durationUntilNextMonth := refillTime - uint64(blockTime)

	// get the block creation time
	downtimeParams := k.downtimeKeeper.GetParams(ctx)
	blockCreationTime := uint64(downtimeParams.DowntimeDuration.Seconds())

	// return the number of block until next month + 5%
	blocks := (durationUntilNextMonth / blockCreationTime) * 105 / 100
	return int64(blocks)
}

func (k Keeper) AddCollectedFees(ctx sdk.Context, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ValidatorsBlockPoolName, k.feeCollectorName, fees)
}
