package keeper

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) SetPairingRelayCache(project string, chainID string, provider string, pairedProviders []epochstoragetypes.StakeEntry) {
	key := types.NewPairingRelayCacheKey(project, chainID, provider)
	k.setPairingCache(key, pairedProviders, k.pairingRelayCache)
}

func (k Keeper) GetPairingRelayCache(project string, chainID string, provider string) ([]epochstoragetypes.StakeEntry, bool) {
	key := types.NewPairingRelayCacheKey(project, chainID, provider)
	return k.getPairingCache(key, k.pairingRelayCache)
}

func (k Keeper) SetPairingQueryCache(project string, chainID string, epoch uint64, pairedProviders []epochstoragetypes.StakeEntry) {
	key := types.NewPairingQueryCacheKey(project, chainID, epoch)
	k.setPairingCache(key, pairedProviders, k.pairingQueryCache)
}

func (k Keeper) GetPairingQueryCache(project string, chainID string, epoch uint64) ([]epochstoragetypes.StakeEntry, bool) {
	key := types.NewPairingQueryCacheKey(project, chainID, epoch)
	return k.getPairingCache(key, k.pairingQueryCache)
}

func (k Keeper) setPairingCache(key string, pairedProviders []epochstoragetypes.StakeEntry, cache *map[string][]epochstoragetypes.StakeEntry) {
	if cache == nil {
		// pairing cache is not initialized, will be in next epoch so simply skip
		return
	}
	(*cache)[key] = pairedProviders
}

func (k Keeper) getPairingCache(key string, cache *map[string][]epochstoragetypes.StakeEntry) ([]epochstoragetypes.StakeEntry, bool) {
	if cache == nil {
		// pairing cache is not initialized, will be in next epoch so simply skip
		return nil, false
	}

	if providers, ok := (*cache)[key]; ok {
		return providers, true
	}

	return nil, false
}
