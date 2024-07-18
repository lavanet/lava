package keeper

import (
	"strconv"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/types"
)

/* ########## StakeEntry ############ */

// GetStakeEntry gets a specific stake entry from the stake entries KV store
// Since the stake entries KV store's key includes the provider's stake (which is not known), we iterate over all
// the providers with the same epoch and chainID and compare the requested address to find the provider
func (k Keeper) GetStakeEntry(ctx sdk.Context, epoch uint64, chainID string, provider string) (types.StakeEntry, bool) {
	rng := collections.NewSuperPrefixedTripleRange[uint64, string, collections.Pair[uint64, string]](epoch, chainID)

	iter, err := k.stakeEntries.Iterate(ctx, rng)
	if err != nil {
		return types.StakeEntry{}, false
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		v, err := iter.Value()
		if err != nil {
			return types.StakeEntry{}, false
		}

		if v.Address == provider {
			return v, true
		}
	}

	return types.StakeEntry{}, false
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
		panic(err)
	}
}

// RemoveAllStakeEntriesForEpoch removes all the stake entries of a specific epoch
func (k Keeper) RemoveAllStakeEntriesForEpoch(ctx sdk.Context, epoch uint64) {
	rng := collections.NewPrefixedTripleRange[uint64, string, collections.Pair[uint64, string]](epoch)
	iter, err := k.stakeEntries.Iterate(ctx, rng)
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			panic(err)
		}

		err = k.stakeEntries.Remove(ctx, key)
		if err != nil {
			panic(err)
		}
	}
}

// GetAllStakeEntries gets all the stake entries of a specific epoch
func (k Keeper) GetAllStakeEntriesForGenesis(ctx sdk.Context) []types.StakeStorage {
	iter, err := k.stakeEntries.Iterate(ctx, nil)
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	storagesMap := map[uint64]types.StakeStorage{}
	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			panic(err)
		}
		epoch := key.K1()

		entry, err := iter.Value()
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
	for _, storage := range storagesMap {
		storages = append(storages, storage)
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
		panic(err)
	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(err)
	}
	return entries
}

// GetAllStakeEntriesForEpochChainId gets all the stake entries of a specific epoch and a specific chain
func (k Keeper) GetAllStakeEntriesForEpochChainId(ctx sdk.Context, epoch uint64, chainID string) []types.StakeEntry {
	rng := collections.NewSuperPrefixedTripleRange[uint64, string, collections.Pair[uint64, string]](epoch, chainID)
	iter, err := k.stakeEntries.Iterate(ctx, rng)
	if err != nil {
		panic(err)

	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(err)
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

	// try to find the stake entry by the vault address. Need to loop because we set provider address in key
	return k.GetStakeEntryCurrentForChainIdByVault(ctx, chainID, provider)
}

// SetStakeEntryCurrent sets a current stake entry in the store
func (k Keeper) SetStakeEntryCurrent(ctx sdk.Context, stakeEntry types.StakeEntry) {
	key := collections.Join(stakeEntry.Chain, stakeEntry.Address)
	err := k.stakeEntriesCurrent.Set(ctx, key, stakeEntry)
	if err != nil {
		panic(err)
	}
}

// RemoveStakeEntryCurrent deletes a current stake entry from the store
func (k Keeper) RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, provider string) {
	key := collections.Join(chainID, provider)
	err := k.stakeEntriesCurrent.Remove(ctx, key)
	if err != nil {
		panic(err)
	}
}

// GetAllStakeEntriesCurrent gets all the current stake entries
func (k Keeper) GetAllStakeEntriesCurrent(ctx sdk.Context) []types.StakeEntry {
	iter, err := k.stakeEntriesCurrent.Iterate(ctx, nil)
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(err)
	}
	return entries
}

// GetAllStakeEntriesCurrentForChainId gets all the current stake entries for a specific chain
func (k Keeper) GetAllStakeEntriesCurrentForChainId(ctx sdk.Context, chainID string) []types.StakeEntry {
	rng := collections.NewPrefixedPairRange[string, string](chainID)
	iter, err := k.stakeEntriesCurrent.Iterate(ctx, rng)
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	entries, err := iter.Values()
	if err != nil {
		panic(err)
	}
	return entries
}

// GetStakeEntryCurrentForChainIdByVault gets all the current stake entry for a specific chain by vault address
func (k Keeper) GetStakeEntryCurrentForChainIdByVault(ctx sdk.Context, chainID string, vault string) (val types.StakeEntry, found bool) {
	rng := collections.NewPrefixedPairRange[string, string](chainID)
	iter, err := k.stakeEntriesCurrent.Iterate(ctx, rng)
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		entry, err := iter.Value()
		if err != nil {
			panic(err)
		}
		if entry.Vault == vault {
			return entry, true
		}
	}

	return types.StakeEntry{}, false
}
