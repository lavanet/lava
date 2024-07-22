package keeper

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) SetPairingRelayCache(project string, chainID string, epoch uint64, pairedProviders []epochstoragetypes.StakeEntry) {
	k.setPairingCache(project, chainID, epoch, pairedProviders, k.pairingRelayCache)
}

func (k Keeper) GetPairingRelayCache(project string, chainID string, epoch uint64) ([]epochstoragetypes.StakeEntry, bool) {
	return k.getPairingCache(project, chainID, epoch, k.pairingRelayCache)
}

func (k Keeper) SetPairingQueryCache(project string, chainID string, epoch uint64, pairedProviders []epochstoragetypes.StakeEntry) {
	k.setPairingCache(project, chainID, epoch, pairedProviders, k.pairingQueryCache)
}

func (k Keeper) GetPairingQueryCache(project string, chainID string, epoch uint64) ([]epochstoragetypes.StakeEntry, bool) {
	return k.getPairingCache(project, chainID, epoch, k.pairingQueryCache)
}

func (k Keeper) setPairingCache(project string, chainID string, epoch uint64, pairedProviders []epochstoragetypes.StakeEntry, cache *map[types.PairingCacheKey][]epochstoragetypes.StakeEntry) {
	if cache == nil {
		// pairing cache is not initialized, will be in next epoch so simply skip
		return
	}
	(*cache)[types.NewPairingCacheKey(project, chainID, epoch)] = pairedProviders
}

func (k Keeper) getPairingCache(project string, chainID string, epoch uint64, cache *map[types.PairingCacheKey][]epochstoragetypes.StakeEntry) ([]epochstoragetypes.StakeEntry, bool) {
	if cache == nil {
		// pairing cache is not initialized, will be in next epoch so simply skip
		return nil, false
	}

	if providers, ok := (*cache)[types.NewPairingCacheKey(project, chainID, epoch)]; ok {
		return providers, true
	}

	return nil, false
}
