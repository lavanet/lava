package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/projects/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		epochstorageKeeper types.EpochStorageKeeper

		projectsFS      common.FixationStore
		developerKeysFS common.FixationStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	epochstorageKeeper types.EpochStorageKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	projectsfs := common.NewFixationStore(storeKey, cdc, types.ProjectsFixationPrefix)
	developerKeysfs := common.NewFixationStore(storeKey, cdc, types.DeveloperKeysFixationPrefix)

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

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.projectsFS.AdvanceBlock(ctx)
	k.developerKeysFS.AdvanceBlock(ctx)
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) ExportProjects(ctx sdk.Context) []commontypes.RawMessage {
	return k.projectsFS.Export(ctx)
}

func (k Keeper) InitProjects(ctx sdk.Context, data []commontypes.RawMessage) {
	k.projectsFS.Init(ctx, data)
}

func (k Keeper) ExportDevelopers(ctx sdk.Context) []commontypes.RawMessage {
	return k.developerKeysFS.Export(ctx)
}

func (k Keeper) InitDevelopers(ctx sdk.Context, data []commontypes.RawMessage) {
	k.developerKeysFS.Init(ctx, data)
}
