package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
)

func NewKeeper(cdc codec.BinaryCodec, sk sdk.StoreKey) Keeper {
	return Keeper{
		storeKey: sk,
		cdc:      cdc,
	}
}

type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec
}

func (k Keeper) ExportGenesis(ctx sdk.Context) (*v1.GenesisState, error) {
	return new(v1.GenesisState), nil
}

func (k Keeper) ImportGenesis(ctx sdk.Context, gs *v1.GenesisState) error {
	return nil
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
}
