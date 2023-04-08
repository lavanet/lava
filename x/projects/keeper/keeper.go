package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	common "github.com/lavanet/lava/common"
	"github.com/lavanet/lava/x/projects/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		projectsFS      common.FixationStore
		developerKeysFS common.FixationStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	projectsfs := common.NewFixationStore(storeKey, cdc, types.ProjectsFixationPrefix)
	developerKeysfs := common.NewFixationStore(storeKey, cdc, types.DeveloperKeysFixationPrefix)

	return &Keeper{
		cdc:             cdc,
		storeKey:        storeKey,
		memKey:          memKey,
		paramstore:      ps,
		projectsFS:      *projectsfs,
		developerKeysFS: *developerKeysfs,
	}
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.projectsFS.AdvanceBlock(ctx)
	k.developerKeysFS.AdvanceBlock(ctx)
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
