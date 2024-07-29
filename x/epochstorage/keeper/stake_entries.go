package keeper

import (
	"sort"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
)

/* ########## StakeEntry ############ */

// GetStakeEntry gets a specific stake entry from the stake entries KV store
// Since the stake entries KV store's key includes the provider's stake (which is not known), we iterate over all
// the providers with the same epoch and chainID and compare the requested address to find the provider
func (k Keeper) GetStakeEntry(ctx sdk.Context, epoch uint64, chainID string, provider string) (types.StakeEntry, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesPrefix)
	iterator := sdk.KVStorePrefixIterator(store, types.StakeEntryKeyPrefixEpochChainId(epoch, chainID))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var val types.StakeEntry
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		if val.Address == provider {
			return val, true
		}
	}

	return types.StakeEntry{}, false
}

// Set stake entry
func (k Keeper) SetStakeEntry(ctx sdk.Context, epoch uint64, stakeEntry types.StakeEntry) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesPrefix)
	b := k.cdc.MustMarshal(&stakeEntry)
	store.Set(types.StakeEntryKey(epoch, stakeEntry.Chain, stakeEntry.EffectiveStake(), stakeEntry.Address), b)
}

// RemoveAllStakeEntriesForEpoch removes all the stake entries of a specific epoch
func (k Keeper) RemoveAllStakeEntriesForEpoch(ctx sdk.Context, epoch uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesPrefix)
	iterator := sdk.KVStorePrefixIterator(store, utils.SerializeBigEndian(epoch))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

// GetAllStakeEntries gets all the stake entries of a specific epoch
func (k Keeper) GetAllStakeEntriesForGenesis(ctx sdk.Context) []types.StakeStorage {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	storagesMap := map[uint64]types.StakeStorage{}
	for ; iterator.Valid(); iterator.Next() {
		var entry types.StakeEntry
		k.cdc.MustUnmarshal(iterator.Value(), &entry)
		epoch, err := types.ExtractEpochFromStakeEntryKey(string(iterator.Key()))
		if err != nil {
			panic(err)
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
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesPrefix)
	iterator := sdk.KVStorePrefixIterator(store, utils.SerializeBigEndian(epoch))
	defer iterator.Close()

	var entries []types.StakeEntry
	for ; iterator.Valid(); iterator.Next() {
		var entry types.StakeEntry
		k.cdc.MustUnmarshal(iterator.Value(), &entry)
		entries = append(entries, entry)
	}

	return entries
}

// GetAllStakeEntriesForEpochChainId gets all the stake entries of a specific epoch and a specific chain
func (k Keeper) GetAllStakeEntriesForEpochChainId(ctx sdk.Context, epoch uint64, chainID string) []types.StakeEntry {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesPrefix)
	iterator := sdk.KVStoreReversePrefixIterator(store, types.StakeEntryKeyPrefixEpochChainId(epoch, chainID))
	defer iterator.Close()

	var entries []types.StakeEntry
	for ; iterator.Valid(); iterator.Next() {
		var entry types.StakeEntry
		k.cdc.MustUnmarshal(iterator.Value(), &entry)
		entries = append(entries, entry)
	}

	return entries
}

/* ########## Current StakeEntry ############ */

// GetStakeEntryCurrent returns a specific current stake entry (with both vault/provider)
func (k Keeper) GetStakeEntryCurrent(ctx sdk.Context, chainID string, provider string) (val types.StakeEntry, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesCurrentPrefix)
	b := store.Get(types.StakeEntryKeyCurrent(chainID, provider))
	if b != nil {
		k.cdc.MustUnmarshal(b, &val)
		return val, true
	}

	// try to find the stake entry by the vault address. Need to loop because we set provider address in key
	return k.GetStakeEntryCurrentForChainIdByVault(ctx, chainID, provider)
}

// SetStakeEntryCurrent sets a current stake entry in the store
func (k Keeper) SetStakeEntryCurrent(ctx sdk.Context, stakeEntry types.StakeEntry) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesCurrentPrefix)
	b := k.cdc.MustMarshal(&stakeEntry)
	store.Set(types.StakeEntryKeyCurrent(stakeEntry.Chain, stakeEntry.Address), b)
}

// RemoveStakeEntryCurrent deletes a current stake entry from the store
func (k Keeper) RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, provider string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesCurrentPrefix)
	store.Delete(types.StakeEntryKeyCurrent(chainID, provider))
}

// GetAllStakeEntriesCurrent gets all the current stake entries
func (k Keeper) GetAllStakeEntriesCurrent(ctx sdk.Context) []types.StakeEntry {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesCurrentPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var entries []types.StakeEntry
	for ; iterator.Valid(); iterator.Next() {
		var entry types.StakeEntry
		k.cdc.MustUnmarshal(iterator.Value(), &entry)
		entries = append(entries, entry)
	}

	return entries
}

// GetAllStakeEntriesCurrentForChainId gets all the current stake entries for a specific chain
func (k Keeper) GetAllStakeEntriesCurrentForChainId(ctx sdk.Context, chainID string) []types.StakeEntry {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesCurrentPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(chainID))
	defer iterator.Close()

	var entries []types.StakeEntry
	for ; iterator.Valid(); iterator.Next() {
		var entry types.StakeEntry
		k.cdc.MustUnmarshal(iterator.Value(), &entry)
		entries = append(entries, entry)
	}

	return entries
}

// GetStakeEntryCurrentForChainIdByVault gets all the current stake entry for a specific chain by vault address
func (k Keeper) GetStakeEntryCurrentForChainIdByVault(ctx sdk.Context, chainID string, vault string) (val types.StakeEntry, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.StakeEntriesCurrentPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(chainID))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var entry types.StakeEntry
		k.cdc.MustUnmarshal(iterator.Value(), &entry)
		if entry.Vault == vault {
			return entry, true
		}
	}

	return val, false
}
