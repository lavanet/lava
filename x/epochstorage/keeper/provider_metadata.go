package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/utils"
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

// GetProviderMetadataByVault gets the provider metadata for a specific vault address
func (k Keeper) GetProviderMetadataByVault(ctx sdk.Context, vault string) (val types.ProviderMetadata, found bool) {
	pk, err := k.providersMetaData.Indexes.Index.MatchExact(ctx, vault)
	if err != nil {
		return types.ProviderMetadata{}, false
	}

	entry, err := k.providersMetaData.Get(ctx, pk)
	if err != nil {
		utils.LavaFormatError("GetProviderMetadataByVault: Get with primary key failed", err,
			utils.LogAttr("vault", vault),
		)
		return types.ProviderMetadata{}, false
	}

	return entry, true
}
