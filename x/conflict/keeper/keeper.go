package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/v2/x/conflict/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		pairingKeeper      types.PairingKeeper
		epochstorageKeeper types.EpochstorageKeeper
		specKeeper         types.SpecKeeper
		stakingKeeper      types.StakingKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	pairingKeeper types.PairingKeeper,
	epochstorageKeeper types.EpochstorageKeeper,
	specKeeper types.SpecKeeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		memKey:             memKey,
		paramstore:         ps,
		bankKeeper:         bankKeeper,
		accountKeeper:      accountKeeper,
		pairingKeeper:      pairingKeeper,
		epochstorageKeeper: epochstorageKeeper,
		specKeeper:         specKeeper,
		stakingKeeper:      stakingKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// we dont want to do the calculation here too, epochStorage keeper did it
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	return k.epochstorageKeeper.GetEpochStart(ctx) == uint64(ctx.BlockHeight())
}

func (k Keeper) PushFixations(ctx sdk.Context) {
	k.epochstorageKeeper.PushFixatedParams(ctx, 0, 0)
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.CheckAndHandleAllVotes(ctx)
}
