package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	fixationtypes "github.com/lavanet/lava/v2/x/fixationstore/types"
	"github.com/lavanet/lava/v2/x/projects/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		epochstorageKeeper types.EpochStorageKeeper

		projectsFS      fixationtypes.FixationStore
		developerKeysFS fixationtypes.FixationStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	epochstorageKeeper types.EpochStorageKeeper,
	fixationStoreKeeper types.FixationStoreKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	projectsfs := fixationStoreKeeper.NewFixationStore(storeKey, types.ProjectsFixationPrefix)
	developerKeysfs := fixationStoreKeeper.NewFixationStore(storeKey, types.DeveloperKeysFixationPrefix)

	return &Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		memKey:             memKey,
		paramstore:         ps,
		projectsFS:         *projectsfs,
		developerKeysFS:    *developerKeysfs,
		epochstorageKeeper: epochstorageKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) ExportProjects(ctx sdk.Context) fixationtypes.GenesisState {
	return k.projectsFS.Export(ctx)
}

func (k Keeper) InitProjects(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.projectsFS.Init(ctx, gs)
}

func (k Keeper) ExportDevelopers(ctx sdk.Context) fixationtypes.GenesisState {
	return k.developerKeysFS.Export(ctx)
}

func (k Keeper) InitDevelopers(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.developerKeysFS.Init(ctx, gs)
}
