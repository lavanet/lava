package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	common "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper       types.BankKeeper
		accountKeeper    types.AccountKeeper
		monthlyRewardsTS timerstoretypes.TimerStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
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

		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
	}

	subsTimerCallback := func(ctx sdk.Context, subkey, _ []byte) {
		keeper.DistributeMonthlyBonusRewards(ctx)
	}

	keeper.monthlyRewardsTS = *timerStoreKeeper.NewTimerStoreBeginBlock(storeKey, types.MonthlyRewardsTSPrefix).
		WithCallbackByBlockTime(subsTimerCallback)

	return &keeper
}

func (k Keeper) SetNextMonthRewardTime(ctx sdk.Context) {
	k.monthlyRewardsTS.AddTimerByBlockTime(ctx, uint64(common.NextMonth(ctx.BlockTime()).Unix()), []byte{}, []byte{})
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
