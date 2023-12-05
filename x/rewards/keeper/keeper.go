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
		specKeeper     types.SpecKeeper
		epochstorage   types.EpochstorageKeeper
		downtimeKeeper types.DowntimeKeeper
		stakingKeeper  types.StakingKeeper

		monthlyRewardsTS timerstoretypes.TimerStore

		feeCollectorName string
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
		specKeeper:     specKeeper,
		epochstorage:   epochStorageKeeper,
		downtimeKeeper: downtimeKeeper,
		stakingKeeper:  stakingKeeper,

		feeCollectorName: feeCollectorName,
	}

	subsTimerCallback := func(ctx sdk.Context, subkey, _ []byte) {
		keeper.DistributeMonthlyBonusRewards(ctx)
	}

	keeper.monthlyRewardsTS = *timerStoreKeeper.NewTimerStoreBeginBlock(storeKey, types.MonthlyRewardsTSPrefix).
		WithCallbackByBlockTime(subsTimerCallback)

	return &keeper
}

func (k Keeper) SetNextMonthRewardTime(ctx sdk.Context) {
	k.monthlyRewardsTS.AddTimerByBlockTime(ctx, uint64(utils.NextMonth(ctx.BlockTime()).Unix()), []byte{}, []byte{})
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
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
