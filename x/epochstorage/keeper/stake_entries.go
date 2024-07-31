package keeper

import (
	"fmt"
	"sort"
	"strconv"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
)

/* ########## StakeEntry ############ */

// GetStakeEntry gets a specific stake entry from the stake entries KV store
// Since the stake entries KV store's key includes the provider's stake (which is not known), we iterate over all
// the providers with the same epoch and chainID and compare the requested address to find the provider
func (k Keeper) GetStakeEntry(ctx sdk.Context, epoch uint64, chainID string, provider string) (types.StakeEntry, bool) {
	pk, err := k.stakeEntries.Indexes.Index.MatchExact(ctx, collections.Join3(epoch, chainID, provider))
	if err != nil {
		utils.LavaFormatWarning("GetStakeEntry: MatchExact with ref key failed", err,
			utils.LogAttr("epoch", epoch),
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("provider", provider),
		)
		return types.StakeEntry{}, false
	}

	entry, err := k.stakeEntries.Get(ctx, pk)
	if err != nil {
		utils.LavaFormatError("GetStakeEntry: Get with primary key failed", err,
			utils.LogAttr("epoch", epoch),
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("provider", provider),
		)
		return types.StakeEntry{}, false
	}

	return entry, true
}

// Set stake entry
func (k Keeper) SetStakeEntry(ctx sdk.Context, epoch uint64, stakeEntry types.StakeEntry) {
	stake := uint64(0)
	if !stakeEntry.Stake.IsNil() && stakeEntry.Stake.Amount.IsUint64() {
		stake = stakeEntry.Stake.Amount.Uint64()
	}
	key := collections.Join3(epoch, stakeEntry.Chain, collections.Join(stake, stakeEntry.Address))
	err := k.stakeEntries.Set(ctx, key, stakeEntry)
	if err != nil {
		panic(fmt.Errorf("SetStakeEntry: Failed to set entry with key %v, error: %w", key, err))
	}
}

// RemoveAllStakeEntriesForEpoch removes all the stake entries of a specific epoch
func (k Keeper) RemoveAllStakeEntriesForEpoch(ctx sdk.Context, epoch uint64) {
	rng := collections.NewPrefixedTripleRange[uint64, string, collections.Pair[uint64, string]](epoch)
	iter, err := k.stakeEntries.Iterate(ctx, rng)
	if err != nil {
		panic(fmt.Errorf("RemoveAllStakeEntriesForEpoch: Failed to create iterator for epoch %d, error: %w", epoch, err))
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			panic(fmt.Errorf("RemoveAllStakeEntriesForEpoch: Failed to get key from iterator for epoch %d, error: %w", epoch, err))
		}

		err = k.stakeEntries.Remove(ctx, key)
		if err != nil {
			panic(fmt.Errorf("RemoveAllStakeEntriesForEpoch: Failed to remove entry with key %v, error: %w", key, err))
		}
	}
}

// GetAllStakeEntries gets all the stake entries of a specific epoch
func (k Keeper) GetAllStakeEntriesForGenesis(ctx sdk.Context) []types.StakeStorage {
	iter, err := k.stakeEntries.Iterate(ctx, nil)
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesForGenesis: Failed to create iterator, error: %w", err))
	}
	defer iter.Close()

	storagesMap := map[uint64]types.StakeStorage{}
	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			panic(fmt.Errorf("GetAllStakeEntriesForGenesis: Failed to get key from iterator, error: %w", err))
		}
		epoch := key.K1()

		entry, err := iter.Value()
		if err != nil {
			panic(fmt.Errorf("GetAllStakeEntriesForGenesis: Failed to get value from iterator for epoch %d, error: %w", epoch, err))
		}
		if _, ok := storagesMap[epoch]; !ok {
			storagesMap[epoch] = types.StakeStorage{
				Index:          strconv.FormatUint(epoch, 10),
				EpochBlockHash: k.GetEpochHash(ctx, epoch),
			}
		}
		storage := storagesMap[epoch]
		storage.StakeEntries = append(storage.StakeEntries, entry)
		storagesMap[epoch] = storage
	}

	var storages []types.StakeStorage
	var keys []uint64
	for key := range storagesMap {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, key := range keys {
		storages = append(storages, storagesMap[key])
	}

	currentEntries := k.GetAllStakeEntriesCurrent(ctx)
	storages = append(storages, types.StakeStorage{Index: "", StakeEntries: currentEntries}) // empty index for genesis

	return storages
}

// GetAllStakeEntriesForEpoch gets all the stake entries of a specific epoch
func (k Keeper) GetAllStakeEntriesForEpoch(ctx sdk.Context, epoch uint64) []types.StakeEntry {
	rng := collections.NewPrefixedTripleRange[uint64, string, collections.Pair[uint64, string]](epoch)
	iter, err := k.stakeEntries.Iterate(ctx, rng)
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesForEpoch: Failed to create iterator for epoch %d, error: %w", epoch, err))
	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesForEpoch: Failed to get entries for epoch %d, error: %w", epoch, err))
	}
	return entries
}

// GetAllStakeEntriesForEpochChainId gets all the stake entries of a specific epoch and a specific chain
func (k Keeper) GetAllStakeEntriesForEpochChainId(ctx sdk.Context, epoch uint64, chainID string) []types.StakeEntry {
	rng := collections.NewSuperPrefixedTripleRange[uint64, string, collections.Pair[uint64, string]](epoch, chainID)
	iter, err := k.stakeEntries.Iterate(ctx, rng)
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesForEpochChainId: Failed to create iterator for epoch %d and chain ID %s, error: %w", epoch, chainID, err))
	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesForEpochChainId: Failed to get entries for epoch %d and chain ID %s, error: %w", epoch, chainID, err))
	}
	return entries
}

/* ########## Current StakeEntry ############ */

// GetStakeEntryCurrent returns a specific current stake entry (with both vault/provider)
func (k Keeper) GetStakeEntryCurrent(ctx sdk.Context, chainID string, provider string) (val types.StakeEntry, found bool) {
	key := collections.Join(chainID, provider)
	entry, err := k.stakeEntriesCurrent.Get(ctx, key)
	if err == nil {
		return entry, true
	}

	// try to find the stake entry by the vault address
	return k.GetStakeEntryCurrentForChainIdByVault(ctx, chainID, provider)
}

// SetStakeEntryCurrent sets a current stake entry in the store
func (k Keeper) SetStakeEntryCurrent(ctx sdk.Context, stakeEntry types.StakeEntry) {
	key := collections.Join(stakeEntry.Chain, stakeEntry.Address)
	err := k.stakeEntriesCurrent.Set(ctx, key, stakeEntry)
	if err != nil {
		panic(fmt.Errorf("SetStakeEntryCurrent: Failed to set entry for key %v, error: %w", key, err))
	}
}

// RemoveStakeEntryCurrent deletes a current stake entry from the store
func (k Keeper) RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, provider string) {
	key := collections.Join(chainID, provider)
	err := k.stakeEntriesCurrent.Remove(ctx, key)
	if err != nil {
		panic(fmt.Errorf("RemoveStakeEntryCurrent: Failed to remove entry with key %v, error: %w", key, err))
	}
}

// GetAllStakeEntriesCurrent gets all the current stake entries
func (k Keeper) GetAllStakeEntriesCurrent(ctx sdk.Context) []types.StakeEntry {
	iter, err := k.stakeEntriesCurrent.Iterate(ctx, nil)
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesCurrent: Failed to create iterator, error: %w", err))
	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesCurrent: Failed to get entries, error: %w", err))
	}
	return entries
}

// GetAllStakeEntriesCurrentForChainId gets all the current stake entries for a specific chain
func (k Keeper) GetAllStakeEntriesCurrentForChainId(ctx sdk.Context, chainID string) []types.StakeEntry {
	rng := collections.NewPrefixedPairRange[string, string](chainID)
	iter, err := k.stakeEntriesCurrent.Iterate(ctx, rng)
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesCurrentForChainId: Failed to create iterator for chain ID %s, error: %w", chainID, err))
	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(fmt.Errorf("GetAllStakeEntriesCurrentForChainId: Failed to get entries for chain ID %s, error: %w", chainID, err))
	}
	return entries
}

// GetStakeEntryCurrentForChainIdByVault gets all the current stake entry for a specific chain by vault address
func (k Keeper) GetStakeEntryCurrentForChainIdByVault(ctx sdk.Context, chainID string, vault string) (val types.StakeEntry, found bool) {
	pk, err := k.stakeEntriesCurrent.Indexes.Index.MatchExact(ctx, collections.Join(chainID, vault))
	if err != nil {
		utils.LavaFormatWarning("GetStakeEntryCurrentForChainIdByVault: MatchExact with primary key failed", err,
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("vault", vault),
		)
		return types.StakeEntry{}, false
	}

	entry, err := k.stakeEntriesCurrent.Get(ctx, pk)
	if err != nil {
		utils.LavaFormatError("GetStakeEntryCurrentForChainIdByVault: Get with primary key failed", err,
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("vault", vault),
		)
		return types.StakeEntry{}, false
	}

	return entry, true
}
