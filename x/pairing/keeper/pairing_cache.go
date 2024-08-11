package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

func (k Keeper) SetPairingRelayCache(ctx sdk.Context, project string, chainID string, provider string, pairedProviders []epochstoragetypes.StakeEntry) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PairingRelayCachePrefix)
	cache := types.PairingRelayCache{Entries: pairedProviders}
	b := k.cdc.MustMarshal(&cache)
	store.Set([]byte(types.NewPairingRelayCacheKey(project, chainID, provider)), b)
}

func (k Keeper) GetPairingRelayCache(ctx sdk.Context, project string, chainID string, provider string) ([]epochstoragetypes.StakeEntry, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PairingRelayCachePrefix)
	b := store.Get([]byte(types.NewPairingRelayCacheKey(project, chainID, provider)))
	if b == nil {
		return []epochstoragetypes.StakeEntry{}, false
	}
	var cache types.PairingRelayCache
	k.cdc.MustUnmarshal(b, &cache)
	return cache.Entries, true
}

// ResetPairingRelayCache is used to remove all entries from the PairingRelayCache KV store
// this function is called in the module's EndBlock so the data written in the KV store
// will be deleted before it's written to the state
func (k Keeper) ResetPairingRelayCache(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PairingRelayCachePrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

func (k Keeper) SetPairingQueryCache(project string, chainID string, epoch uint64, pairedProviders []epochstoragetypes.StakeEntry) {
	if k.pairingQueryCache == nil {
		// pairing cache is not initialized, will be in next epoch so simply skip
		return
	}
	key := types.NewPairingQueryCacheKey(project, chainID, epoch)

	(*k.pairingQueryCache)[key] = pairedProviders
}

func (k Keeper) GetPairingQueryCache(project string, chainID string, epoch uint64) ([]epochstoragetypes.StakeEntry, bool) {
	if k.pairingQueryCache == nil {
		// pairing cache is not initialized, will be in next epoch so simply skip
		return nil, false
	}
	key := types.NewPairingQueryCacheKey(project, chainID, epoch)
	if providers, ok := (*k.pairingQueryCache)[key]; ok {
		return providers, true
	}

	return nil, false
}
