package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/x/packages/types"
)

type (
	Keeper struct {
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace
		packagesFs common.FixationStore
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

	return &Keeper{
		memKey:     memKey,
		paramstore: ps,
		packagesFs: *common.NewFixationStore(storeKey, cdc, types.PackagesFixationStorePrefix),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetPackagesFixationStore() common.FixationStore {
	return k.packagesFs
}
