package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		specKeeper         types.SpecKeeper
		epochStorageKeeper types.EpochstorageKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper, accountKeeper types.AccountKeeper, specKeeper types.SpecKeeper, epochStorageKeeper types.EpochstorageKeeper,
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
		bankKeeper: bankKeeper, accountKeeper: accountKeeper, specKeeper: specKeeper, epochStorageKeeper: epochStorageKeeper,
	}
	epochStorageKeeper.AddFixationRegistry(string(types.KeyServicersToPairCount), func(ctx sdk.Context) any { return keeper.ServicersToPairCountRaw(ctx) })
	epochStorageKeeper.AddFixationRegistry(string(types.KeyStakeToMaxCUList), func(ctx sdk.Context) any { return keeper.StakeToMaxCUListRaw(ctx) })

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// we dont want to do the calculation here too, epochStorage keeper did it
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	return k.epochStorageKeeper.GetEpochStart(ctx) == uint64(ctx.BlockHeight())
}
