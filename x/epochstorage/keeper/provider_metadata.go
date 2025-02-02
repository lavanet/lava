package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/x/epochstorage/types"
)

func (k Keeper) GetMetadata(ctx sdk.Context, provider string) (types.ProviderMetadata, error) {
	return k.providersMetaData.Get(ctx, provider)
}

func (k Keeper) SetMetadata(ctx sdk.Context, metadata types.ProviderMetadata) {
	err := k.providersMetaData.Set(ctx, metadata.Provider, metadata)
	if err != nil {
		panic(err)
	}
}

func (k Keeper) RemoveMetadata(ctx sdk.Context, provider string) error {
	return k.providersMetaData.Remove(ctx, provider)
}

func (k Keeper) SetAllMetadata(ctx sdk.Context, metadata []types.ProviderMetadata) {
	for _, md := range metadata {
		k.SetMetadata(ctx, md)
	}
}

func (k Keeper) GetAllMetadata(ctx sdk.Context) ([]types.ProviderMetadata, error) {
	iter, err := k.providersMetaData.Iterate(ctx, nil)
	if err != nil {
		panic(err)
	}
	return iter.Values()
}
