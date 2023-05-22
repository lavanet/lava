package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper    types.BankKeeper
		accountKeeper types.AccountKeeper
		specKeeper    types.SpecKeeper

		fixationRegistries map[string]func(sdk.Context) any
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper, accountKeeper types.AccountKeeper, specKeeper types.SpecKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	keeper := &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,
		bankKeeper: bankKeeper, accountKeeper: accountKeeper, specKeeper: specKeeper,

		fixationRegistries: make(map[string]func(sdk.Context) any),
	}

	keeper.AddFixationRegistry(string(types.KeyEpochBlocks), func(ctx sdk.Context) any { return keeper.EpochBlocksRaw(ctx) })
	keeper.AddFixationRegistry(string(types.KeyEpochsToSave), func(ctx sdk.Context) any { return keeper.EpochsToSaveRaw(ctx) })
	keeper.AddFixationRegistry(string(types.KeyUnstakeHoldBlocks), func(ctx sdk.Context) any { return keeper.UnstakeHoldBlocksRaw(ctx) })
	keeper.AddFixationRegistry(string(types.KeyUnstakeHoldBlocksStatic), func(ctx sdk.Context) any { return keeper.UnstakeHoldBlocksStaticRaw(ctx) })

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) AddFixationRegistry(fixationKey string, getParamFunction func(sdk.Context) any) {
	if _, ok := k.fixationRegistries[fixationKey]; ok {
		panic(fmt.Sprintf("duplicate fixation registry %s", fixationKey))
	}
	k.fixationRegistries[fixationKey] = getParamFunction
}

func (k *Keeper) GetFixationRegistries() map[string]func(sdk.Context) any {
	return k.fixationRegistries
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	if k.IsEpochStart(ctx) {
		// run functions that are supposed to run in epoch start
		k.EpochStart(ctx)

		// Notify world we have a new session

		details := map[string]string{"height": fmt.Sprintf("%d", ctx.BlockHeight()), "description": "New Block Epoch Started"}
		utils.LogLavaEvent(ctx, k.Logger(ctx), "new_epoch", details, "")
	}
}
