package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

func (k Keeper) SetPairingRelayCache(ctx sdk.Context, project string, chainID string, epoch uint64, pairedProviders []epochstoragetypes.StakeEntry, allowedCu uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PairingRelayCachePrefix)
	cache := types.PairingRelayCache{Entries: pairedProviders, AllowedCu: allowedCu}
	b := k.cdc.MustMarshal(&cache)
	store.Set([]byte(types.NewPairingCacheKey(project, chainID, epoch)), b)
}

func (k Keeper) GetPairingRelayCache(ctx sdk.Context, project string, chainID string, epoch uint64) ([]epochstoragetypes.StakeEntry, uint64, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PairingRelayCachePrefix)
	b := store.Get([]byte(types.NewPairingCacheKey(project, chainID, epoch)))
	if b == nil {
		return []epochstoragetypes.StakeEntry{}, 0, false
	}
	var cache types.PairingRelayCache
	k.cdc.MustUnmarshal(b, &cache)
	return cache.Entries, cache.AllowedCu, true
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
	key := types.NewPairingCacheKey(project, chainID, epoch)

	(*k.pairingQueryCache)[key] = pairedProviders
}

func (k Keeper) GetPairingQueryCache(project string, chainID string, epoch uint64) ([]epochstoragetypes.StakeEntry, bool) {
	if k.pairingQueryCache == nil {
		// pairing cache is not initialized, will be in next epoch so simply skip
		return nil, false
	}
	key := types.NewPairingCacheKey(project, chainID, epoch)
	if providers, ok := (*k.pairingQueryCache)[key]; ok {
		return providers, true
	}

	return nil, false
}
