package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/x/servicer/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		evidenceKeeper     types.EvidenceKeeper
		specKeeper         types.SpecKeeper
		userKeeper         types.UserKeeper
		epochStorageKeeper types.EpochStorageKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper, accountKeeper types.AccountKeeper, evidenceKeeper types.EvidenceKeeper, specKeeper types.SpecKeeper, userKeeper types.UserKeeper, epochStorageKeeper types.EpochStorageKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{

		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,
		bankKeeper: bankKeeper, accountKeeper: accountKeeper, evidenceKeeper: evidenceKeeper, specKeeper: specKeeper, userKeeper: userKeeper, epochStorageKeeper: epochStorageKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

//we dont want to do the calculation here too, epochStorage keeper did it
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	return k.epochStorageKeeper.GetEpochStart(ctx) == uint64(ctx.BlockHeight())
}
