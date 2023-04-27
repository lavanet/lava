package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/x/plans/types"
)

type (
	Keeper struct {
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace
		plansFS    common.FixationStore
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

	fs := *common.NewFixationStore(storeKey, cdc, types.PlanFixationStorePrefix)

	return &Keeper{
		memKey:     memKey,
		paramstore: ps,
		plansFS:    fs,
	}
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.plansFS.AdvanceBlock(ctx)
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
